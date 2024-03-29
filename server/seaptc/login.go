package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/garyburd/web/cookie"
	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type loginService struct {
	*application
	loginStateCodec *cookie.Codec
}

func (svc *loginService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	svc.application = a
	svc.loginStateCodec = cookie.NewCodec("login", cookie.WithSecure(!a.devMode))
	return nil
}

func (svc *loginService) errorTemplate() *templates.Template { return nil }

func (svc *loginService) makeHandler(v interface{}) func(*requestContext) error {
	f, ok := v.(func(*loginService, *requestContext) error)
	if !ok {
		return nil
	}
	return func(rc *requestContext) error { return f(svc, rc) }
}

func (svc *loginService) oauth2ConfigForRequest(rc *requestContext) *oauth2.Config {
	proto := "https"
	if svc.devMode {
		proto = "http"
	}
	return &oauth2.Config{
		ClientID:     svc.config.LoginClient.ID,
		ClientSecret: svc.config.LoginClient.Secret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		RedirectURL:  fmt.Sprintf("%s://%s/login/callback", proto, rc.request.Host),
		Endpoint:     google.Endpoint,
	}
}

func (svc *loginService) Serve_dashboard_login(rc *requestContext) error {
	p := make([]byte, 32)
	rand.Read(p)
	state := fmt.Sprintf("%x", p)
	if err := svc.loginStateCodec.Encode(rc.response, state, rc.request.FormValue("_ref")); err != nil {
		return err
	}
	c := svc.oauth2ConfigForRequest(rc)
	http.Redirect(rc.response, rc.request, c.AuthCodeURL(state), http.StatusSeeOther)
	return nil
}

func (svc *loginService) Serve_login_callback(rc *requestContext) error {
	var state, ref string
	if err := svc.loginStateCodec.Decode(rc.request, &state, &ref); err != nil {
		return err
	}
	if rc.request.FormValue("state") != state {
		return httperror.ErrForbidden
	}
	c := svc.oauth2ConfigForRequest(rc)
	token, err := c.Exchange(rc.ctx, rc.request.FormValue("code"))
	if err != nil {
		return &httperror.Error{Status: http.StatusBadRequest, Err: err}
	}
	client := c.Client(rc.ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var userInfo struct {
		Email string
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return err
	}
	id := strings.ToLower(userInfo.Email)
	if svc.adminIDs[id] || svc.isStaff(rc.ctx, id) {
		svc.staffIDCodec.Encode(rc.response, id)
		rc.logf("login success: %s", id)
	} else {
		rc.setFlashMessage("info", fmt.Sprintf("The account %s is not authorized to access this service.", userInfo.Email))
		rc.logf("login fail: %s", id)
	}
	if ref == "" {
		ref = "/dashboard"
	}
	http.Redirect(rc.response, rc.request, ref, http.StatusSeeOther)
	return nil
}

func (svc *loginService) Serve_dashboard_logout(rc *requestContext) error {
	svc.staffIDCodec.Encode(rc.response)
	http.Redirect(rc.response, rc.request, "/dashboard", http.StatusSeeOther)
	return nil
}
