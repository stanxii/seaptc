package main

import (
	"context"
	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
)

type catalogService struct {
	templates struct {
	}
}

func (svc *catalogService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	return nil
}

func (svc *catalogService) convert(v interface{}) func(*requestContext) error {
	f := v.(func(*catalogService, *requestContext) error)
	return func(rc *requestContext) error { return f(svc, rc) }
}

var homeHTML = []byte(`<!DOCTYPE html><html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
<a href="https://seattlebsa.org/ptc">2019 Program and Training Conference</a>
</body>
</html>`)

func (svc *catalogService) Serve_(rc *requestContext) error {
	if rc.request.URL.Path != "/" {
		return httperror.ErrNotFound
	}
	rc.response.Write(homeHTML)
	return nil
}
