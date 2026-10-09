// Package apidocs serves a service's OpenAPI document and a Scalar viewer for
// it:
//
//	GET /docs               the Scalar API reference
//	GET /docs/openapi.yaml  the raw OpenAPI document
//
// Neither route exists unless the caller enables them — see Register.
package apidocs

import (
	"html"
	"net/http"
	"strings"
)

// The paths Register mounts.
const (
	UIPath   = "/docs"
	SpecPath = "/docs/openapi.yaml"
)

// scalarVersion is pinned. A floating tag changes the page whenever upstream
// publishes.
const scalarVersion = "1.67.0"

// The Scalar bundle is loaded from a CDN, not embedded: the standalone build
// is several megabytes, which is a lot to carry in every binary for a page
// production never serves. The CDN dependency exists only where the docs do.
const page = `<!doctype html>
<html>
  <head>
    <title>{{TITLE}}</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
  </head>
  <body>
    <div id="app"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@` + scalarVersion + `/dist/browser/standalone.js"></script>
    <script>
      Scalar.createApiReference('#app', { url: '` + SpecPath + `' });
    </script>
  </body>
</html>`

// Router is the part of a router Register needs. chi.Router satisfies it.
type Router interface {
	Get(pattern string, h http.HandlerFunc)
}

// Register mounts the two routes on r when enabled is true, and does nothing
// at all when it is false.
//
// Not registering is the point. A route that exists and refuses still tells a
// caller the service has docs and where they live, and it is one middleware
// mistake away from serving them. Disabled, both paths are an ordinary 404.
//
// spec is the OpenAPI document, usually embedded with go:embed. title is the
// page title. Both routes are unauthenticated: mount them outside any
// authenticated group.
func Register(r Router, enabled bool, title string, spec []byte) {
	if !enabled {
		return
	}
	ui := []byte(strings.Replace(page, "{{TITLE}}", html.EscapeString(title), 1))

	r.Get(UIPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(ui)
	})
	r.Get(SpecPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(spec)
	})
}
