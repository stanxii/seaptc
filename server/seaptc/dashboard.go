package main

import (
	"context"
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
		Form          *templates.Template `html:"dashboard/form.html"`
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
		sort.SliceStable(classes, func(i, j int) bool {
			return registered[classes[i].Number] < registered[classes[j].Number]
		})
		reverse(classes)
	case "available":
		sort.SliceStable(classes, func(i, j int) bool {
			m := classes[i].Capacity - registered[classes[i].Number]
			if classes[i].Capacity == 0 {
				m = 9999
			}
			n := classes[j].Capacity - registered[classes[j].Number]
			if classes[j].Capacity == 0 {
				n = 9999
			}
			return m < n
		})
		reverse(classes)
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
	var (
		g            errgroup.Group
		participants []*model.Participant
		classes      []*model.Class
	)

	g.Go(func() error {
		var err error
		participants, err = svc.store.GetAllParticipants(rc.context())
		return err
	})

	g.Go(func() error {
		var err error
		classes, err = svc.store.GetAllClasses(rc.context())
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	model.SortParticipants(participants, rc.request.FormValue("sort"))

	var data = struct {
		Participants   []*model.Participant
		SessionClasses interface{}
	}{
		participants,
		model.NewClassMap(classes).ParticipantSessionClasses,
	}
	return rc.respond(svc.templates.Participants, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_participants_(rc *requestContext) error {
	if !rc.isStaff {
		return httperror.ErrForbidden
	}

	id := strings.TrimPrefix(rc.request.URL.Path, "/dashboard/participants/")

	var (
		g           errgroup.Group
		participant *model.Participant
		conf        *model.Conference
		classes     []*model.Class
	)

	g.Go(func() error {
		var err error
		participant, err = svc.store.GetParticipant(rc.context(), id)
		if err == store.ErrNotFound {
			err = httperror.ErrNotFound
		}
		return err
	})

	g.Go(func() error {
		var err error
		conf, err = svc.store.GetConference(rc.context())
		return err
	})

	g.Go(func() error {
		var err error
		classes, err = svc.store.GetAllClasses(rc.context())
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	var data = struct {
		Participant    *model.Participant
		SessionClasses []*model.SessionClass
		Lunch          *model.Lunch
	}{
		participant,
		model.NewClassMap(classes).ParticipantSessionClasses(participant),
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

	return rc.redirect("/dashboard/admin", "info", "Import %d records; ", len(participants), summary)
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
		data.Form.Set("year", strconv.Itoa(conf.Year))
		data.Form.Set("month", strconv.Itoa(conf.Month))
		data.Form.Set("day", strconv.Itoa(conf.Day))
		p, _ := json.MarshalIndent(conf.Lunches, "", "  ")
		data.Form.Set("lunches", string(p))
		return rc.respond(svc.templates.Conference, http.StatusOK, &data)
	}

	setInt := func(pi *int, key string) {
		var err error
		*pi, err = strconv.Atoi(data.Form.Get(key))
		if err != nil {
			data.Invalid[key] = "is-invalid"
		}
	}
	setInt(&conf.Year, "year")
	setInt(&conf.Month, "month")
	setInt(&conf.Day, "day")

	if err := json.Unmarshal([]byte(data.Form.Get("lunches")), &conf.Lunches); err != nil {
		data.Invalid["lunches"] = err.Error()
	}

	conf.RegistrationURL = data.Form.Get("registrationURL")
	conf.CatalogStatusMessage = data.Form.Get("catalogStatusMessage")
	conf.NoClassDescription = data.Form.Get("noClassDescription")
	conf.OABanquetDescription = data.Form.Get("oaBanquetDescription")
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
	data := struct {
		DevMode bool
	}{
		DevMode: svc.devMode,
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

var formSorts = map[string]func([]*model.Participant){
	"debugFirst": func(participants []*model.Participant) {
		sort.Slice(participants, func(i, j int) bool {
			a := len(participants[i].NicknameOrFirstName())
			b := len(participants[j].NicknameOrFirstName())
			switch {
			case a > b:
				return true
			case a < b:
				return false
			default:
				return model.DefaultParticipantLess(participants[i], participants[j])
			}
		})
	},
	"debugLast": func(participants []*model.Participant) {
		sort.Slice(participants, func(i, j int) bool {
			a := len(participants[i].LastName)
			if participants[i].Suffix != "" {
				a += 1 + len(participants[i].Suffix)
			}
			b := len(participants[j].LastName)
			if participants[j].Suffix != "" {
				a += 1 + len(participants[j].Suffix)
			}
			switch {
			case a > b:
				return true
			case a < b:
				return false
			default:
				return model.DefaultParticipantLess(participants[i], participants[j])
			}
		})
	},
}

func (svc *dashboardService) Serve_dashboard_forms_(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}
	what := strings.TrimPrefix(rc.request.URL.Path, "/dashboard/forms/")
	if what == "" {
		return httperror.ErrNotFound
	}

	ids := []string{what}
	if sort := formSorts[what]; sort != nil {
		participants, err := svc.store.GetAllParticipants(rc.context())
		if err != nil {
			return err
		}
		sort(participants)
		ids = make([]string, len(participants))
		for i := range participants {
			ids[i] = participants[i].ID
		}
	}
	return svc.renderForms(rc, ids)
}

func (svc *dashboardService) renderForms(rc *requestContext, ids []string) error {

	var (
		g            errgroup.Group
		participants []*model.Participant
		conf         *model.Conference
		classes      []*model.Class
	)

	g.Go(func() error {
		var err error
		participants, err = svc.store.GetParticipants(rc.context(), ids)
		return err
	})

	g.Go(func() error {
		var err error
		conf, err = svc.store.GetConference(rc.context())
		return err
	})

	g.Go(func() error {
		var err error
		classes, err = svc.store.GetAllClasses(rc.context())
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	var data = struct {
		Participants   []*model.Participant
		Conference     *model.Conference
		Lunch          interface{}
		SessionClasses interface{}
	}{
		participants,
		conf,
		conf.ParticipantLunch,
		model.NewClassMap(classes).ParticipantSessionClasses,
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
