package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
		Index      *templates.Template `html:"dashboard/index.html dashboard/root.html common.html"`
		Error      *templates.Template `html:"dashboard/error.html dashboard/root.html common.html"`
		Classes    *templates.Template `html:"dashboard/classes.html dashboard/root.html common.html"`
		Class      *templates.Template `html:"dashboard/class.html dashboard/root.html common.html"`
		Admin      *templates.Template `html:"dashboard/admin.html dashboard/root.html common.html"`
		Conference *templates.Template `html:"dashboard/conference.html dashboard/root.html common.html"`
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

	classes, err := sheet.GetClasses(rc.context(), svc.config)
	if err != nil {
		return err
	}

	suggestedSchedules, err := sheet.GetSuggestedSchedules(rc.context(), svc.config)
	if err != nil {
		return err
	}

	n, err := svc.store.UpdateClassesFromSheet(rc.context(), classes, rc.request.FormValue("all") != "")
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
	}{
		rc.request.Form,
		make(map[string]string),
		conf,
		model.ProgramDescriptions,
	}

	if rc.request.Method != "POST" {
		data.Form.Set("year", strconv.Itoa(conf.Year))
		data.Form.Set("month", strconv.Itoa(conf.Month))
		data.Form.Set("day", strconv.Itoa(conf.Day))
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
	conf.RegistrationURL = data.Form.Get("registrationURL")
	conf.CatalogStatusMessage = data.Form.Get("catalogStatusMessage")
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

func (svc *dashboardService) handleSessionEventsCORS(rc *requestContext) bool {
	h := rc.response.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "*")
	if rc.request.Method != "OPTIONS" {
		return false
	}
	rc.response.WriteHeader(http.StatusNoContent)
	return true
}
