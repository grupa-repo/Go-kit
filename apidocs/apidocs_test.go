package apidocs_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grupa-repo/go-kit/apidocs"
)

// mux adapts the standard ServeMux to apidocs.Router.
type mux struct{ *http.ServeMux }

func (m mux) Get(pattern string, h http.HandlerFunc) { m.HandleFunc("GET "+pattern, h) }

func get(m mux, path string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	m.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
	return rr
}

const spec = "openapi: 3.0.3\ninfo:\n  title: example\n"

func TestRegisterEnabled(t *testing.T) {
	m := mux{http.NewServeMux()}
	apidocs.Register(m, true, "example — API docs", []byte(spec))

	ui := get(m, apidocs.UIPath)
	if ui.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", apidocs.UIPath, ui.Code)
	}
	if ct := ui.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("UI Content-Type = %q", ct)
	}
	body := ui.Body.String()
	if !strings.Contains(body, "<title>example — API docs</title>") {
		t.Error("the page does not carry the title")
	}
	if !strings.Contains(body, "url: '"+apidocs.SpecPath+"'") {
		t.Error("the page does not point the viewer at the spec path")
	}

	doc := get(m, apidocs.SpecPath)
	if doc.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", apidocs.SpecPath, doc.Code)
	}
	if ct := doc.Header().Get("Content-Type"); ct != "application/yaml; charset=utf-8" {
		t.Errorf("spec Content-Type = %q", ct)
	}
	if doc.Body.String() != spec {
		t.Errorf("spec body = %q, want the document passed in", doc.Body.String())
	}
}

// Disabled must mean "not registered", not "registered and refusing". A 401 or
// 403 would still confirm the service has docs and name the path.
func TestRegisterDisabledServesNothing(t *testing.T) {
	m := mux{http.NewServeMux()}
	apidocs.Register(m, false, "example", []byte(spec))

	for _, p := range []string{apidocs.UIPath, apidocs.SpecPath} {
		if rr := get(m, p); rr.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", p, rr.Code)
		}
	}
}

func TestTitleIsEscaped(t *testing.T) {
	m := mux{http.NewServeMux()}
	apidocs.Register(m, true, `</title><script>alert(1)</script>`, []byte(spec))

	if body := get(m, apidocs.UIPath).Body.String(); strings.Contains(body, "<script>alert(1)</script>") {
		t.Error("the title reached the page unescaped")
	}
}
