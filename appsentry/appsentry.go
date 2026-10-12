// Package appsentry sets up Sentry error reporting and forwards logged errors
// to it. With no DSN it does nothing, so a local run needs no configuration.
package appsentry

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// flushTimeout bounds how long a process waits for queued events on its way
// out.
const flushTimeout = 2 * time.Second

// alwaysScrubbed are the request headers that never leave the process,
// whatever Options.ScrubHeaders says.
var alwaysScrubbed = []string{"Authorization", "Proxy-Authorization", "Cookie"}

// Options configures Init.
type Options struct {
	// DSN is the project's ingest address. Empty disables reporting.
	DSN string

	// Environment labels events. Pass one: left empty, Sentry files the
	// events under "production".
	Environment string

	// Release labels events with the running version. Empty means
	// PlatformRelease().
	Release string

	// ServerName replaces the host name on events. A dyno's host name is
	// random, so a fixed name is what makes events groupable by service.
	ServerName string

	TracesSampleRate float64

	// Debug makes the SDK log what it does to stderr.
	Debug bool

	// ScrubHeaders are request headers removed from every event, in addition
	// to Authorization, Proxy-Authorization and Cookie. Name every header
	// that carries a credential. Matching ignores case.
	ScrubHeaders []string
}

// transport replaces the SDK's HTTP transport in tests.
var transport sentry.Transport

// Init configures the process-wide Sentry client. The returned flush sends
// what is still queued and should be deferred by the caller; it is never nil.
// enabled is false, with a nil error, when o.DSN is empty.
func Init(o Options) (flush func(), enabled bool, err error) {
	if o.DSN == "" {
		return func() {}, false, nil
	}

	release := o.Release
	if release == "" {
		release = PlatformRelease()
	}

	err = sentry.Init(sentry.ClientOptions{
		Dsn:              o.DSN,
		Environment:      o.Environment,
		Release:          release,
		ServerName:       o.ServerName,
		TracesSampleRate: o.TracesSampleRate,
		Debug:            o.Debug,
		AttachStacktrace: true,
		BeforeSend:       scrubber(o.ScrubHeaders),
		Transport:        transport,
	})
	if err != nil {
		return func() {}, false, err
	}
	return func() { sentry.Flush(flushTimeout) }, true, nil
}

// PlatformRelease returns the commit the platform says is running, or "" when
// it does not say. Heroku sets HEROKU_SLUG_COMMIT at run time when dyno
// metadata is enabled, and SOURCE_VERSION during a build.
func PlatformRelease() string {
	for _, name := range []string{"HEROKU_SLUG_COMMIT", "SOURCE_VERSION"} {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

// Hook returns a zerolog hook that reports every event at error level or above
// to Sentry, at the matching severity. It does nothing until Init has enabled
// reporting, so it is safe to attach unconditionally and before Init.
//
// The report carries the event's message only. A hook cannot read the fields
// of the event it is given.
func Hook() zerolog.Hook { return hook{} }

type hook struct{}

func (hook) Run(_ *zerolog.Event, level zerolog.Level, msg string) {
	if level < zerolog.ErrorLevel {
		return
	}
	hub := sentry.CurrentHub().Clone()
	if hub.Client() == nil {
		return
	}
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(severity(level))
		hub.CaptureException(errors.New(msg))
	})
	// A fatal event exits the process as soon as it is written, before any
	// deferred flush runs. Without this the report of why a service died is
	// the one report that never arrives.
	if level == zerolog.FatalLevel {
		hub.Flush(flushTimeout)
	}
}

func severity(l zerolog.Level) sentry.Level {
	switch l {
	case zerolog.FatalLevel, zerolog.PanicLevel:
		return sentry.LevelFatal
	case zerolog.ErrorLevel:
		return sentry.LevelError
	case zerolog.WarnLevel:
		return sentry.LevelWarning
	default:
		return sentry.LevelInfo
	}
}

// scrubber returns the BeforeSend that strips credentials from an event's
// request before it is sent.
func scrubber(extra []string) func(*sentry.Event, *sentry.EventHint) *sentry.Event {
	names := append(append([]string{}, alwaysScrubbed...), extra...)
	return func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
		if event.Request == nil {
			return event
		}
		event.Request.Cookies = ""
		for header := range event.Request.Headers {
			for _, name := range names {
				if strings.EqualFold(header, name) {
					delete(event.Request.Headers, header)
					break
				}
			}
		}
		return event
	}
}
