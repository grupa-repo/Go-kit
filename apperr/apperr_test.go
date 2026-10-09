package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/grupa-repo/go-kit/apperr"
)

func TestMessage(t *testing.T) {
	sentinel := apperr.New("group not found")
	forbidden := apperr.New("forbidden")
	notAdmin := apperr.Wrap(forbidden, "forbidden: caller is not an admin of the group")

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"plain error stays private", errors.New(`pq: relation "task" does not exist`), ""},
		{"wrapped plain error stays private", fmt.Errorf("failed to create task: %w", errors.New("connection refused")), ""},
		{"caller-facing error", sentinel, "group not found"},
		{"wrapping text around it is dropped", fmt.Errorf("delete group 12: %w", sentinel), "group not found"},
		{"outermost caller-facing error wins", fmt.Errorf("remove member: %w", notAdmin), "forbidden: caller is not an admin of the group"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := apperr.Message(c.err); got != c.want {
				t.Errorf("Message() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestWrapMatchesItsCause(t *testing.T) {
	forbidden := apperr.New("forbidden")
	notAdmin := apperr.Wrap(forbidden, "forbidden: caller is not an admin of the group")

	if !errors.Is(notAdmin, forbidden) {
		t.Error("a wrapped error should match its cause")
	}
	if errors.Is(forbidden, notAdmin) {
		t.Error("the cause should not match the error that wraps it")
	}
	if errors.Is(apperr.New("forbidden"), forbidden) {
		t.Error("two errors with the same text should stay distinct sentinels")
	}
}
