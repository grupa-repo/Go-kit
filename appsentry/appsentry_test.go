package appsentry_test

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/grupa-repo/go-kit/applog"
	"github.com/grupa-repo/go-kit/appsentry"
	"github.com/rs/zerolog"
)

// testDSN is well-formed and points nowhere; the capturing transport means
// nothing is sent.
const testDSN = "https://public@sentry.invalid/1"

// capture is a transport that keeps events instead of sending them.
type capture struct {
	mu      sync.Mutex
	events  []*sentry.Event
	flushes int
}

func (c *capture) Configure(sentry.ClientOptions) {}
func (c *capture) Close()                         {}

func (c *capture) SendEvent(e *sentry.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *capture) Flush(time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flushes++
	return true
}

func (c *capture) FlushWithContext(context.Context) bool { return c.Flush(0) }

func (c *capture) sent() []*sentry.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*sentry.Event(nil), c.events...)
}

// start enables reporting into a capturing transport, and turns it off again
// when the test ends so no other test inherits a client.
func start(t *testing.T, o appsentry.Options) *capture {
	t.Helper()
	c := &capture{}
	appsentry.SetTransport(c)
	t.Cleanup(func() {
		appsentry.SetTransport(nil)
		sentry.CurrentHub().BindClient(nil)
	})
	o.DSN = testDSN
	_, enabled, err := appsentry.Init(o)
	if err != nil || !enabled {
		t.Fatalf("Init: enabled=%v err=%v", enabled, err)
	}
	return c
}

func TestInitWithoutDSNIsDisabled(t *testing.T) {
	flush, enabled, err := appsentry.Init(appsentry.Options{})
	if enabled || err != nil {
		t.Fatalf("enabled=%v err=%v, want false and nil", enabled, err)
	}
	flush() // must be callable
}

func TestInitRejectsMalformedDSN(t *testing.T) {
	flush, enabled, err := appsentry.Init(appsentry.Options{DSN: "not a dsn"})
	if err == nil || enabled {
		t.Fatalf("enabled=%v err=%v, want an error", enabled, err)
	}
	flush()
}

func TestHookDoesNothingBeforeInit(t *testing.T) {
	var buf bytes.Buffer
	l := applog.New(applog.Options{Output: &buf, Hooks: []zerolog.Hook{appsentry.Hook()}})
	l.Error().Msg("boom") // must not panic
	if buf.Len() == 0 {
		t.Error("the line itself was not written")
	}
}

func TestHookReportsErrorsAtTheirSeverity(t *testing.T) {
	c := start(t, appsentry.Options{Environment: "qa", Release: "abc123", ServerName: "svc"})
	var buf bytes.Buffer
	l := applog.New(applog.Options{Level: "debug", Output: &buf, Hooks: []zerolog.Hook{appsentry.Hook()}})

	l.Info().Msg("fine")
	l.Warn().Msg("hmm")
	l.Error().Msg("boom")

	events := c.sent()
	if len(events) != 1 {
		t.Fatalf("%d events sent, want 1 (the error only)", len(events))
	}
	e := events[0]
	if e.Level != sentry.LevelError {
		t.Errorf("level = %q, want error", e.Level)
	}
	if len(e.Exception) == 0 || e.Exception[0].Value != "boom" {
		t.Errorf("exception = %+v, want value boom", e.Exception)
	}
	if e.Environment != "qa" || e.Release != "abc123" || e.ServerName != "svc" {
		t.Errorf("environment=%q release=%q server=%q", e.Environment, e.Release, e.ServerName)
	}
}

func TestReleaseFallsBackToThePlatform(t *testing.T) {
	t.Setenv("HEROKU_SLUG_COMMIT", "")
	t.Setenv("SOURCE_VERSION", "")
	if got := appsentry.PlatformRelease(); got != "" {
		t.Errorf("PlatformRelease() = %q with nothing set", got)
	}
	t.Setenv("SOURCE_VERSION", "build")
	if got := appsentry.PlatformRelease(); got != "build" {
		t.Errorf("PlatformRelease() = %q, want build", got)
	}
	t.Setenv("HEROKU_SLUG_COMMIT", "slug")
	if got := appsentry.PlatformRelease(); got != "slug" {
		t.Errorf("PlatformRelease() = %q, want slug", got)
	}

	c := start(t, appsentry.Options{})
	sentry.CaptureMessage("x")
	if events := c.sent(); len(events) != 1 || events[0].Release != "slug" {
		t.Errorf("events = %+v, want one with release slug", events)
	}
}

func TestCredentialHeadersAreScrubbed(t *testing.T) {
	c := start(t, appsentry.Options{ScrubHeaders: []string{"X-Api-Key"}})

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.AddEventProcessor(func(e *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			e.Request = &sentry.Request{
				Cookies: "session=1",
				Headers: map[string]string{
					"Authorization":       "Bearer t",
					"proxy-authorization": "Basic p",
					"Cookie":              "session=1",
					"x-api-key":           "k",
					"User-Agent":          "ua",
				},
			}
			return e
		})
		sentry.CaptureMessage("x")
	})

	events := c.sent()
	if len(events) != 1 {
		t.Fatalf("%d events sent, want 1", len(events))
	}
	req := events[0].Request
	if req.Cookies != "" {
		t.Errorf("cookies = %q, want empty", req.Cookies)
	}
	if len(req.Headers) != 1 || req.Headers["User-Agent"] != "ua" {
		t.Errorf("headers = %v, want only User-Agent", req.Headers)
	}
}

// A fatal event exits the process before a deferred flush can run, so the hook
// has to flush for itself. The hook is run directly: logging at fatal would
// end the test binary.
func TestHookFlushesOnFatal(t *testing.T) {
	c := start(t, appsentry.Options{})

	appsentry.Hook().Run(nil, zerolog.ErrorLevel, "boom")
	if c.flushes != 0 {
		t.Fatalf("flushed %d times on an error, want 0", c.flushes)
	}
	appsentry.Hook().Run(nil, zerolog.FatalLevel, "dying")
	if c.flushes != 1 {
		t.Errorf("flushed %d times on a fatal, want 1", c.flushes)
	}
	events := c.sent()
	if len(events) != 2 || events[1].Level != sentry.LevelFatal {
		t.Errorf("events = %d, last level %q; want 2 and fatal", len(events), events[len(events)-1].Level)
	}
}
