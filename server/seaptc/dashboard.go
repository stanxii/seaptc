package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"

	"github.com/seaptc/server/model"
	"github.com/seaptc/server/sheet"
	"github.com/seaptc/server/store"
)

type dashboardService struct {
	*application
	templates struct {
		Index        *templates.Template `html:"dashboard/index.html dashboard/root.html common.html"`
		Error        *templates.Template `html:"dashboard/error.html dashboard/root.html common.html"`
		Classes      *templates.Template `html:"dashboard/classes.html dashboard/root.html common.html"`
		Class        *templates.Template `html:"dashboard/class.html dashboard/root.html common.html"`
		SessionEvent *templates.Template `html:"dashboard/sessionevent.html"`
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
	return rc.respond(svc.templates.Index, http.StatusOK, nil)
}

func (svc *dashboardService) Serve_dashboard_participants(rc *requestContext) error {
	return rc.respond(svc.templates.Index, http.StatusOK, nil)
}

func (svc *dashboardService) Serve_dashboard_classes(rc *requestContext) error {
	var data struct {
		Classes []*model.Class
	}
	classes, err := svc.store.GetAllClasses(rc.context())
	if err != nil {
		return err
	}
	data.Classes = model.SortClasses(classes, rc.request.FormValue("sort"))
	return rc.respond(svc.templates.Classes, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_classes_(rc *requestContext) error {
	class, err := svc.getClass(rc, "/dashboard/classes/")
	if err != nil {
		return err
	}

	var data = struct {
		Class          *model.Class
		Participants   []*model.Participant
		InstructorView bool
		InstructorURL  string
		Lunch          *model.Lunch // XXX
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
	return rc.respond(svc.templates.Class, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_refresh__classes(rc *requestContext) error {
	if rc.request.Method != "POST" {
		return httperror.ErrMethodNotAllowed
	}
	if !rc.isStaff {
		return httperror.ErrForbidden
	}

	classes, err := sheet.Fetch(rc.context(), svc.config)
	if err != nil {
		return err
	}
	n, err := svc.store.UpdateClassesFromSheet(rc.context(), classes)
	if err != nil {
		return err
	}
	rc.setFlashMessage("info", "%d classes loaded from sheet, %d modified", len(classes), n)

	ref := rc.request.FormValue("ref")
	if ref == "" {
		ref = "/dashboard/classes"
	}
	return rc.redirect(ref, http.StatusSeeOther)
}

func (svc *dashboardService) Serve_dashboard_sessionevents_(rc *requestContext) error {
	h := rc.response.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "*")
	if rc.request.Method == "OPTIONS" {
		rc.response.WriteHeader(http.StatusNoContent)
		return nil
	}

	class, err := svc.getClass(rc, "/dashboard/sessionevents/")
	if err != nil {
		return err
	}
	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	date := conf.Date()
	start := date.Add(model.Sessions[class.Start()].Start)
	end := date.Add(model.Sessions[class.End()].End)

	description := fmt.Sprintf("%d: %s", class.Number, class.Title)
	if class.TitleNotes != "" {
		description = fmt.Sprintf("%s (%s)", description, class.TitleNotes)
	}

	var maxEventAttendees string
	switch {
	case class.Capacity < 0:
		// canceled or disabled
		maxEventAttendees = "0"
	case class.Capacity > 0:
		// explicity capacity, zero is infinite
		maxEventAttendees = strconv.Itoa(class.Capacity)
	}

	noteData := struct {
		*model.Class
		AppropriateFor string
		Sessions       string
	}{
		Class: class,
	}

	programs := class.ProgramDescriptions(false)
	if len(programs) > 0 && programs[0].Code != "all" {
		var buf []byte
		for i, p := range programs {
			if i == 0 {
				// no separator
			} else if i == len(programs)-1 {
				buf = append(buf, " and "...)
			} else {
				buf = append(buf, ", "...)
			}
			buf = append(buf, p.Name...)
		}
		noteData.AppropriateFor = string(buf)
	}

	if class.Length <= 1 {
		noteData.Sessions = fmt.Sprintf("1 hour, session %d", class.Start()+1)
	} else {
		noteData.Sessions = fmt.Sprintf("%d hours, sessions %d – %d", class.Length, class.Start()+1, class.End()+1)
	}

	var note bytes.Buffer
	if err := svc.templates.SessionEvent.Execute(&note, &noteData); err != nil {
		return err
	}

	data := map[string]string{
		"Description":           description,
		"ActivityDate":          date.Format("1/2/2006"),
		"EndDate":               date.Format("1/2/2006"),
		"AllowRegEdit":          "on",
		"Notes":                 note.String(),
		"Address":               "9600 College Way North",
		"City":                  "Seattle",
		"State":                 "WA",
		"Postal_Code":           "98103",
		"Country":               "US",
		"RegistrationStartDate": fmt.Sprintf("1/1/%d", conf.Year),
		"RegistrationStartHour": "12",
		"RegistrationStartMin":  "10",
		"RegistrationStartAMPM": "AM",
		"RegisterByDate":        date.Format("1/2/2006"),
		"RegisterByHour":        "10",
		"RegisterByMin":         "0",
		"RegisterByAMPM":        "AM",
		"ActivityFromHour":      start.Format("3"),
		"ActivityFromMin":       start.Format("4"),
		"ActivityFromAMPM":      start.Format("PM"),
		"ActivityTillHour":      end.Format("3"),
		"ActivityTillMin":       end.Format("4"),
		"ActivityTillAMPM":      end.Format("PM"),
		"MaxAttendees":          maxEventAttendees,
	}

	p, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	rc.response.Header().Set("Content-Type", "application/json")
	rc.response.Write(p)
	return nil
}
