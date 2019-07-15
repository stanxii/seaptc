package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"

	"github.com/seaptc/server/model"
)

// catalogService displays the class catalog. To keep the database read count
// down, the pages are built and cached in the database.
type catalogService struct {
	*application

	templates struct {
		Index   *templates.Template `html:"catalog/index.html catalog/root.html"`
		Error   *templates.Template `html:"catalog/error.html catalog/root.html"`
		Program *templates.Template `html:"catalog/program.html catalog/root.html"`
		All     *templates.Template `html:"catalog/all.html catalog/root.html"`
		New     *templates.Template `html:"catalog/new.html catalog/root.html"`
	}

	mu               sync.RWMutex
	pages            map[string]*model.Page
	lastValidateTime time.Time
}

func (svc *catalogService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	svc.application = a
	svc.pages = make(map[string]*model.Page)
	tm.NewFromFields(&svc.templates)
	return nil
}

func (svc *catalogService) errorTemplate() *templates.Template {
	return svc.templates.Error
}

func (svc *catalogService) makeHandler(v interface{}) func(*requestContext) error {
	f, ok := v.(func(*catalogService, *requestContext) error)
	if !ok {
		return nil
	}
	return func(rc *requestContext) error { return f(svc, rc) }
}

func (svc *catalogService) Serve_catalog_(rc *requestContext) error {
	// We don't have enough traffic to bother with single flighting the
	// datastore access in this function.

	if svc.devMode {
		rc.request.ParseForm()
		if _, ok := rc.request.Form["build"]; ok {
			changed, total, err := svc.buildCatalog(rc.ctx)
			if err != nil {
				return err
			}
			rc.logf("class catalog rebuilt, %d of %d pages changed", changed, total)
		}
	}

	path := rc.request.URL.Path

	svc.mu.RLock()
	validateCache := rc.isAdmin || time.Since(svc.lastValidateTime) > 10*time.Minute
	page, found := svc.pages[path]
	svc.mu.RUnlock()

	if validateCache {
		// Purge stale pages from the in-memory cache. Ensure that all pages
		// have a cache entry (possibly a nil tombstone) for quick detection of
		// not found errors.
		hashes, err := svc.store.GetPageHashes(rc.ctx)
		if err != nil {
			return err
		}
		svc.mu.Lock()
		svc.lastValidateTime = time.Now()
		for path, hash := range hashes {
			if page := svc.pages[path]; page == nil || page.Hash != hash {
				svc.pages[path] = nil // nil is tombstone for handling not found errors
			}
		}
		page, found = svc.pages[path]
		svc.mu.Unlock()
	}

	if !found {
		return httperror.ErrNotFound
	}

	if page == nil {
		rc.logf("Fetching %s from datastore", path)
		var err error
		page, err = svc.store.GetPage(rc.ctx, path)
		if err != nil {
			return err
		}

		svc.mu.Lock()
		svc.pages[path] = page
		svc.mu.Unlock()
	}

	etag := fmt.Sprintf(`"%s"`, page.Hash)

	h := rc.response.Header()
	if !svc.devMode && !rc.isAdmin {
		h.Set("Cache-Control", "public, max-age=300")
	}
	h.Set("Etag", etag)
	if rc.request.Header.Get("If-None-Match") == etag {
		rc.response.WriteHeader(http.StatusNotModified)
		return nil
	}
	h.Set("Content-Type", page.ContentType)
	h.Set("Content-Length", strconv.Itoa(len(page.Data)))
	if page.Compressed {
		h.Set("Content-Encoding", "gzip")
	}
	if rc.request.Method != "HEAD" {
		rc.response.Write(page.Data)
	}
	return nil
}

func (svc *catalogService) Serve_dashboard_rebuildCatalog(rc *requestContext) error {
	if rc.request.Method != "POST" {
		return httperror.ErrMethodNotAllowed
	}
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}
	changed, total, err := svc.buildCatalog(rc.ctx)
	if err != nil {
		return err
	}

	return rc.redirect("/dashboard/admin", "info", "Class catalog rebuilt, %d of %d pages changed.", changed, total)
}

