package workflow

import (
	"strings"
	"testing"
)

func TestFacadeDispatchesOperationsThroughItsAdapter(t *testing.T) {
	called := ""
	facade := New(func(args []string) error {
		called = strings.Join(args, " ")
		return nil
	}, nil)
	if err := facade.Dispatch([]string{"status", "task-1"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if called != "status task-1" {
		t.Fatalf("adapter args = %q", called)
	}
}
