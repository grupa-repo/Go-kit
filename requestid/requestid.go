// Package requestid carries one identifier per HTTP request, so every log line
// and downstream call made on its behalf can be joined back to it.
package requestid

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

// Header is the header the identifier arrives in and is echoed back on.
const Header = "X-Request-ID"

// maxLen bounds an inbound identifier. A UUID is 36 characters; anything much
// longer is not an identifier.
const maxLen = 128

type contextKey struct{}

// Middleware takes the request's identifier from Header, or generates one,
// stores it in the request context and echoes it on the response.
//
// An inbound value is used only if it is short and made of characters that are
// safe to write to a log. The value is caller-controlled and ends up on every
// log line for the request, so anything else is replaced, not trusted.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(Header)
		if !valid(id) {
			id = New()
		}
		w.Header().Set(Header, id)
		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), id)))
	})
}

// NewContext returns a copy of ctx carrying id. Use it to hand the identifier
// to work that outlives the request, such as a queued job.
func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext returns the identifier stored in ctx, or "" when there is none.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

// New returns a random version 4 UUID in its canonical text form.
func New() string {
	var b [16]byte
	// crypto/rand.Read never returns an error; it panics if the system source
	// fails.
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 9562 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func valid(id string) bool {
	if id == "" || len(id) > maxLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-', c == '_', c == '.', c == ':':
		default:
			return false
		}
	}
	return true
}
