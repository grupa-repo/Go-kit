package appsentry

import "github.com/getsentry/sentry-go"

// SetTransport makes Init hand events to t instead of sending them.
func SetTransport(t sentry.Transport) { transport = t }
