package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	htemp "html/template"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/web/templates"
	"github.com/seaptc/server/model"
	"golang.org/x/net/xsrftoken"
)

// templateContext is the root value passed to template execute.
type templateContext struct {
	rc *requestContext

	// Data specific to each request handler.
	Data interface{}
}

func (tc *templateContext) Request() *http.Request { return tc.rc.request }
func (tc *templateContext) IsAdmin() bool          { return tc.rc.isAdmin }
func (tc *templateContext) IsStaff() bool          { return tc.rc.isStaff }

func (tc *templateContext) FlashMessage() interface{} {
	rc := tc.rc
	a := rc.application
	var result struct{ Kind, Message string }
	if err := a.flashCodec.Decode(rc.request, &result.Kind, &result.Message); err != nil {
		return nil
	}
	a.flashCodec.Encode(rc.response, nil)
	return &result
}

func (tc *templateContext) XSRFToken(path string) htemp.HTML {
	rc := tc.rc
	id := fmt.Sprintf("%s\000%s", rc.staffID, rc.participantID)
	t := xsrftoken.Generate(rc.application.config.XSRFKey, id, path)
	return htemp.HTML(fmt.Sprintf(`<input type="hidden" name="_xsrftoken" value="%s">`, t))
}

func (tc *templateContext) Sort(text string, key string) (htemp.HTML, error) {
	if key == "" {
		return "", errors.New("sort key cannot be empty string")
	}
	request := tc.rc.request
	if request.FormValue("sort") == key {
		if key[0] == '-' {
			key = key[1:]
		} else {
			key = "-" + key
		}
	}
	qp := request.URL.Query()
	qp.Set("sort", key)
	ucopy := *request.URL
	ucopy.RawQuery = qp.Encode()
	return htemp.HTML(`<a href="` + ucopy.RequestURI() + `">` + htemp.HTMLEscapeString(text) + `</a>`), nil
}

func newTemplateManager(assetDir string) *templates.Manager {
	quoteCleaner := strings.NewReplacer("\t", " ", "\r", " ", "\n", " ")
	var fileHashes sync.Map

	return &templates.Manager{
		HTMLFuncs: map[string]interface{}{
			"add": func(values ...int) int {
				result := 0
				for _, v := range values {
					result += v
				}
				return result
			},
			"fmtTime": func(layout string, t time.Time) string {
				return t.In(model.TimeLocation).Format(layout)
			},
			"truncate": func(s string, n int) string {
				i := 0
				for j := range s {
					i++
					if i > n {
						return s[:j] + "..."
					}
				}
				return s
			},
			"staticFile": func(s string) (string, error) {
				if u, ok := fileHashes.Load(s); ok {
					return u.(string), nil
				}
				p := filepath.Join(assetDir, "static", s)
				f, err := os.Open(p)
				if err != nil {
					return "", err
				}
				defer f.Close()
				h := md5.New()
				io.Copy(h, f)
				u := fmt.Sprintf("%s?%x", path.Join("/static", s), h.Sum(nil))
				fileHashes.Store(s, u)
				return u, nil
			},
			"isInvalid": func(m map[string]string, k string) string {
				if _, invalid := m[k]; invalid {
					return " is-invalid"
				}
				return ""
			},
		},
		TextFuncs: map[string]interface{}{
			"csv": func(s string) string {
				if s == "" {
					return ""
				}
				s = strings.TrimSpace(quoteCleaner.Replace(s))
				if strings.IndexAny(s, `",`) < 0 {
					return s
				}
				return `"` + strings.Replace(s, `"`, `""`, -1) + `"`
			},
		},
	}
}
