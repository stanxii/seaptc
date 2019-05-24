package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"

	"github.com/seaptc/server/model"
)

type suggestedSchedule struct {
	Name    string
	Classes []*catalogClass
}

type catalogClass struct {
	Number int
	Length int
	Title  string

	// Meaning of flags depends on context.
	Flag bool
}

type catalogService struct {
	*application

	templates struct {
		Index   *templates.Template `html:"catalog/index.html catalog/root.html"`
		Error   *templates.Template `html:"catalog/error.html catalog/root.html"`
		Program *templates.Template `html:"catalog/program.html catalog/root.html"`
		All     *templates.Template `html:"catalog/all.html catalog/root.html"`
		New     *templates.Template `html:"catalog/new.html catalog/root.html"`
	}

	mu    sync.RWMutex
	pages map[string]*model.Page
}

func (svc *catalogService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	svc.application = a
	svc.pages = make(map[string]*model.Page)

	tm.NewFromFields(&svc.templates)
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			n, err := svc.validatePageCache(context.Background())
			if err != nil {
				log.Printf("catalog cache validation error: %v", err)
			} else {
				log.Printf("catalog cache eviction: %d", n)
			}
		}
	}()
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

var catalogBuilders = map[string]func(*catalogService, *bytes.Buffer, *model.Conference, []*model.Class) error{
	"/catalog/all": (*catalogService).buildAllPage,
	"/catalog/new": (*catalogService).buildNewPage,
	"/catalog/cub": func(svc *catalogService, buf *bytes.Buffer, conf *model.Conference, classes []*model.Class) error {
		return svc.buildProgramPage(buf, conf, classes, model.CubScoutProgram)
	},
	"/catalog/bsa": func(svc *catalogService, buf *bytes.Buffer, conf *model.Conference, classes []*model.Class) error {
		return svc.buildProgramPage(buf, conf, classes, model.ScoutsBSAProgram)
	},
	"/catalog/ven": func(svc *catalogService, buf *bytes.Buffer, conf *model.Conference, classes []*model.Class) error {
		return svc.buildProgramPage(buf, conf, classes, model.VenturingProgram)
	},
	"/catalog/sea": func(svc *catalogService, buf *bytes.Buffer, conf *model.Conference, classes []*model.Class) error {
		return svc.buildProgramPage(buf, conf, classes, model.SeaScoutProgram)
	},
	"/catalog/com": func(svc *catalogService, buf *bytes.Buffer, conf *model.Conference, classes []*model.Class) error {
		return svc.buildProgramPage(buf, conf, classes, model.CommissionerProgram)
	},
	"/catalog/you": func(svc *catalogService, buf *bytes.Buffer, conf *model.Conference, classes []*model.Class) error {
		return svc.buildProgramPage(buf, conf, classes, model.YouthProgram)
	},
}

func (svc *catalogService) Serve_(rc *requestContext) error {
	if rc.request.URL.Path != "/" {
		return httperror.ErrNotFound
	}
	return rc.respond(svc.templates.Index, http.StatusOK, nil)
}

func (svc *catalogService) validatePageCache(ctx context.Context) (int, error) {
	hashes, err := svc.store.GetPageHashes(ctx)
	if err != nil {
		return 0, err
	}

	var n int
	svc.mu.Lock()
	defer svc.mu.Unlock()
	for path, page := range svc.pages {
		if page.Hash != hashes[path] {
			delete(svc.pages, path)
			n++
		}
	}
	return n, nil
}

