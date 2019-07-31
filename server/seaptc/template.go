package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	htemp "html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/garyburd/web/templates"
)

// templateContext is the root value passed to template execute.
type templateContext struct {
	rc *requestContext

	// Data specific to each request handler.
	Data interface{}
}

func (tc *templateContext) Request() *http.Request  { return tc.rc.request }
func (tc *templateContext) IsAdmin() bool           { return tc.rc.isAdmin }
func (tc *templateContext) IsStaff() bool           { return tc.rc.isStaff }
func (tc *templateContext) ParticipantName() string { return tc.rc.participantName }
func (tc *templateContext) ConferenceDate(fmt string) string {
	return tc.rc.application.conferenceDate.Format(fmt)
}

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
	return htemp.HTML(fmt.Sprintf(`<input type="hidden" name="_xsrftoken" value="%s">`,
		tc.rc.xsrfToken(path)))
}

func (tc *templateContext) Sort(text string, key string) (htemp.HTML, error) {
	if key == "" {
		return "", errors.New("sort key cannot be empty string")
	}
	var isDefault bool
	if key[0] == '!' {
		isDefault = true
		key = key[1:]
	}

	qp := tc.rc.request.URL.Query()
	sort := qp.Get("sort")
	reverse := "-"
	if isDefault && sort == "" {
		sort = key
	} else if len(sort) > 0 && sort[0] == '-' {
		reverse = ""
		sort = sort[1:]
	}

	if sort == key {
		sort = reverse + key
	} else {
		sort = key
	}

	if isDefault && sort == key {
		qp.Del("sort")
	} else {
		qp.Set("sort", sort)
	}

	ucopy := *tc.rc.request.URL
	ucopy.RawQuery = qp.Encode()
	return htemp.HTML(`<a href="` + ucopy.RequestURI() + `">` + htemp.HTMLEscapeString(text) + `</a>`), nil
}

func newTemplateManager(assetDir string) *templates.Manager {
	quoteCleaner := strings.NewReplacer("\t", " ", "\r", " ", "\n", " ")
	var fileHashes sync.Map

	return &templates.Manager{
		HTMLFuncs: map[string]interface{}{
			"args": func(values ...interface{}) []interface{} {
				return values
			},
			// add adds integers.
			"add": func(values ...int) int {
				result := 0
				for _, v := range values {
					result += v
				}
				return result
			},
			// truncate truncates s to n runes.
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
			// staticFile returns URL of static file w/ cache busting hash.
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
			// isInvalid returns Bootstrap CSS class for invalid form control if k key is present in m.
			"isInvalid": func(m map[string]string, k string) string {
				if _, invalid := m[k]; invalid {
					return " is-invalid"
				}
				return ""
			},
			// rget gets a value from v. An error is returned if v does not
			// have the key k. Use this function to detect typos in keys.
			"rget": func(v url.Values, k string) (string, error) {
				vs := v[k]
				if len(vs) == 0 {
					return "", fmt.Errorf("key %q is missing", k)
				}
				return vs[0], nil
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

func blankOrYes(v bool) string {
	if v {
		return "yes"
	}
	return ""
}
