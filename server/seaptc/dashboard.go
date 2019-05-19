package main

import (
	"context"
	"net/http"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
)

type dashboardService struct {
	templates struct {
		Index *templates.Template `html:"dashboard/root.html dashboard/index.html"`
		Error *templates.Template `html:"dashboard/root.html dashboard/error.html"`
	}
}

func (svc *dashboardService) init(ctx context.Context, a *application, tm *templates.Manager) error {
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
