package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grupa-repo/go-kit/problem"
)

func TestWrite(t *testing.T) {
	rr := httptest.NewRecorder()
	problem.Write(rr, http.StatusNotFound, problem.New("tasks_get_not_found", "Not Found", "task does not exist"))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if got := rr.Header().Get("Content-Type"); got != problem.ContentType {
		t.Errorf("Content-Type = %q, want %q", got, problem.ContentType)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	want := map[string]any{"title": "Not Found", "error_code": "tasks_get_not_found", "detail": "task does not exist"}
	if len(body) != len(want) {
		t.Errorf("body = %v, want exactly %v", body, want)
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("body[%q] = %v, want %v", k, body[k], v)
		}
	}
}

func TestWriteOmitsAnEmptyDetail(t *testing.T) {
	rr := httptest.NewRecorder()
	problem.Write(rr, http.StatusInternalServerError, problem.New("common_server_error", "Internal Server Error", ""))

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if _, ok := body["detail"]; ok {
		t.Errorf("an empty detail should be omitted, got %v", body)
	}
}
