package requestid_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/grupa-repo/go-kit/requestid"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// serve runs one request through the middleware and returns the identifier the
// handler saw and the one echoed on the response.
func serve(t *testing.T, inbound string) (seen, echoed string) {
	t.Helper()
	h := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestid.FromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if inbound != "" {
		req.Header.Set(requestid.Header, inbound)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return seen, rr.Header().Get(requestid.Header)
}

func TestMiddlewareKeepsAnInboundID(t *testing.T) {
	const id = "5f0c2a7e-3b1d-4c8e-9a6f-0d2e4b6c8a10"
	seen, echoed := serve(t, id)
	if seen != id || echoed != id {
		t.Errorf("seen %q, echoed %q, want both %q", seen, echoed, id)
	}
}

func TestMiddlewareGeneratesWhenAbsent(t *testing.T) {
	seen, echoed := serve(t, "")
	if !uuidV4.MatchString(seen) {
		t.Errorf("generated id %q is not a version 4 UUID", seen)
	}
	if echoed != seen {
		t.Errorf("echoed %q, want the id the handler saw, %q", echoed, seen)
	}
}

func TestMiddlewareReplacesAnUnsafeInboundID(t *testing.T) {
	cases := map[string]string{
		"newline":   "abc\nlevel=error forged",
		"space":     "abc def",
		"quote":     `abc"def`,
		"too long":  strings.Repeat("a", 129),
		"non-ascii": "abcé",
	}
	for name, inbound := range cases {
		t.Run(name, func(t *testing.T) {
			h := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := requestid.FromContext(r.Context()); got == inbound || !uuidV4.MatchString(got) {
					t.Errorf("handler saw %q, want a fresh UUID", got)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			// Set on the map directly: Header.Set is fine with these values,
			// but a real server would have rejected some of them earlier.
			req.Header[requestid.Header] = []string{inbound}
			h.ServeHTTP(httptest.NewRecorder(), req)
		})
	}
}

func TestMiddlewareAcceptsTheLongestID(t *testing.T) {
	id := strings.Repeat("a", 128)
	if seen, _ := serve(t, id); seen != id {
		t.Errorf("a 128-character id should be kept, got %q", seen)
	}
}

func TestFromContextWithoutAnID(t *testing.T) {
	if got := requestid.FromContext(context.Background()); got != "" {
		t.Errorf("FromContext on a bare context = %q, want \"\"", got)
	}
}

func TestNewContextRoundTrip(t *testing.T) {
	ctx := requestid.NewContext(context.Background(), "job-42")
	if got := requestid.FromContext(ctx); got != "job-42" {
		t.Errorf("FromContext = %q, want %q", got, "job-42")
	}
}

func TestNewIsUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := requestid.New()
		if !uuidV4.MatchString(id) {
			t.Fatalf("New() = %q, not a version 4 UUID", id)
		}
		if seen[id] {
			t.Fatalf("New() repeated %q", id)
		}
		seen[id] = true
	}
}
