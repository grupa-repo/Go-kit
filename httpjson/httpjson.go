// Package httpjson reads and writes JSON request and response bodies.
package httpjson

import (
	"encoding/json"
	"errors"
	"net/http"
)

// MaxBodyBytes is the largest request body Decode will read.
const MaxBodyBytes = 1 << 20 // 1 MiB

var (
	// ErrNoBody means the request carried no body at all.
	ErrNoBody = errors.New("httpjson: missing request body")
	// ErrTooLarge means the body was longer than MaxBodyBytes.
	ErrTooLarge = errors.New("httpjson: request body too large")
)

// Write sends v as a JSON response with the given status.
func Write(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// Decode reads the request body into v.
//
// It returns ErrNoBody for a request without one and ErrTooLarge past
// MaxBodyBytes; any other error is the decoder's. The decoder's text names Go
// struct fields, so log it and answer with a fixed message.
func Decode(r *http.Request, v any) error {
	if r.Body == nil || r.Body == http.NoBody {
		return ErrNoBody
	}
	err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, MaxBodyBytes)).Decode(v)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return ErrTooLarge
	}
	return err
}
