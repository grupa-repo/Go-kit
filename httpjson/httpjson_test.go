package httpjson_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grupa-repo/go-kit/httpjson"
)

type payload struct {
	Name string `json:"name"`
}

func TestWrite(t *testing.T) {
	rr := httptest.NewRecorder()
	if err := httpjson.Write(rr, http.StatusCreated, payload{Name: "groceries"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != `{"name":"groceries"}` {
		t.Errorf("body = %s", got)
	}
}

func TestDecode(t *testing.T) {
	var p payload
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"groceries"}`))
	if err := httpjson.Decode(req, &p); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if p.Name != "groceries" {
		t.Errorf("Name = %q, want groceries", p.Name)
	}
}

func TestDecodeErrors(t *testing.T) {
	t.Run("no body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		if err := httpjson.Decode(req, &payload{}); !errors.Is(err, httpjson.ErrNoBody) {
			t.Errorf("err = %v, want ErrNoBody", err)
		}
	})
	t.Run("nil body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Body = nil
		if err := httpjson.Decode(req, &payload{}); !errors.Is(err, httpjson.ErrNoBody) {
			t.Errorf("err = %v, want ErrNoBody", err)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":`))
		err := httpjson.Decode(req, &payload{})
		if err == nil || errors.Is(err, httpjson.ErrNoBody) || errors.Is(err, httpjson.ErrTooLarge) {
			t.Errorf("err = %v, want the decoder's error", err)
		}
	})
	t.Run("wrong type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":5}`))
		if err := httpjson.Decode(req, &payload{}); err == nil {
			t.Error("a number in a string field should not decode")
		}
	})
	t.Run("too large", func(t *testing.T) {
		body := `{"name":"` + strings.Repeat("a", httpjson.MaxBodyBytes) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		if err := httpjson.Decode(req, &payload{}); !errors.Is(err, httpjson.ErrTooLarge) {
			t.Errorf("err = %v, want ErrTooLarge", err)
		}
	})
}
