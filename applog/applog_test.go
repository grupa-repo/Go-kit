package applog_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grupa-repo/go-kit/applog"
	"github.com/grupa-repo/go-kit/requestid"
	"github.com/rs/zerolog"
)

// line decodes the single JSON line a logger wrote.
func line(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("output is not one JSON line: %v\n%s", err, buf.String())
	}
	return m
}

func TestParseLevel(t *testing.T) {
	cases := map[string]zerolog.Level{
		"debug":    zerolog.DebugLevel,
		"info":     zerolog.InfoLevel,
		"warn":     zerolog.WarnLevel,
		"error":    zerolog.ErrorLevel,
		"fatal":    zerolog.FatalLevel,
		"disabled": zerolog.Disabled,
		"":         zerolog.InfoLevel,
		"verbose":  zerolog.InfoLevel,
		"WARN":     zerolog.InfoLevel,
	}
	for in, want := range cases {
		if got := applog.ParseLevel(in); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNewWritesJSONWithLevelMessageAndTime(t *testing.T) {
	var buf bytes.Buffer
	applog.New(applog.Options{Output: &buf}).Info().Str("k", "v").Msg("hello")

	m := line(t, &buf)
	if m["level"] != "info" || m["message"] != "hello" || m["k"] != "v" {
		t.Errorf("unexpected line: %v", m)
	}
	for _, field := range []string{"time", "caller"} {
		if _, ok := m[field]; !ok {
			t.Errorf("line has no %q field: %v", field, m)
		}
	}
}

func TestLevelFiltersLowerEvents(t *testing.T) {
	var buf bytes.Buffer
	l := applog.New(applog.Options{Level: "warn", Output: &buf})

	l.Info().Msg("dropped")
	if buf.Len() != 0 {
		t.Fatalf("info written at level warn: %s", buf.String())
	}
	l.Warn().Msg("kept")
	if m := line(t, &buf); m["level"] != "warn" {
		t.Errorf("unexpected line: %v", m)
	}
}

func TestDisabledWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	applog.New(applog.Options{Level: "disabled", Output: &buf}).Error().Msg("dropped")
	if buf.Len() != 0 {
		t.Errorf("wrote at level disabled: %s", buf.String())
	}
}

func TestLoggersAreIndependent(t *testing.T) {
	var quiet, loud bytes.Buffer
	applog.New(applog.Options{Level: "error", Output: &quiet}).Info().Msg("x")
	applog.New(applog.Options{Level: "debug", Output: &loud}).Debug().Msg("x")
	if quiet.Len() != 0 || loud.Len() == 0 {
		t.Errorf("second New did not get its own level: quiet=%q loud=%q", quiet.String(), loud.String())
	}
}

func TestConsoleIsNotJSON(t *testing.T) {
	var buf bytes.Buffer
	applog.New(applog.Options{Console: true, Output: &buf}).Info().Msg("hello")
	out := buf.String()
	if !strings.Contains(out, "hello") || strings.HasPrefix(out, "{") {
		t.Errorf("console output = %q", out)
	}
}

type recordingHook struct{ got []string }

func (h *recordingHook) Run(_ *zerolog.Event, level zerolog.Level, msg string) {
	h.got = append(h.got, level.String()+":"+msg)
}

func TestHooksSeeOnlyEventsThatPassLevel(t *testing.T) {
	var buf bytes.Buffer
	h := &recordingHook{}
	l := applog.New(applog.Options{Level: "error", Output: &buf, Hooks: []zerolog.Hook{h}})

	l.Info().Msg("quiet")
	l.Error().Msg("boom")

	if len(h.got) != 1 || h.got[0] != "error:boom" {
		t.Errorf("hook saw %v, want [error:boom]", h.got)
	}
}

func TestWithReqID(t *testing.T) {
	var buf bytes.Buffer
	l := applog.New(applog.Options{Output: &buf})

	plain := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, id := l.WithReqID(plain); id != "" {
		t.Errorf("id = %q for a request without one", id)
	}

	req := plain.WithContext(requestid.NewContext(plain.Context(), "abc-123"))
	z, id := l.WithReqID(req)
	if id != "abc-123" {
		t.Fatalf("id = %q, want abc-123", id)
	}
	z.Info().Msg("handled")
	if m := line(t, &buf); m[requestid.Header] != "abc-123" {
		t.Errorf("line does not carry the request id: %v", m)
	}
}
