package httpserver_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grupa-repo/go-kit/httpserver"
)

// start runs s on a free local port and returns its base URL, a cancel func
// for its context, and a channel that receives Serve's result.
func start(t *testing.T, s httpserver.Server) (url string, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	result := make(chan error, 1)
	go func() { result <- s.Serve(ctx, ln) }()
	return "http://" + ln.Addr().String(), cancel, result
}

func wait(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("server did not return")
		return nil
	}
}

func TestServesThenStopsCleanlyOnCancel(t *testing.T) {
	var listening, shutdown atomic.Bool
	url, cancel, done := start(t, httpserver.Server{
		Handler:     http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "ok") }),
		OnListening: func(net.Addr) { listening.Store(true) },
		OnShutdown:  func() { shutdown.Store(true) },
	})

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
	if !listening.Load() {
		t.Error("OnListening was not called")
	}
	if shutdown.Load() {
		t.Error("OnShutdown was called before the context was cancelled")
	}

	cancel()
	if err := wait(t, done); err != nil {
		t.Errorf("Serve returned %v, want nil after a clean drain", err)
	}
	if !shutdown.Load() {
		t.Error("OnShutdown was not called")
	}
	if _, err := http.Get(url); err == nil {
		t.Error("the server still accepts connections after it returned")
	}
}

func TestInFlightRequestFinishesDuringDrain(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	url, cancel, done := start(t, httpserver.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-release
			_, _ = io.WriteString(w, "finished")
		}),
	})

	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			got <- result{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		got <- result{body: string(b), err: err}
	}()

	<-started
	cancel()

	// The server must still be draining: the handler has not been released.
	select {
	case err := <-done:
		t.Fatalf("Serve returned (%v) while a request was still in flight", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	r := <-got
	if r.err != nil || r.body != "finished" {
		t.Errorf("in-flight request got (%q, %v), want (\"finished\", nil)", r.body, r.err)
	}
	if err := wait(t, done); err != nil {
		t.Errorf("Serve returned %v, want nil", err)
	}
}

func TestDrainGivesUpAfterShutdownTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	url, cancel, done := start(t, httpserver.Server{
		ShutdownTimeout: 50 * time.Millisecond,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-release
		}),
	})

	go func() {
		if resp, err := http.Get(url); err == nil {
			resp.Body.Close()
		}
	}()
	<-started
	cancel()

	if err := wait(t, done); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Serve returned %v, want context.DeadlineExceeded", err)
	}
}

func TestRunReportsAListenFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	s := httpserver.Server{Addr: ln.Addr().String(), Handler: http.NotFoundHandler()}
	if err := s.Run(context.Background()); err == nil {
		t.Error("Run on an address already in use returned nil")
	}
}

func TestSignalContextStopReleasesIt(t *testing.T) {
	ctx, stop := httpserver.SignalContext(context.Background())
	if ctx.Err() != nil {
		t.Fatalf("context is done before any signal: %v", ctx.Err())
	}
	stop()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Error("stop did not cancel the context")
	}
}
