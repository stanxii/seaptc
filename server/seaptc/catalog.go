package main

import (
	"context"
	"net/http"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
)

type catalogService struct {
	templates struct {
		Index *templates.Template `html:"catalog/index.html"`
		Error *templates.Template `html:"catalog/error.html"`
	}
}

func (svc *catalogService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	tm.NewFromFields(&svc.templates)
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

func (svc *catalogService) Serve_(rc *requestContext) error {
	if rc.request.URL.Path != "/" {
		return httperror.ErrNotFound
	}
	return rc.respond(svc.templates.Index, http.StatusOK, nil)
}
