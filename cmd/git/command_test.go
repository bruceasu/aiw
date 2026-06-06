package git

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestDispatchWithHelpReturnsNil(t *testing.T) {
	out := captureOutput(t, func() {
		if err := Dispatch([]string{"help"}); err != nil {
			t.Fatalf("dispatch help: %v", err)
		}
	})
	if !strings.Contains(out, "aiw git") {
		t.Fatalf("expected help output, got:\n%s", out)
	}
}

func TestDispatchUnknownCommandReturnsError(t *testing.T) {
	err := Dispatch([]string{"definitely-unknown"})
	if err == nil {
		t.Fatal("expected unknown command to return error")
	}
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	os.Stdout = w
	os.Stderr = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	done := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close pipe reader: %v", err)
	}

	return <-done
}
