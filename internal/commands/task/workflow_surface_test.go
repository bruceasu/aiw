package task

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// This is a deliberate discovery surface: low-level lifecycle adapters remain
// callable, but only these operations are offered in shell completion.
var publicWorkflowSurface = []string{
	"plan", "sync", "advance", "run", "supervise", "recommend-routing",
	"focused-test", "delivery", "local-merge", "delivery-failed", "report",
	"diagnose", "recover", "repair", "repair-metadata", "help",
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

	workflowSource := read("internal/commands/task/workflow_commands.go")
	help := captureWorkflowHelp(t)
	completion := read("internal/commands/completion/completion.go")
	completionBlock := regexp.MustCompile(`(?s)var workflowCommands = \[\]string\{(.*?)\}`).FindStringSubmatch(completion)
	if len(completionBlock) != 2 {
		t.Fatal("completion workflowCommands list not found")
	}
	completionNames := regexp.MustCompile(`"([^\"]+)"`).FindAllStringSubmatch(completionBlock[1], -1)
	var actualCompletion []string
	for _, match := range completionNames {
		actualCompletion = append(actualCompletion, match[1])
	}
	if strings.Join(actualCompletion, " ") != strings.Join(publicWorkflowSurface, " ") {
		t.Errorf("completion surface = %q; want %q", actualCompletion, publicWorkflowSurface)
	}

	for _, surface := range []struct {
		name string
		text string
	}{
		{"help", help},
		{"README", read("README.md")},
		{"command inventory", read("openspec/changes/command-help-consistency/command-inventory.md")},
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
