package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/seaptc/server/data"
	"github.com/seaptc/server/sheet"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
)

type dashboardService struct {
	*application
	templates struct {
		Index   *templates.Template `html:"dashboard/index.html dashboard/root.html common.html"`
		Error   *templates.Template `html:"dashboard/error.html dashboard/root.html common.html"`
		Classes *templates.Template `html:"dashboard/classes.html dashboard/root.html common.html"`
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
		Classes []*data.Class
	}

	classes, err := svc.store.GetClasses(rc.context())
	if err != nil {
		return err
	}
	data.Classes = classes.Slice

	switch rc.request.FormValue("sort") {
	case "location":
		sort.Slice(data.Classes, func(i, j int) bool { return data.Classes[i].Location < data.Classes[j].Location })
	case "responsibility":
		sort.Slice(data.Classes, func(i, j int) bool { return data.Classes[i].Responsibility < data.Classes[j].Responsibility })
	case "capacity":
		sort.Slice(data.Classes, func(i, j int) bool { return data.Classes[i].Capacity < data.Classes[j].Capacity })
	}

	return rc.respond(svc.templates.Classes, http.StatusOK, &data)
}

func (svc *dashboardService) Serve_dashboard_classes_refresh(rc *requestContext) error {
	if rc.request.Method != "POST" {
		return httperror.ErrMethodNotAllowed
	}
	if !rc.isStaff {
		fmt.Println(svc.staffIDs, rc.staffID)
		return httperror.ErrForbidden
	}

	classes, err := sheet.Fetch(rc.context(), svc.config)
	if err != nil {
		return err
	}
	if err := svc.store.UpdateClasses(rc.context(), classes, (*data.Class).SheetFields); err != nil {
		return err
	}
	rc.setFlashMessage("info", "%d classes updated", len(classes))

	ref := rc.request.FormValue("ref")
	if ref == "" {
		ref = "/dashboard/classes"
	}
	return rc.redirect(ref, http.StatusSeeOther)
}
