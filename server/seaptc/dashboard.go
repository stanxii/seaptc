package main

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
	"golang.org/x/sync/errgroup"
	"rsc.io/qr"

	"github.com/seaptc/server/dk"
	"github.com/seaptc/server/model"
	"github.com/seaptc/server/sheet"
	"github.com/seaptc/server/store"
)

type dashboardService struct {
	*application
	templates struct {
		Admin        *templates.Template `html:"dashboard/admin.html dashboard/root.html common.html"`
		Class        *templates.Template `html:"dashboard/class.html dashboard/root.html common.html"`
		Classes      *templates.Template `html:"dashboard/classes.html dashboard/root.html common.html"`
		Conference   *templates.Template `html:"dashboard/conference.html dashboard/root.html common.html"`
		Error        *templates.Template `html:"dashboard/error.html dashboard/root.html common.html"`
		Index        *templates.Template `html:"dashboard/index.html dashboard/root.html common.html"`
		Instructors  *templates.Template `html:"dashboard/instructors.html dashboard/root.html common.html"`
		LunchCount   *templates.Template `html:"dashboard/lunchCount.html dashboard/root.html common.html"`
		LunchList    *templates.Template `html:"dashboard/lunchList.html dashboard/root.html common.html"`
		Participant  *templates.Template `html:"dashboard/participant.html dashboard/root.html common.html"`
		Participants *templates.Template `html:"dashboard/participants.html dashboard/root.html common.html"`
		Reprint      *templates.Template `html:"dashboard/reprint.html dashboard/root.html common.html"`

		LunchStickers *templates.Template `html:"dashboard/lunchStickers.html"`
		Form          *templates.Template `html:"dashboard/form.html blurbs.html"`
	}
}

func (svc *dashboardService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	svc.application = a
	tm.NewFromFields(&svc.templates)
	return nil
}

func (svc *dashboardService) errorTemplate() *templates.Template {
	return svc.templates.Error
}

func (svc *dashboardService) makeHandler(v interface{}) func(*requestContext) error {
	f, ok := v.(func(*dashboardService, *requestContext) error)
	if !ok {
		return nil
	}
	return func(rc *requestContext) error { return f(svc, rc) }
}

func (svc *dashboardService) getClass(rc *requestContext, pathPrefix string) (*model.Class, error) {
	number, err := strconv.Atoi(strings.TrimPrefix(rc.request.URL.Path, pathPrefix))
	if err != nil {
		return nil, httperror.ErrNotFound
	}
	class, err := svc.store.GetClass(rc.context(), number)
	if err == store.ErrNotFound {
		err = httperror.ErrNotFound
	}
	return class, err
}

func (svc *dashboardService) Serve_dashboard_(rc *requestContext) error {
	// Use dashboard error template for all not found requests to the /dashboard/ tree.
	return httperror.ErrNotFound
}