func (svc *catalogService) Serve_catalog_(rc *requestContext) error {
	path := rc.request.URL.Path
	if _, ok := catalogBuilders[path]; !ok {
		return httperror.ErrNotFound
	}

	if rc.isAdmin {
		n, err := svc.validatePageCache(rc.context())
		if err != nil {
			return err
		}
		if n > 0 {
			rc.logf("evicted %d pages from page cache", n)
		}
	}

	svc.mu.RLock()
	page := svc.pages[path]
	svc.mu.RUnlock()

	if page == nil {
		// Load from the datastore.
		//
		// The request rate for these pages is low enough that it's not worth
		// using x/sync/singleflight or similar to ensure that only one
		// goroutine loads the page from the dataostore.

		rc.logf("Fetching %s from datastore", path)
		var err error
		page, err = svc.store.GetPage(rc.context(), path)
		if err != nil {
			return err
		}

		svc.mu.Lock()
		svc.pages[path] = page
		svc.mu.Unlock()
	}

	etag := fmt.Sprintf(`"%s"`, page.Hash)

	h := rc.response.Header()
	if !svc.devMode {
		h.Set("Cache-Control", "public, max-age=300")
	}
	h.Set("Etag", etag)
	if rc.request.Header.Get("If-None-Match") == etag {
		rc.response.WriteHeader(http.StatusNotModified)
		return nil
	}
	h.Set("Content-Type", page.ContentType)
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

func (svc *catalogService) Serve_dashboard_rebuild__catalog(rc *requestContext) error {
	if rc.request.Method != "POST" {
		return httperror.ErrMethodNotAllowed
	}
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	hashes, err := svc.store.GetPageHashes(rc.context())
	if err != nil {
		return err
	}

	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	classes, err := svc.store.GetAllClassesFull(rc.context())
	if err != nil {
		return err
	}
	model.SortClasses(classes, model.Class_Number)

	var buf, cbuf bytes.Buffer

	var n int
	for path, builder := range catalogBuilders {
		buf.Reset()
		cbuf.Reset()

		err := builder(svc, &buf, conf, classes)
		if err != nil {
			return fmt.Errorf("page %s: %v", path, err)
		}

		hash := md5.Sum(buf.Bytes())

		w := gzip.NewWriter(&cbuf)
		w.Write(buf.Bytes())
		w.Close()

		page := model.Page{
			Path:        path,
			ContentType: "text/html",
			Hash:        fmt.Sprintf("%x", hash[:]),
			Compressed:  true,
			Data:        cbuf.Bytes(),
		}

		err = svc.store.SetPage(rc.context(), &page)
		if err != nil {
			return err
		}

		if page.Hash != hashes[page.Path] {
			n++
		}
	}

	rc.logf("class catalog rebuilt, %d of %d pages changed", n, len(catalogBuilders))
	rc.setFlashMessage("info", "Class catalog rebuilt, %d of %d pages changed.", n, len(catalogBuilders))

	ref := rc.request.FormValue("ref")
	if ref == "" {
		ref = "/dashboard/classes"
	}
	return rc.redirect(ref, http.StatusSeeOther)
}

func (svc *catalogService) buildAllPage(buf *bytes.Buffer, conf *model.Conference, classes []*model.Class) error {
	var data = struct {
		Morning    [][]*catalogClass
		Afternoon  [][]*catalogClass
		Conference *model.Conference
		Classes    []*model.Class
		Key        []*model.ProgramDescription
	}{
		Morning:    catalogGrid(classes, true),
		Afternoon:  catalogGrid(classes, false),
		Conference: conf,
		Classes:    classes,
		Key:        model.ProgramDescriptions,
	}

	return svc.templates.All.Execute(buf, &data)
}

func (svc *catalogService) buildNewPage(buf *bytes.Buffer, conf *model.Conference, classes []*model.Class) error {
	var data = struct {
		Conference *model.Conference
		Classes    []*model.Class
		Key        []*model.ProgramDescription
	}{
		Conference: conf,
		Classes:    classes,
		Key:        model.ProgramDescriptions,
	}
	return svc.templates.New.Execute(buf, &data)
}

func (svc *catalogService) buildProgramPage(buf *bytes.Buffer, conf *model.Conference, classes []*model.Class, program int) error {

	mask := 1 << uint(program)
	var programClasses []*model.Class
	for _, c := range classes {
		if c.Programs&mask != 0 {
			programClasses = append(programClasses, c)
		}
	}

	pd := model.ProgramDescriptions[program]

	var data = struct {
		Title              string
		Program            *model.ProgramDescription
		Conference         *model.Conference
		Classes            []*model.Class
		SuggestedSchedules []*suggestedSchedule
	}{
		Title:      strings.Title(pd.Name),
		Program:    pd,
		Conference: conf,
		Classes:    programClasses,
	}
	return svc.templates.Program.Execute(buf, &data)
}

func catalogGrid(classes []*model.Class, morning bool) [][]*catalogClass {
	// Separate classes into rows.

	rows := make([][]*catalogClass, 100)
	for _, c := range classes {
		start, end := c.StartEnd()
		if start < 0 || end >= model.NumSession {
			continue
		}

		cc := &catalogClass{
			Number: c.Number,
			Length: c.Length,
			Title:  c.Title,
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
