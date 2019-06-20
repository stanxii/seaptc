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
	"rsc.io/qr"

	"github.com/seaptc/server/dk"
	"github.com/seaptc/server/model"
	"github.com/seaptc/server/sheet"
	"github.com/seaptc/server/store"
)

type dashboardService struct {
	*application
	templates struct {
		Admin              *templates.Template `html:"dashboard/admin.html dashboard/root.html common.html"`
		Class              *templates.Template `html:"dashboard/class.html dashboard/root.html common.html"`
		Classes            *templates.Template `html:"dashboard/classes.html dashboard/root.html common.html"`
		Conference         *templates.Template `html:"dashboard/conference.html dashboard/root.html common.html"`
		Error              *templates.Template `html:"dashboard/error.html dashboard/root.html common.html"`
		FetchRegistrations *templates.Template `html:"dashboard/fetchRegistrations.html dashboard/root.html common.html"`
		Index              *templates.Template `html:"dashboard/index.html dashboard/root.html common.html"`
		Instructors        *templates.Template `html:"dashboard/instructors.html dashboard/root.html common.html"`
		Participant        *templates.Template `html:"dashboard/participant.html dashboard/root.html common.html"`
		Participants       *templates.Template `html:"dashboard/participants.html dashboard/root.html common.html"`
		Reprint            *templates.Template `html:"dashboard/reprint.html dashboard/root.html common.html"`

		Form *templates.Template `html:"dashboard/form.html"`
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

	conf, err := svc.store.GetConference(rc.context())
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
	class, err := svc.getClass(rc, "/dashboard/classes/")
	if err != nil {
		return err
	}

	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
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

	participant, err := svc.store.GetParticipant(rc.context(), id)
	if err == store.ErrNotFound {
		return httperror.ErrNotFound
	} else if err != nil {
		return err
	}

	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	classes, err := svc.store.GetAllClasses(rc.context())
	if err != nil {
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

	n, err := svc.store.ImportParticipants(rc.context(), participants)
	if err != nil {
		return err
	}

	return rc.redirect("/dashboard/admin", "info", "%d registrations loaded from file, %d modified", len(participants), n)
}

func (svc *dashboardService) Serve_dashboard_fetchRegistrations(rc *requestContext) error {
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

func (svc *dashboardService) Serve_dashboard_refreshClasses(rc *requestContext) error {
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
	participants, err := svc.store.GetParticipants(rc.context(), ids)
	if err != nil {
		return err
	}
	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	classes, err := svc.store.GetAllClasses(rc.context())
	if err != nil {
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
