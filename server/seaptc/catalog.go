package main

import (
	"context"
	"net/http"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"

	"github.com/seaptc/server/model"
)

type catalogService struct {
	templates struct {
		Index   *templates.Template `html:"catalog/index.html catalog/root.html"`
		Error   *templates.Template `html:"catalog/error.html catalog/root.html"`
		Program *templates.Template `html:"catalog/program.html catalog/root.html"`
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

func (svc *catalogService) serveProgram(rc *requestContext, program int) error {

	return nil
}

func (svc *catalogService) Serve_catalog_cubscout(rc *requestContext) error {
	return svc.serveProgram(rc, model.CubScoutProgram)
}

func (svc *catalogService) Server_catalog_scoutsbsa(rc *requestContext) error {
	return svc.serveProgram(rc, model.ScoutsBSAProgram)
}

func (svc *catalogService) Server_catalog_venturing(rc *requestContext) error {
	return svc.serveProgram(rc, model.VenturingProgram)
}

func (svc *catalogService) Server_catalog_seascout(rc *requestContext) error {
	return svc.serveProgram(rc, model.SeaScoutProgram)
}

func (svc *catalogService) Serve_catalog_commisioner(rc *requestContext) error {
	return svc.serveProgram(rc, model.CommissionerProgram)
}

func (svc *catalogService) Serve_catalog_youth(rc *requestContext) error {
	return svc.serveProgram(rc, model.YouthProgram)
}

func (svc *catalogService) Serve_catalog_all(rc *requestContext) error {
	return nil
}

func (svc *catalogService) Serve_catalog_new(rc *requestContext) error {
	return nil
}
