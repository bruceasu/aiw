package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	workflowcmd "aiw/internal/workflow/facade"
)

// This is a deliberate discovery surface: low-level lifecycle adapters remain
// callable, but only these operations are offered in shell completion.
var publicWorkflowSurface = []string{
	"plan", "status", "run", "supervise", "diagnose", "complete", "help",
}

func TestPublicWorkflowSurfaceStaysAligned(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", "..", ".."))
	read := func(path string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return string(content)
	}

	workflowSource := read("internal/workflow/cli/command.go")
	help := captureWorkflowHelp(t)
	completion := read("internal/commands/completion/completion.go")
	if !strings.Contains(completion, "workflowcmd.OperationNames") {
		t.Fatal("completion must consume the Workflow facade operation registry")
	}
	actualCompletion := workflowcmd.OperationNames
	if strings.Join(actualCompletion, " ") != strings.Join(publicWorkflowSurface, " ") {
		t.Errorf("completion surface = %q; want %q", actualCompletion, publicWorkflowSurface)
	}

	for _, surface := range []struct {
		name string
		text string
	}{
		{"help", help},
		{"README", read("README.md")},
	} {
		for _, operation := range publicWorkflowSurface {
			if !strings.Contains(surface.text, operation) {
				t.Errorf("%s is missing public workflow operation %q", surface.name, operation)
			}
		}
	}
	for _, operation := range publicWorkflowSurface {
		if !strings.Contains(workflowSource, `"`+operation+`"`) {
			t.Errorf("workflow dispatch source is missing operation %q", operation)
		}
	}
}

func captureWorkflowHelp(t *testing.T) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	printWorkflowHelp()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original
	var output bytes.Buffer
	if _, err := io.Copy(&output, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