func (svc *dashboardService) Serve_dashboard(rc *requestContext) error {
	participants, err := svc.store.GetAllParticipants(rc.context())
	if err != nil {
		return err
	}

	data := struct {
		Councils  map[string]int
		Districts map[string]map[string]int
		Types     map[string]int
		Total     int
	}{
		make(map[string]int),
		make(map[string]map[string]int),
		make(map[string]int),
		0,
	}

	for _, p := range participants {
		data.Total++
		data.Councils[p.Council]++
		data.Types[p.Type()]++
		if p.Council == "Chief Seattle" {
			d := data.Districts[p.District]
			if d == nil {
				d = make(map[string]int)
				data.Districts[p.District] = d
			}
			unitName := p.Unit()
			d[unitName]++
			d[""]++
		}
	}

	return rc.respond(svc.templates.Index, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_classes(rc *requestContext) error {

	var (
		g          errgroup.Group
		classes    []*model.Class
		registered map[int]int
		conf       *model.Conference
	)

	g.Go(func() error {
		var err error
		classes, err = svc.store.GetAllClasses(rc.context())
		return err
	})

	g.Go(func() error {
		var err error
		registered, err = svc.store.GetClassParticipantCounts(rc.context())
		return err
	})

	g.Go(func() error {
		var err error
		conf, err = svc.store.GetConference(rc.context())
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	model.SortClasses(classes, rc.request.FormValue("sort"))
	switch sortKey, reverse := model.SortKeyReverse(rc.request.FormValue("sort")); sortKey {
	case "registered":
		sort.SliceStable(classes, reverse(func(i, j int) bool {
			return registered[classes[i].Number] < registered[classes[j].Number]
		}))
	case "available":
		sort.SliceStable(classes, reverse(func(i, j int) bool {
			m := classes[i].Capacity - registered[classes[i].Number]
			if classes[i].Capacity == 0 {
				m = 9999
			}
			n := classes[j].Capacity - registered[classes[j].Number]
			if classes[j].Capacity == 0 {
				n = 9999
			}
			return m < n
		}))
	}

	var data = struct {
		Classes    []*model.Class
		Lunch      interface{}
		Registered interface{}
		Available  interface{}
	}{
		Classes: classes,
		Lunch:   conf.ClassLunch,
		Registered: func(c *model.Class) string {
			n := registered[c.Number]
			if n == 0 {
				return ""
			}
			return strconv.Itoa(n)
		},
		Available: func(c *model.Class) string {
			if c.Capacity == 0 {
				return ""
			}
			return strconv.Itoa(c.Capacity - registered[c.Number])
		},
	}
	return rc.respond(svc.templates.Classes, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_classes_(rc *requestContext) error {
	var (
		g     errgroup.Group
		class *model.Class
		conf  *model.Conference
	)

	g.Go(func() error {
		var err error
		class, err = svc.getClass(rc, "/dashboard/classes/")
		return err
	})

	g.Go(func() error {
		var err error
		conf, err = svc.store.GetConference(rc.context())
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	var data = struct {
		InstructorView    bool
		Class             *model.Class
		Participants      []*model.Participant
		ParticipantEmails []string
		InstructorURL     string
		Lunch             *model.Lunch
	}{
		Class:          class,
		InstructorView: rc.isStaff,
		Lunch:          conf.ClassLunch(class),
	}

	if len(data.Class.AccessToken) >= 4 {
		protocol := "https"
		if svc.devMode {
			protocol = "http"
		}
		data.InstructorURL = fmt.Sprintf("%s://%s/dashboard/classes/%d?t=%s", protocol, rc.request.Host, class.Number, data.Class.AccessToken)
		if rc.request.FormValue("t") == data.Class.AccessToken {
			data.InstructorView = true
		}
	}

	if data.InstructorView {
		var err error
		data.Participants, err = svc.store.GetClassParticipants(rc.context(), class.Number)
		if err != nil {
			return err
		}
		model.SortParticipants(data.Participants, rc.request.FormValue("sort"))
		for _, p := range data.Participants {
			data.ParticipantEmails = append(data.ParticipantEmails, p.Emails()...)
		}
		sort.Strings(data.ParticipantEmails)
		// Deduplicate
		i := 0
		prev := ""
		for _, e := range data.ParticipantEmails {
			if e != prev {
				prev = e
				data.ParticipantEmails[i] = e
				i++
			}
		}
		data.ParticipantEmails = data.ParticipantEmails[:i]
	}

	return rc.respond(svc.templates.Class, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_participants(rc *requestContext) error {
	participants, err := svc.store.GetAllParticipants(rc.context())
	if err != nil {
		return err
	}

	classMaps, err := svc.store.GetCachedClassMaps(rc.context())
	if err != nil {
		return err
	}

	model.SortParticipants(participants, rc.request.FormValue("sort"))

	var data = struct {
		Participants   []*model.Participant
		SessionClasses interface{}
	}{
		participants,
		classMaps.ParticipantSessionClasses,
	}
	return rc.respond(svc.templates.Participants, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_participants_(rc *requestContext) error {
	if !rc.isStaff {
		return httperror.ErrForbidden
	}

	id := strings.TrimPrefix(rc.request.URL.Path, "/dashboard/participants/")

	conf, err := svc.store.GetCachedConference(rc.context())
	if err != nil {
		return err
	}

	classMaps, err := svc.store.GetCachedClassMaps(rc.context())
	if err != nil {
		return err
	}

	participant, err := svc.store.GetParticipant(rc.context(), id)
	if err == store.ErrNotFound {
		return httperror.ErrNotFound
	} else if err != nil {
		return err
	}

	var data = struct {
		Participant    *model.Participant
		Conference     *model.Conference
		SessionClasses []*model.SessionClass
		Lunch          *model.Lunch
	}{
		participant,
		conf,
		classMaps.ParticipantSessionClasses(participant),
		conf.ParticipantLunch(participant),
	}
	return rc.respond(svc.templates.Participant, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_uploadRegistrations(rc *requestContext) error {
	if rc.request.Method != "POST" {
		return httperror.ErrMethodNotAllowed
	}
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	f, _, err := rc.request.FormFile("file")
	if err == http.ErrMissingFile {
		return &httperror.Error{Status: 400, Message: "Export file not uploaded"}
	}
	if err != nil {
		return err
	}
	defer f.Close()

	participants, err := dk.ParseCSV(f)
	if err != nil {
		return err
	}

	summary, err := svc.store.ImportParticipants(rc.context(), participants)
	if err != nil {
		return err
	}

	return rc.redirect("/dashboard/admin", "info", "Import %d records; %s", len(participants), summary)
}

func (svc *dashboardService) Serve_dashboard_refreshClasses(rc *requestContext) error {
	if rc.request.Method != "POST" {
		return httperror.ErrMethodNotAllowed
	}
	if !rc.isStaff {
		return httperror.ErrForbidden
	}

	var (
		g                  errgroup.Group
		classes            []*model.Class
		suggestedSchedules []*model.SuggestedSchedule
	)

	g.Go(func() error {
		var err error
		classes, err = sheet.GetClasses(rc.context(), svc.config)
		return err
	})

	g.Go(func() error {
		var err error
		suggestedSchedules, err = sheet.GetSuggestedSchedules(rc.context(), svc.config)
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	n, err := svc.store.ImportClasses(rc.context(), classes)
	if err != nil {
		return err
	}

	err = svc.store.SetSuggestedSchedules(rc.context(), suggestedSchedules)
	if err != nil {
		return err
	}

	return rc.redirect("/dashboard/classes", "info", "%d classes loaded from sheet, %d modified", len(classes), n)
}

func (svc *dashboardService) Serve_dashboard_conference(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}
	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	rc.request.ParseForm()
	data := struct {
		Form       url.Values
		Invalid    map[string]string
		Conference *model.Conference
		Programs   []*model.ProgramDescription
		Lunches    string
	}{
		Form:       rc.request.Form,
		Invalid:    make(map[string]string),
		Conference: conf,
		Programs:   model.ProgramDescriptions,
	}

	if rc.request.Method != "POST" {
		p, _ := json.MarshalIndent(conf.Lunches, "", "  ")
		data.Form.Set("lunches", string(p))
		return rc.respond(svc.templates.Conference, http.StatusOK, &data)
	}

	if err := json.Unmarshal([]byte(data.Form.Get("lunches")), &conf.Lunches); err != nil {
		data.Invalid["lunches"] = err.Error()
	}

	conf.RegistrationURL = data.Form.Get("registrationURL")
	conf.CatalogStatusMessage = data.Form.Get("catalogStatusMessage")
	conf.NoClassDescription = data.Form.Get("noClassDescription")
	conf.OABanquetDescription = data.Form.Get("oaBanquetDescription")
	conf.OpeningLocation = data.Form.Get("openingLocation")
	conf.OABanquetLocation = data.Form.Get("oaBanquetLocation")

	conf.StaffIDs = data.Form.Get("staffIDs")
	if len(data.Invalid) > 0 {
		return rc.respond(svc.templates.Conference, http.StatusOK, &data)
	}

	err = svc.store.SetConference(rc.context(), conf)
	if err != nil {
		return err
	}

	return rc.redirect(rc.request.URL.Path, "info", "Conference updated.")
}

func (svc *dashboardService) Serve_dashboard_instructors(rc *requestContext) error {
	if !rc.isStaff {
		return httperror.ErrForbidden
	}
	data := struct {
	}{}
	return rc.respond(svc.templates.Instructors, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_lunchCount(rc *requestContext) error {

	var (
		g            errgroup.Group
		participants []*model.Participant
		conf         *model.Conference
	)

	g.Go(func() error {
		var err error
		participants, err = svc.store.GetAllParticipants(rc.context())
		return err
	})

	g.Go(func() error {
		var err error
		conf, err = svc.store.GetConference(rc.context())
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	data := struct {
		Lunch       map[*model.Lunch]int
		Restriction map[string]int
		Count       map[string]int
		Total       int
	}{
		make(map[*model.Lunch]int),
		make(map[string]int),
		make(map[string]int),
		0,
	}
	for _, p := range participants {
		lunch := conf.ParticipantLunch(p)
		data.Lunch[lunch]++
		data.Restriction[p.DietaryRestrictions]++
		data.Count[fmt.Sprintf("%d:%s", lunch, p.DietaryRestrictions)]++
		data.Total++
	}
	return rc.respond(svc.templates.LunchCount, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_lunchList(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	var (
		g            errgroup.Group
		participants []*model.Participant
		conf         *model.Conference
	)

	g.Go(func() error {
		var err error
		participants, err = svc.store.GetAllParticipants(rc.context())
		return err
	})

	g.Go(func() error {
		var err error
		conf, err = svc.store.GetConference(rc.context())
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	participants = model.FilterParticipants(participants, func(p *model.Participant) bool { return p.DietaryRestrictions != "" })
	model.SortParticipants(participants, "")

	data := make(map[*model.Lunch][]*model.Participant)
	for _, p := range participants {
		l := conf.ParticipantLunch(p)
		data[l] = append(data[l], p)
	}
	return rc.respond(svc.templates.LunchList, http.StatusOK, data)
}

func (svc *dashboardService) Serve_dashboard_lunchStickers(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	var (
		g            errgroup.Group
		participants []*model.Participant
		conf         *model.Conference
	)

	g.Go(func() error {
		var err error
		participants, err = svc.store.GetAllParticipants(rc.context())
		return err
	})

	g.Go(func() error {
		var err error
		conf, err = svc.store.GetConference(rc.context())
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	participants = model.FilterParticipants(participants, func(p *model.Participant) bool { return p.DietaryRestrictions != "" })

	sort.Slice(participants, func(i, j int) bool {
		a := participants[i]
		b := participants[j]
		alunch := conf.ParticipantLunch(a)
		blunch := conf.ParticipantLunch(b)
		switch {
		case alunch.Name < blunch.Name:
			return true
		case alunch.Name > blunch.Name:
			return false
		case a.DietaryRestrictions < b.DietaryRestrictions:
			return true
		case a.DietaryRestrictions > b.DietaryRestrictions:
			return false
		default:
			return model.DefaultParticipantLess(a, b)
		}
	})

	iv := func(name string, def int) int {
		v, _ := strconv.Atoi(rc.request.FormValue(name))
		if v <= 0 {
			return def
		}
		return v
	}
	sv := func(name string, def string) string {
		v := rc.request.FormValue(name)
		if v == "" {
			return def
		}
		return v
	}

	var data = struct {
		Rows    int
		Columns int
		Top     string
		Left    string
		Width   string
		Height  string
		Gutter  string
		Font    string
		Pages   [][][]*model.Participant
		Lunch   interface{}
	}{
		iv("rows", 7),
		iv("columns", 2),
		sv("top", "0.8in"),
		sv("left", "0in"),
		sv("width", "4.25in"),
		sv("height", "1.325in"),
		sv("gutter", "0.in"),
		sv("font", "16pt"),
		nil,
		conf.ParticipantLunch,
	}
	for len(participants) > 0 {
		var page [][]*model.Participant
		for i := 0; i < data.Rows && len(participants) > 0; i++ {
			n := len(participants)
			if n > data.Columns {
				n = data.Columns
			}
			page = append(page, participants[:n])
			participants = participants[n:]
		}
		data.Pages = append(data.Pages, page)
	}
	return rc.respond(svc.templates.LunchStickers, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_admin(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	conf, err := svc.store.GetCachedConference(rc.context())
	if err != nil {
		return err
	}

	data := struct {
		DevMode    bool
		Conference *model.Conference
	}{
		DevMode:    svc.devMode,
		Conference: conf,
	}
	return rc.respond(svc.templates.Admin, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_reprintForms(rc *requestContext) error {
	if !rc.isStaff {
		return httperror.ErrForbidden
	}
	if rc.request.Method == "POST" {
		rc.request.ParseForm()
		ids := rc.request.Form["id"]
		n, err := svc.store.SetParticipantsPrintForm(rc.context(), ids, true)
		if err != nil {
			return err
		}
		return rc.redirect("/dashboard/admin", "info", "%d participants selected, %d reprints queued.", len(ids), n)
	}
	participants, err := svc.store.GetAllParticipants(rc.context())
	if err != nil {
		return err
	}
	model.SortParticipants(participants, "")
	data := struct {
		Participants []*model.Participant
	}{
		participants,
	}
	return rc.respond(svc.templates.Reprint, http.StatusOK, &data)
}

var formOptions = map[string]*struct {
	filter bool
	auto   int
	limit  int
	sort   func([]*model.Participant)
}{
	"auto": {
		filter: true,
		auto:   60,
		limit:  50,
		sort: func(participants []*model.Participant) {
			sort.Slice(participants, func(i, j int) bool {
				return model.DefaultParticipantLess(participants[j], participants[i])
			})
		}},
	"batch": {
		filter: true,
		limit:  50,
		sort: func(participants []*model.Participant) {
			// Sort by staff role and name
			sort.Slice(participants, func(i, j int) bool {
				switch {
				case participants[j].StaffRole < participants[i].StaffRole:
					return true
				case participants[j].StaffRole > participants[i].StaffRole:
					return false
				default:
					return model.DefaultParticipantLess(participants[j], participants[i])
				}
			})
		}},
	"first": {
		sort: func(participants []*model.Participant) {
			// Descending by length of first name.
			sort.Slice(participants, func(i, j int) bool {
				a := len(participants[j].FirstName)
				b := len(participants[i].FirstName)
				switch {
				case a < b:
					return true
				case a > b:
					return false
				default:
					return model.DefaultParticipantLess(participants[i], participants[j])
				}
			})
		}},
	"last": {
		sort: func(participants []*model.Participant) {
			// Descending by length of last name.
			sort.Slice(participants, func(i, j int) bool {
				a := len(participants[j].LastName)
				if participants[i].Suffix != "" {
					a += 1 + len(participants[i].Suffix)
				}
				b := len(participants[i].LastName)
				if participants[j].Suffix != "" {
					a += 1 + len(participants[j].Suffix)
				}
				switch {
				case a < b:
					return true
				case a > b:
					return false
				default:
					return model.DefaultParticipantLess(participants[i], participants[j])
				}
			})
		}},
}

func (svc *dashboardService) Serve_dashboard_forms(rc *requestContext) error {
	if !rc.isStaff {
		return httperror.ErrForbidden
	}

	if rc.request.Method == "POST" {
		rc.request.ParseForm()
		ids := rc.request.Form["id"]
		_, err := svc.store.SetParticipantsPrintForm(rc.context(), ids, false)
		if err != nil {
			return err
		}
	}

	options := formOptions[rc.request.FormValue("options")]
	if options == nil {
		options = formOptions["batch"]
	}

	participants, err := svc.store.GetAllParticipants(rc.context())
	if err != nil {
		return err
	}

	options.sort(participants)

	if options.filter {
		participants = model.FilterParticipants(participants, func(p *model.Participant) bool { return p.PrintForm })
	}

	if options.limit > 0 && len(participants) > options.limit {
		participants = participants[:options.limit]
	}

	ids := make([]string, len(participants))
	for i := range participants {
		ids[i] = participants[i].ID
	}

	return svc.renderForms(rc, options.auto, !options.filter, ids)
}

func (svc *dashboardService) Serve_dashboard_forms_(rc *requestContext) error {
	if !rc.isStaff {
		return httperror.ErrForbidden
	}
	id := strings.TrimPrefix(rc.request.URL.Path, "/dashboard/forms/")
	if id == "" {
		return httperror.ErrNotFound
	}
	return svc.renderForms(rc, 0, true, []string{id})
}

func (svc *dashboardService) renderForms(rc *requestContext, auto int, preview bool, ids []string) error {

	var data = struct {
		Participants   []*model.Participant
		Conference     *model.Conference
		Lunch          interface{}
		SessionClasses interface{}
		Auto           int
		Preview        bool
	}{
		Auto:    auto,
		Preview: preview,
	}

	if len(ids) > 0 {
		var g errgroup.Group

		g.Go(func() error {
			var err error
			data.Participants, err = svc.store.GetParticipantsByID(rc.context(), ids)
			return err
		})

		g.Go(func() error {
			conf, err := svc.store.GetConference(rc.context())
			data.Conference = conf
			data.Lunch = conf.ParticipantLunch
			return err
		})

		g.Go(func() error {
			classes, err := svc.store.GetAllClasses(rc.context())
			classMaps := model.NewClassMaps(classes)
			data.SessionClasses = classMaps.ParticipantSessionClasses
			return err
		})

		if err := g.Wait(); err != nil {
			return err
		}
	}

	return rc.respond(svc.templates.Form, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_vcard(rc *requestContext) error {
	rc.request.ParseForm()
	vcard := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\n")
	for name, values := range rc.request.Form {
		value := strings.TrimSpace(values[0])
		if value == "" {
			continue
		}
		vcard = append(vcard, name...)
		vcard = append(vcard, ':')
		for i := range value {
			b := value[i]
			switch b {
			case '\\':
				vcard = append(vcard, `\\`...)
			case '\n':
				vcard = append(vcard, `\n`...)
			case '\r':
				vcard = append(vcard, `\r`...)
			case ',':
				vcard = append(vcard, `\,`...)
			case ':':
				vcard = append(vcard, `\:`...)
			case ';':
				vcard = append(vcard, `\;`...)
			default:
				vcard = append(vcard, b)
			}
		}
		vcard = append(vcard, "\r\n"...)
	}
	vcard = append(vcard, "END:VCARD\r\n"...)
	code, err := qr.Encode(string(vcard), qr.L)
	if err != nil {
		return err
	}
	rc.response.Header().Set("Content-Type", "image/png")
	rc.response.Write(code.PNG())
	return nil
}

func (svc *dashboardService) Serve_dashboard_exportClasses(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	classes, err := svc.store.GetAllClassesFull(rc.context())
	if err != nil {
		return err
	}

	rc.response.Header().Set("Content-Type", "text/csv")
	rc.response.Header().Set("Content-Disposition", `attachment; filename="classes.csv"`)

	w := csv.NewWriter(rc.response)
	w.Write([]string{
		"Number",
		"Length",
		"Title",
		"Instructors",
		"InstructorEmails",
	})

	for _, c := range classes {
		w.Write([]string{
			fmt.Sprintf("%d", c.Number),
			fmt.Sprintf("%d", c.Length),
			c.Title,
			strings.Join(c.InstructorNames, ", "),
			strings.Join(c.InstructorEmails, ", "),
		})
	}
	w.Flush()
	return nil
}

func (svc *dashboardService) Serve_dashboard_exportParticipants(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	participants, err := svc.store.GetAllParticipantsFull(rc.context())
	if err != nil {
		return err
	}

	classMaps, err := svc.store.GetCachedClassMaps(rc.context())
	if err != nil {
		return err
	}

	rc.response.Header().Set("Content-Type", "text/csv")
	rc.response.Header().Set("Content-Disposition", `attachment; filename="participants.csv"`)

	w := csv.NewWriter(rc.response)
	record := []string{
		"ID",
		"Reg#",
		"Name",
		"Type",
		"Email",
		"Email2",
		"Phone",
		"City",
		"State",
		"Zip",
		"StaffRole",
		"Council",
		"District",
		"Unit",
		"Marketing",
		"ScoutingYears",
		"BSA#",
		"Banquet",
		"No Show",
		"Staff Notes",
	}
	for i := 0; i < model.NumSession; i++ {
		record = append(record, fmt.Sprintf("class_%d", i+1), fmt.Sprintf("instr_%d", i+1))
	}
	w.Write(record)

	for _, p := range participants {
		sessionClasses := classMaps.ParticipantSessionClasses(p)

		var email2 string
		if p.Youth && p.RegisteredByEmail != p.Email {
			email2 = p.RegisteredByEmail
		}

		record := []string{
			p.ID,
			p.RegistrationNumber,
			p.Name(),
			p.Type(),
			p.Email,
			email2,
			p.Phone,
			p.City,
			p.State,
			p.Zip,
			p.StaffRole,
			p.Council,
			p.District,
			p.Unit(),
			p.Marketing,
			p.ScoutingYears,
			p.BSANumber,
			strconv.FormatBool(p.OABanquet),
			strconv.FormatBool(p.NoShow),
			strings.Replace(p.Notes, "\n", " ", -1),
		}
		for _, c := range sessionClasses {
			record = append(record, c.NumberDotPart(), strconv.FormatBool(c.Instructor))
		}
		w.Write(record)
	}
	w.Flush()
	return nil
}

func (svc *dashboardService) rand() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}

func (svc *dashboardService) Serve_dashboard_evalCodes(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	classes, err := svc.store.GetAllClassesFull(rc.context())
	if err != nil {
		return err
	}

	// Collect codes in use.
	evaluationCodes := make(map[string]int)
	accessTokens := make(map[string]int)
	for _, class := range classes {
		if class.AccessToken != "" {
			num, ok := accessTokens[class.AccessToken]
			if ok {
				return &httperror.Error{
					Status:  http.StatusInternalServerError,
					Message: fmt.Sprintf("Access token %s used in class %d and %d", class.AccessToken, num, class.Number),
				}
			}
			accessTokens[class.AccessToken] = class.Number
		}
		for _, code := range strings.Split(class.EvaluationCodes, ",") {
			code = strings.TrimSpace(code)
			num, ok := evaluationCodes[code]
			if ok {
				return &httperror.Error{
					Status:  http.StatusInternalServerError,
					Message: fmt.Sprintf("Code %s used in class %d and %d", code, num, class.Number),
				}
			}
			evaluationCodes[code] = class.Number
		}
	}

	// Don't assign these codes
	evaluationCodes["0000"] = 0
	evaluationCodes["1234"] = 0

	for _, class := range classes {
		if class.AccessToken == "" {
			for i := 0; i < 1000; i++ {
				r, err := svc.rand()
				if err != nil {
					return err
				}
				token := fmt.Sprintf("%08x", r)
				_, ok := accessTokens[token]
				if !ok {
					accessTokens[token] = class.Number
					class.AccessToken = token
					break
				}
			}
		}
		codes := strings.Split(class.EvaluationCodes, ",")
		if len(codes) > class.Length {
			// Remove extra codes.
			codes = codes[:class.Length]
		} else {
			// Add codes to class as needed.
			for i := len(codes); i < class.Length; i++ {
				for j := 0; j < 1000; j++ {
					r, err := svc.rand()
					if err != nil {
						return err
					}
					code := fmt.Sprintf("%04d", r%10000)
					_, ok := evaluationCodes[code]
					if !ok {
						evaluationCodes[code] = class.Number
						codes = append(codes, code)
						break
					}
				}
			}
		}
		class.EvaluationCodes = strings.Join(codes, ", ")
	}

	rc.response.Header().Set("Content-Type", "text/csv")
	rc.response.Header().Set("Content-Disposition", `attachment; filename="classes.csv"`)

	model.SortClasses(classes, "")

	// Quote tokens and codes in output to prevent spreadsheet from
	// interpreting the values as numbers.
	fmt.Fprintf(rc.response, "\"class\",\"accessToken\",\"evaluationCodes\"\n")
	for _, class := range classes {
		fmt.Fprintf(rc.response, "\"%d\",\"=\"\"%s\"\"\",\"=\"\"%s\"\"\"\n", class.Number, class.AccessToken, class.EvaluationCodes)
	}
	return nil
}

func (svc *dashboardService) Serve_dashboard_setDebugTime(rc *requestContext) error {
	what := rc.request.FormValue("time")
	svc.debugTimeCodec.Encode(rc.response, what)
	return rc.redirect("/dashboard/admin", "info", "Debug time set to %q.", what)
}
