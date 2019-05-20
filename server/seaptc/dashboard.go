package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/seaptc/server/model"
	"github.com/seaptc/server/sheet"
	"github.com/seaptc/server/store"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
)

type dashboardService struct {
	*application
	templates struct {
		Index   *templates.Template `html:"dashboard/index.html dashboard/root.html common.html"`
		Error   *templates.Template `html:"dashboard/error.html dashboard/root.html common.html"`
		Classes *templates.Template `html:"dashboard/classes.html dashboard/root.html common.html"`
		Class   *templates.Template `html:"dashboard/class.html dashboard/root.html common.html"`
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

func (svc *dashboardService) Serve_dashboard_(rc *requestContext) error {
	// Use dashboard error template for all not found requests to the /dashboard/ tree.
	return httperror.ErrNotFound
}

func (svc *dashboardService) Serve_dashboard(rc *requestContext) error {
	return rc.respond(svc.templates.Index, http.StatusOK, nil)
}

func (svc *dashboardService) Serve_dashboard_classes(rc *requestContext) error {
	var data struct {
		Classes []*model.Class
	}

	classes, err := svc.store.GetClasses(rc.context(), 0)
	if err != nil {
		return err
	}

	data.Classes = model.SortedClasses(classes, rc.request.FormValue("sort"))
	return rc.respond(svc.templates.Classes, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_classes_(rc *requestContext) error {
	number := strings.TrimPrefix(rc.request.URL.Path, "/dashboard/classes/")

	var data struct {
		InstructorView bool
		Class          *model.Class
		Participants   []*model.Participant
		Lunch          *model.Lunch // XXX
		InstructorURL  string
	}

	var err error
	data.Class, err = svc.store.GetClass(rc.context(), number)
	if store.IsNotFoundError(err) {
		return httperror.ErrNotFound
	} else if err != nil {
		return err
	}

	data.InstructorView = rc.isStaff
	if len(data.Class.AccessToken) >= 4 {
		protocol := "https"
		if svc.devMode {
			protocol = "http"
		}
		data.InstructorURL = fmt.Sprintf("%s://%s/dashboard/classes/%s?t=%s", protocol, rc.request.Host, number, data.Class.AccessToken)
		if rc.request.FormValue("t") == data.Class.AccessToken {
			data.InstructorView = true
		}
	}
	return rc.respond(svc.templates.Class, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_classes_refresh(rc *requestContext) error {
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
