// Package httpserver runs an HTTP server that stops cleanly: when its context
// is cancelled it stops accepting connections, lets in-flight requests finish,
// and only then returns.
//
// A host that restarts a process sends SIGTERM and follows it with SIGKILL
// after a grace period. A server that ignores the first signal has every
// in-flight request cut off by the second.
package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Defaults for the zero value of Server.
const (
	// DefaultShutdownTimeout leaves room inside a 30-second grace period to
	// exit before being killed.
	DefaultShutdownTimeout = 10 * time.Second
	// DefaultReadHeaderTimeout stops a connection that opens and then sends
	// nothing from being held open indefinitely.
	DefaultReadHeaderTimeout = 10 * time.Second
)

// Server is the configuration for one run. Addr and Handler are required.
type Server struct {
	Addr    string
	Handler http.Handler

	// ReadHeaderTimeout bounds how long a client may take to send its request
	// headers. It covers a WebSocket handshake but not the connection after
	// the upgrade. Zero means DefaultReadHeaderTimeout.
	ReadHeaderTimeout time.Duration

	// ShutdownTimeout bounds the drain of in-flight requests once the context
	// is cancelled. Zero means DefaultShutdownTimeout.
	ShutdownTimeout time.Duration

	// OnListening, if set, is called once the listener is open, with the
	// address it is bound to.
	OnListening func(addr net.Addr)

	// OnShutdown, if set, is called when the context is cancelled, before the
	// drain starts.
	OnShutdown func()
}

// Run listens on s.Addr and serves until ctx is cancelled, then drains and
// returns. It returns nil after a clean drain, the listen or serve error if
// the server stopped by itself, and context.DeadlineExceeded if requests were
// still running when ShutdownTimeout expired.
//
// Shutdown does not wait for hijacked connections such as WebSockets; those
// close when the process exits unless the caller closes them first.
func (s Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	return s.Serve(ctx, ln)
}

// Serve is Run on a listener the caller opened. It closes ln before returning.
func (s Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           s.Handler,
		ReadHeaderTimeout: orDefault(s.ReadHeaderTimeout, DefaultReadHeaderTimeout),
	}
	if s.OnListening != nil {
		s.OnListening(ln.Addr())
	}

	serveErr := make(chan error, 1)
	go func() {
		err := srv.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		if s.OnShutdown != nil {
			s.OnShutdown()
		}
		// Not derived from ctx, which is already cancelled.
		drainCtx, cancel := context.WithTimeout(context.Background(), orDefault(s.ShutdownTimeout, DefaultShutdownTimeout))
		defer cancel()
		return srv.Shutdown(drainCtx)
	}
}

// SignalContext returns a context that is cancelled on SIGINT or SIGTERM, the
// signals an operator and a process manager use to ask for a stop. Call stop
// to release the signal handler.
func SignalContext(parent context.Context) (ctx context.Context, stop context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

func orDefault(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}
