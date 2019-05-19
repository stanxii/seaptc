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
	"github.com/seaptc/server/data"
	"github.com/seaptc/server/datastore"

	"golang.org/x/net/xsrftoken"
	"golang.org/x/xerrors"
)

type application struct {
	devMode  bool
	dsClient *datastore.Client

	nextRequestID int64 // incremented on each request, used in log output

	config data.AppConfig

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

func newApplication(ctx context.Context, dsClient *datastore.Client, devMode bool, assetDir string, services ...applicationService) (http.Handler, error) {

	var config data.AppConfig
	if err := dsClient.GetDocTo(ctx, data.AppConfigPath, &config); err != nil {
		return nil, xerrors.Errorf("get app config: %w", err)
	}

	var hmacKeys [][]byte
	for _, k := range config.HMACKeys {
		hmacKeys = append(hmacKeys, []byte(k))
	}

	a := application{
		devMode:  devMode,
		dsClient: dsClient,
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
			path := strings.Replace(strings.TrimPrefix(m.Name, "Serve"), "_", "/", -1)
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
		rc.participantID = 0
	}

	// Clobber ids if XSRF token is not valid.
	if request.Method != "HEAD" && request.Method != "GET" {
		id := fmt.Sprintf("%s\000%d", rc.staffID, rc.participantID)
		if !xsrftoken.Valid(request.FormValue("_xsrftoken"), rc.application.config.XSRFKey, id, request.URL.Path) {
			rc.participantID = 0
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

	participantID int64
}

func (rc *requestContext) redirect(path string, statusCode int) error {
	http.Redirect(rc.response, rc.request, path, statusCode)
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