func (svc *catalogService) buildCatalog(ctx context.Context) (int, int, error) {
	suggestedSchedules, err := svc.store.GetSuggestedSchedules(ctx)
	if err != nil {
		return 0, 0, err
	}

	hashes, err := svc.store.GetPageHashes(ctx)
	if err != nil {
		return 0, 0, err
	}

	conf, err := svc.store.GetConference(ctx)
	if err != nil {
		return 0, 0, err
	}

	classes, err := svc.store.GetAllClassesFull(ctx)
	if err != nil {
		return 0, 0, err
	}

	model.SortClasses(classes, "number")
	catalogSuggestedSchedules := createCatalogSuggestedSchedules(classes, suggestedSchedules)

	var data = struct {
		Morning            [][]*catalogClass
		Afternoon          [][]*catalogClass
		Conference         *model.Conference
		Date               time.Time
		Classes            []*model.Class
		Key                []*model.ProgramDescription
		SuggestedSchedules []*catalogSuggestedSchedule
		Program            *model.ProgramDescription
		Title              string
	}{
		Morning:    createCatalogGrid(classes, true),
		Afternoon:  createCatalogGrid(classes, false),
		Key:        model.ProgramDescriptions,
		Conference: conf,
		Date:       svc.conferenceDate,
		Classes:    classes,
	}

	pageInfos := []struct {
		path     string
		template *templates.Template
		program  int
	}{
		{"/catalog/", svc.templates.All, -1},
		{"/catalog/new", svc.templates.New, -1},
		{"/catalog/cub", svc.templates.Program, model.CubScoutProgram},
		{"/catalog/bsa", svc.templates.Program, model.ScoutsBSAProgram},
		{"/catalog/ven", svc.templates.Program, model.VenturingProgram},
		{"/catalog/sea", svc.templates.Program, model.SeaScoutProgram},
		{"/catalog/sea", svc.templates.Program, model.SeaScoutProgram},
		{"/catalog/com", svc.templates.Program, model.CommissionerProgram},
		{"/catalog/you", svc.templates.Program, model.YouthProgram},
	}

	var buf, cbuf bytes.Buffer
	var n int

	for _, pageInfo := range pageInfos {
		if pageInfo.program >= 0 {
			data.Program = model.ProgramDescriptions[pageInfo.program]
			data.Title = strings.Title(data.Program.Name)
			data.SuggestedSchedules = catalogSuggestedSchedules[pageInfo.program]
			data.Classes = nil
			mask := 1 << uint(pageInfo.program)
			for _, c := range classes {
				if c.Programs&mask != 0 {
					data.Classes = append(data.Classes, c)
				}
			}
		}

		buf.Reset()
		err := pageInfo.template.Execute(&buf, &data)
		if err != nil {
			return 0, 0, fmt.Errorf("page %s: %v", pageInfo.path, err)
		}

		hash := md5.Sum(buf.Bytes())

		cbuf.Reset()
		w := gzip.NewWriter(&cbuf)
		w.Write(buf.Bytes())
		w.Close()

		page := model.Page{
			Path:        pageInfo.path,
			ContentType: "text/html",
			Hash:        fmt.Sprintf("%x", hash[:]),
			Compressed:  true,
			Data:        cbuf.Bytes(),
		}

		err = svc.store.SetPage(ctx, &page)
		if err != nil {
			return 0, 0, err
		}

		if page.Hash != hashes[page.Path] {
			n++
		}
	}
	return n, len(pageInfos), nil
}

type catalogClass struct {
	Number    int
	Length    int
	Title     string
	TitleNote string
	Flag      bool
}

type catalogSuggestedSchedule struct {
	Name    string
	Classes []*catalogClass
}

func createCatalogSuggestedSchedules(classes []*model.Class, suggestedSchedules []*model.SuggestedSchedule) map[int][]*catalogSuggestedSchedule {
	m := make(map[int]*model.Class)
	for _, c := range classes {
		m[c.Number] = c
	}

	result := make(map[int][]*catalogSuggestedSchedule)
	for _, ss := range suggestedSchedules {
		css := catalogSuggestedSchedule{Name: ss.Name}
		for _, sc := range ss.Classes {
			cc := catalogClass{Number: sc.Number, Length: 1, Flag: sc.Elective, Title: "MISSING"}
			if c := m[sc.Number]; c != nil {
				cc.Length = c.Length
				cc.Title = c.Title
				cc.TitleNote = c.TitleNote
			}
			css.Classes = append(css.Classes, &cc)
		}
		result[ss.Program] = append(result[ss.Program], &css)
	}
	return result
}

func createCatalogGrid(classes []*model.Class, morning bool) [][]*catalogClass {
	// Separate classes into rows.

	rows := make([][]*catalogClass, 100)
	for _, c := range classes {
		start, end := c.StartEnd()
		if start < 0 || end >= model.NumSession {
			continue
		}

		cc := &catalogClass{
			Number:    c.Number,
			Length:    c.Length,
			Title:     c.Title,
			TitleNote: c.TitleNote,
		}

		i := c.Number % 100
		row := rows[i]
		if row == nil {
			row = make([]*catalogClass, 3)
			rows[i] = row
		}

		if morning {
			if start > 2 {
				continue
			}
			if end > 2 {
				cc.Length = 3 - start
				cc.Flag = true
			}
			row[start] = cc
		} else {
			if end < 3 {
				continue
			}
			if start < 3 {
				cc.Length = end - 2
				cc.Flag = true
				start = 3
			}
			row[start-3] = cc
		}
	}

	// Remove unused rows, add dummy classes

	noclass := &catalogClass{Length: 1}
	i := 0
	for _, row := range rows {
		if row == nil {
			continue
		}
		rows[i] = row
		i++

		for j := 0; j < len(row); {
			if row[j] != nil {
				j += row[j].Length
				continue
			}
			row[j] = noclass
			j++
		}
	}
	return rows[:i]
}
