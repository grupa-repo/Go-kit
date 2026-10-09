// Package apperr marks the errors whose text was written for API callers.
//
// A handler must never put err.Error() into a response: the chain under it can
// carry SQL, driver text or Go struct names, and anyone who later wraps the
// error with %w adds to what leaks. Instead, an error that is meant to be read
// by the caller is created here, and Message is the only way its text reaches
// a response. Everything else is logged and answered with a fixed message.
package apperr

import (
	"errors"
	"fmt"
)

// Error is an error whose message is safe to return to an API caller.
type Error struct {
	msg   string
	cause error
}

func (e *Error) Error() string { return e.msg }
func (e *Error) Unwrap() error { return e.cause }

// New returns a caller-facing error. Like errors.New, each call returns a
// distinct value, so the result works as a sentinel with errors.Is.
func New(msg string) error { return &Error{msg: msg} }

// Newf is New with formatting. Format only values the caller supplied or may
// see; never another error.
func Newf(format string, args ...any) error {
	return &Error{msg: fmt.Sprintf(format, args...)}
}

// Wrap returns a caller-facing error that also matches cause under errors.Is.
// msg is the whole message; cause's text is not appended.
func Wrap(cause error, msg string) error { return &Error{msg: msg, cause: cause} }

// Message returns the text of the outermost caller-facing error in err's
// chain, or "" when there is none. An empty detail is omitted from the
// response, leaving the title and error_code to describe the failure.
func Message(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.msg
	}
	return ""
}
