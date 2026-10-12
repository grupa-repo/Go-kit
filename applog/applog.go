// Package applog is the structured logger the services share: zerolog writing
// JSON to stdout, one line per event, with the request's identifier attached
// when there is one.
package applog

import (
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/grupa-repo/go-kit/requestid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// Options configures a Logger. The zero value logs JSON to stdout at info.
type Options struct {
	// Level is the lowest level written; see ParseLevel for the names.
	Level string

	// Console writes human-readable lines instead of JSON. It is for a
	// terminal. The caller decides when that applies; this package takes no
	// view on environment names.
	Console bool

	// Output is where lines go. Nil means os.Stdout.
	Output io.Writer

	// Hooks run on every event that passes Level, in order. This is how an
	// error tracker sees what is logged without this package importing one.
	Hooks []zerolog.Hook
}

// Logger is a levelled logger. Create one at startup and pass it down.
type Logger struct {
	z zerolog.Logger
}

var globals sync.Once

// New returns a Logger. Each call returns an independent one, so a test can
// ask for its own level and output.
func New(o Options) *Logger {
	// zerolog keeps these two in package variables, so they are set once for
	// the process instead of raced over by every caller.
	globals.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = time.RFC3339Nano
	})

	var out io.Writer = os.Stdout
	if o.Output != nil {
		out = o.Output
	}
	if o.Console {
		out = zerolog.ConsoleWriter{Out: out}
	}

	z := zerolog.New(out).With().Caller().Timestamp().Logger().Level(ParseLevel(o.Level))
	for _, h := range o.Hooks {
		z = z.Hook(h)
	}
	return &Logger{z: z}
}

// WithReqID returns a logger that stamps every line with the request's
// identifier, and the identifier itself. Without one in the request's context
// it returns the plain logger and "".
func (l *Logger) WithReqID(r *http.Request) (zerolog.Logger, string) {
	id := requestid.FromContext(r.Context())
	if id == "" {
		return l.z, ""
	}
	return l.z.With().Str(requestid.Header, id).Logger(), id
}

// Fatal starts an event that exits the process with status 1 once it is sent.
func (l *Logger) Fatal() *zerolog.Event { return l.z.Fatal() }

func (l *Logger) Error() *zerolog.Event { return l.z.Error() }

func (l *Logger) Warn() *zerolog.Event { return l.z.Warn() }

func (l *Logger) Info() *zerolog.Event { return l.z.Info() }

func (l *Logger) Debug() *zerolog.Event { return l.z.Debug() }

// ParseLevel maps a LOG_LEVEL value to a zerolog level: "debug", "info",
// "warn", "error", "fatal" or "disabled". Anything else, the empty string
// included, is info, so a typo makes a service chattier, never silent.
func ParseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "disabled":
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}
