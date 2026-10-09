// Package problem is the error body every Grupa HTTP service returns: an
// RFC 9457 problem document carrying a stable, machine-readable error_code.
//
// Clients switch on ErrorCode, so an existing code is API contract and is
// never renamed. Detail is for people; fill it from apperr.Message or a fixed
// string, never from err.Error().
package problem

import (
	"encoding/json"
	"net/http"
)

// ContentType is the media type of an error response.
const ContentType = "application/problem+json"

// Details is the response body. The JSON names are API contract; do not
// change them.
type Details struct {
	Type      string `json:"type,omitempty"`
	Title     string `json:"title"`
	ErrorCode string `json:"error_code"`
	Detail    string `json:"detail,omitempty"`
	Instance  string `json:"instance,omitempty"`
}

// New returns a Details with the fields every error response needs. An empty
// detail is left out of the JSON.
func New(errorCode, title, detail string) Details {
	return Details{ErrorCode: errorCode, Title: title, Detail: detail}
}

// Write sends p with the given status. The encode error is dropped: by then
// the status line is on the wire and there is nothing left to tell the client.
func Write(w http.ResponseWriter, status int, p Details) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
