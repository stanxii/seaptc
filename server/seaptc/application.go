package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/garyburd/web/cookie"
	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
	"github.com/seaptc/server/model"
	"github.com/seaptc/server/store"

	"golang.org/x/net/xsrftoken"
)

type application struct {
	devMode bool
	store   *store.Store

	nextRequestID int64 // incremented on each request, used in log output

	config *model.AppConfig

	flashCodec         *cookie.Codec
	staffIDCodec       *cookie.Codec
	participantIDCodec *cookie.Codec

	adminIDs map[string]bool
	staffIDs map[string]bool
}

type applicationService interface {
	init(context.Context, *application, *templates.Manager) error
	makeHandler(v interface{}) func(*requestContext) error
	errorTemplate() *templates.Template
}

func newApplication(ctx context.Context, st *store.Store, devMode bool, assetDir string, services ...applicationService) (http.Handler, error) {

	config, err := st.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}

	var hmacKeys [][]byte
	for _, k := range config.HMACKeys {
		hmacKeys = append(hmacKeys, []byte(k))
	}

	a := application{
		devMode:  devMode,
		store:    st,
		config:   config,
		adminIDs: make(map[string]bool),
		staffIDs: make(map[string]bool),
		flashCodec: cookie.NewCodec("f",
			cookie.WithSecure(!devMode)),
		staffIDCodec: cookie.NewCodec("s",
			cookie.WithMaxAge(30*24*time.Hour),
			cookie.WithHMACKeys(hmacKeys),
			cookie.WithSecure(!devMode)),
		participantIDCodec: cookie.NewCodec("i",
			cookie.WithMaxAge(24*time.Hour),
			cookie.WithHMACKeys(hmacKeys),
			cookie.WithSecure(!devMode)),
	}

	for _, id := range a.config.AdminIDs {
		a.adminIDs[id] = true
		a.staffIDs[id] = true
	}
	for _, id := range a.config.StaffIDs {
		a.staffIDs[id] = true
	}

	tm := newTemplateManager(assetDir)
	mux := http.NewServeMux()

	// The following handlers should match the static file handlers declared
	// app.yaml.
	staticFile := func(name string) http.HandlerFunc {
		p := filepath.Join(assetDir, "static", name)
		return func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, p)
		}
	}
	mux.Handle("/static/", http.FileServer(http.Dir(assetDir)))
	mux.Handle("/robots.txt", staticFile("robots.txt"))
	mux.Handle("/favicon.ico", staticFile("favicon.ico"))

	for _, svc := range services {
		if err := svc.init(ctx, &a, tm); err != nil {
			return nil, err
		}

		t := reflect.TypeOf(svc)
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			if !strings.HasPrefix(m.Name, "Serve_") {
				continue
			}
			f := svc.makeHandler(m.Func.Interface())
			if f == nil {
				return nil, fmt.Errorf("could not create handler for %v.%s", t, m.Name)
			}
			// Convert __ to - and _ to /.
			path := strings.ReplaceAll(strings.TrimPrefix(m.Name, "Serve"), "__", "-")
			path = strings.ReplaceAll(path, "_", "/")
			mux.Handle(path, &handler{application: &a, svc: svc, f: f})
		}
	}

	if err := tm.Load(filepath.Join(assetDir, "templates"), a.devMode); err != nil {
		return nil, err
	}
	return mux, nil
}

type handler struct {
	application *application
	svc         applicationService
	f           func(*requestContext) error
}

func (h *handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if s := strings.TrimPrefix(request.Host, "dashboard."); s != request.Host {
		// Redirect for those accustomed to previous year's dashboard URL.
		http.Redirect(response, request, fmt.Sprintf("https://%s/dashboard", s), http.StatusSeeOther)
	}

	a := h.application
	rc := requestContext{
		application: a,
		response:    response,
		request:     request,
		logPrefix:   fmt.Sprintf("%06d ", atomic.AddInt64(&a.nextRequestID, 1)),
	}

	if err := a.staffIDCodec.Decode(rc.request, &rc.staffID); err != nil {
		rc.staffID = ""
	}

	if err := a.participantIDCodec.Decode(rc.request, &rc.participantID); err != nil {
		rc.participantID = ""
	}

	// Clobber ids if XSRF token is not valid.
	if request.Method != "HEAD" && request.Method != "GET" {
		id := fmt.Sprintf("%s\000%s", rc.staffID, rc.participantID)
		if !xsrftoken.Valid(request.FormValue("_xsrftoken"), rc.application.config.XSRFKey, id, request.URL.Path) {
			rc.logf("xsrf check failed for staffID=%q, particpiantID=%q", rc.staffID, rc.participantID)
			rc.participantID = ""
			rc.staffID = ""
		}
	}

	rc.isAdmin = a.adminIDs[rc.staffID]
	rc.isStaff = a.staffIDs[rc.staffID]

	rc.logf("request: %s %s %s", rc.request.Method, rc.request.URL.Path, rc.staffID)

	err := h.f(&rc)

	if err != nil {
		rc.logf("resp error: %+v", err)
		e := httperror.Convert(err)
		if t := h.svc.errorTemplate(); t != nil {
			err := rc.respond(h.svc.errorTemplate(), e.Status, e)
			if err != nil {
				rc.logf("error rendering error template: %v", err)
			} else {
				return
			}
		}
		rc.response.Header().Set("Content-Type", "text/plain")
		http.Error(rc.response, e.Message, e.Status)
	}
}

type requestContext struct {
	application *application
	request     *http.Request
	response    http.ResponseWriter

	logPrefix string

	staffID          string
	isAdmin, isStaff bool

	participantID string
}

func (rc *requestContext) redirect(path string, flashKind string, flashFormat string, flashArgs ...interface{}) error {
	rc.setFlashMessage(flashKind, flashFormat, flashArgs...)
	if p := rc.request.FormValue("_ref"); p != "" {
		path = p
	}
	http.Redirect(rc.response, rc.request, path, http.StatusSeeOther)
	return nil
}

func (rc *requestContext) context() context.Context { return rc.request.Context() }

func (rc *requestContext) setFlashMessage(kind, format string, args ...interface{}) {
	a := rc.application
	message := fmt.Sprintf(format, args...)
	if err := a.flashCodec.Encode(rc.response, kind, message); err != nil {
		rc.logf("Error setting flash message %v", err)
	}
}

func (rc *requestContext) logf(format string, args ...interface{}) {
	log.Printf(rc.logPrefix+format, args...)
}

func (rc *requestContext) respond(t *templates.Template, status int, data interface{}) error {
	return t.WriteResponse(rc.response, rc.request, status, &templateContext{rc: rc, Data: data})
}
