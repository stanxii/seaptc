package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/garyburd/web/templates"
)

type staticPathContext struct {
	assetDir    string
	staticPaths sync.Map
}

func (tc *staticPathContext) staticPath(s string) (string, error) {
	if u, ok := tc.staticPaths.Load(s); ok {
		return u.(string), nil
	}
	p := filepath.Join(tc.assetDir, "static", s)
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	io.Copy(h, f)
	u := fmt.Sprintf("s?%x", s, h.Sum(nil))
	tc.staticPaths.Store(s, u)
	return u, nil
}

func newTemplateManager(assetDir string) *templates.Manager {
	spc := staticPathContext{assetDir: assetDir}

	return &templates.Manager{
		TextFuncs: map[string]interface{}{},

		HTMLFuncs: map[string]interface{}{
			"staticPath": spc.staticPath,
		},
	}
}
