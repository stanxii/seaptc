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

	"github.com/seaptc/server/dk"
	"github.com/seaptc/server/model"
	"github.com/seaptc/server/sheet"
	"github.com/seaptc/server/store"
)

type dashboardService struct {
	*application
	templates struct {
		Index              *templates.Template `html:"dashboard/index.html dashboard/root.html common.html"`
		Error              *templates.Template `html:"dashboard/error.html dashboard/root.html common.html"`
		Classes            *templates.Template `html:"dashboard/classes.html dashboard/root.html common.html"`
		Participants       *templates.Template `html:"dashboard/participants.html dashboard/root.html common.html"`
		Participant        *templates.Template `html:"dashboard/participant.html dashboard/root.html common.html"`
		Class              *templates.Template `html:"dashboard/class.html dashboard/root.html common.html"`
		Admin              *templates.Template `html:"dashboard/admin.html dashboard/root.html common.html"`
		Conference         *templates.Template `html:"dashboard/conference.html dashboard/root.html common.html"`
		FetchRegistrations *templates.Template `html:"dashboard/fetchRegistrations.html dashboard/root.html common.html"`
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
	classes, err := svc.store.GetAllClasses(rc.context())
	if err != nil {
		return err
	}

	registered, err := svc.store.GetClassParticipantCounts(rc.context())
	if err != nil {
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
		Registered interface{}
		Available  interface{}
	}{
		Classes: classes,
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
	class, err := svc.getClass(rc, "/dashboard/classes/")
	if err != nil {
		return err
	}

	var data = struct {
		InstructorView    bool
		Class             *model.Class
		Participants      []*model.Participant
		ParticipantEmails []string
		InstructorURL     string
		Lunch             *model.Lunch // XXX
	}{
		Class:          class,
		InstructorView: rc.isStaff,
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
		data.Participants, err = svc.store.GetClassParticipants(rc.context(), class.Number)
		if err != nil {
			return err
		}
		model.SortParticipants(data.Participants, rc.request.FormValue("sort"))
		for _, p := range data.Participants {
			data.ParticipantEmails = append(data.ParticipantEmails, p.Emails()...)
		}
	}

	return rc.respond(svc.templates.Class, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_participants(rc *requestContext) error {
	participants, err := svc.store.GetAllParticipants(rc.context())
	if err != nil {
		return err
	}
	classes, err := svc.store.GetAllClasses(rc.context())
	if err != nil {
		return err
	}
	classMap := model.ClassMap(classes)
	model.SortParticipants(participants, rc.request.FormValue("sort"))

	var data = struct {
		Participants   []*model.Participant
		SessionClasses interface{}
	}{
		Participants: participants,
		SessionClasses: func(p *model.Participant) []*model.SessionClass {
			return model.ParticipantSessionClasses(p, classMap)
		},
	}
	return rc.respond(svc.templates.Participants, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_participants_(rc *requestContext) error {
	if !rc.isStaff {
		return httperror.ErrForbidden
	}

	id := strings.TrimPrefix(rc.request.URL.Path, "/dashboard/participants/")

	participant, err := svc.store.GetParticipant(rc.context(), id)
	if err == store.ErrNotFound {
		return httperror.ErrNotFound
	} else if err != nil {
		return err
	}

	classes, err := svc.store.GetAllClasses(rc.context())
	if err != nil {
		return err
	}
	sessionClasses := model.ParticipantSessionClasses(participant, model.ClassMap(classes))

	var data = struct {
		Participant    *model.Participant
		SessionClasses []*model.SessionClass
	}{
		participant,
		sessionClasses,
	}
	return rc.respond(svc.templates.Participant, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_upload__registrations(rc *requestContext) error {
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

	n, err := svc.store.ImportParticipants(rc.context(), participants)
	if err != nil {
		return err
	}

	return rc.redirect("/dashboard/admin", "info", "%d registrations loaded from file, %d modified", len(participants), n)
}

func (svc *dashboardService) Serve_dashboard_fetch__registrations(rc *requestContext) error {
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	rc.request.ParseForm()
	data := struct {
		Form    url.Values
		Invalid map[string]string
		Err     error
	}{
		rc.request.Form,
		make(map[string]string),
		nil,
	}

	if rc.request.Method != "POST" {
		return rc.respond(svc.templates.FetchRegistrations, http.StatusOK, &data)
	}

	url := rc.request.FormValue("url")
	if url == "" {
		data.Invalid["url"] = ""
	}

	if len(data.Invalid) > 0 {
		return rc.respond(svc.templates.FetchRegistrations, http.StatusOK, &data)
	}

	header := make(http.Header)
	for _, key := range []string{"Accept", "Accept-Language", "User-Agent"} {
		header[key] = rc.request.Header[key]
	}

	participants, err := dk.FetchCSV(rc.context(), url, header)
	if err != nil {
		data.Err = err
		return rc.respond(svc.templates.FetchRegistrations, http.StatusOK, &data)
	}

	n, err := svc.store.ImportParticipants(rc.context(), participants)
	if err != nil {
		return err
	}

	return rc.redirect("/dashboard/admin", "info", "%d registrations loaded from file, %d modified", len(participants), n)
}

func (svc *dashboardService) Serve_dashboard_refresh__classes(rc *requestContext) error {
	if rc.request.Method != "POST" {
		return httperror.ErrMethodNotAllowed
	}
	if !rc.isStaff {
		return httperror.ErrForbidden
	}

	classes, err := sheet.GetClasses(rc.context(), svc.config)
	if err != nil {
		return err
	}

	suggestedSchedules, err := sheet.GetSuggestedSchedules(rc.context(), svc.config)
	if err != nil {
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
	if len(data.Invalid) > 0 {
		return rc.respond(svc.templates.Conference, http.StatusOK, &data)
	}

	err = svc.store.SetConference(rc.context(), conf)
	if err != nil {
		return err
	}

	return rc.redirect(rc.request.URL.Path, "info", "Conference updated.")
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
