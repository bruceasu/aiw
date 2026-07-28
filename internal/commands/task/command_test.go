package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/taskx"
)

func TestParseInitOptionsRequiresPromptsWhenUsingTemplate(t *testing.T) {
	_, err := parseInitOptions([]string{"--template", "go"})
	if err == nil {
		t.Fatal("expected template without --prompts to fail")
	}
}

func TestParsePromptOptionsRejectsListWithMerge(t *testing.T) {
	_, err := parsePromptOptions([]string{"list", "--merge"})
	if err == nil {
		t.Fatal("expected prompts list with merge to fail")
	}
}

func TestParseArchiveOptionsFinalizeEnablesAllFlags(t *testing.T) {
	opts, err := parseArchiveOptions([]string{"--finalize"})
	if err != nil {
		t.Fatalf("parse archive options: %v", err)
	}
	if !opts.Push || !opts.CleanupWT || !opts.DeleteBranch {
		t.Fatalf("expected finalize to enable all flags, got %+v", opts)
	}
}

func TestMergeSpecContentAppendsOnce(t *testing.T) {
	existing := "# ai-support Specification\n"
	incoming := "# File Operations Specification\n\n### Requirement: Detect supported text encodings\n"
	first := mergeSpecContent(existing, incoming, "file-operations")
	if !strings.Contains(first, "archived spec: file-operations") {
		t.Fatalf("expected merged content to include archived marker, got %q", first)
	}
	second := mergeSpecContent(first, incoming, "file-operations")
	if second != first {
		t.Fatalf("expected merge to be idempotent")
	}
}

func TestArchiveTaskSyncsLinkedSpecsIntoGlobalSpecs(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	sourceSpec := []byte("# Demo Spec\nsource version\n")
	if err := os.MkdirAll(filepath.Join(taskx.TaskDir("T-1"), "specs", "file-operations"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(taskx.SpecsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskx.TaskDir("T-1"), "specs", "file-operations", "spec.md"), sourceSpec, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskx.TaskDir("T-1"), "tasks.md"), []byte("# TODO\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := taskx.TaskMeta{
		ID:      "T-1",
		Type:    "task",
		Status:  "DONE",
		Created: taskx.Today(),
		Updated: taskx.Today(),
		Specs:   []string{"file-operations"},
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("T-1"), meta); err != nil {
		t.Fatal(err)
	}

	if err := archiveTask("T-1", ArchiveOptions{}); err != nil {
		t.Fatalf("archive task: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(taskx.SpecsDir, "ai-support", "spec.md"))
	if err != nil {
		t.Fatalf("read synced global spec: %v", err)
	}
	if !strings.Contains(string(got), "source version") {
		t.Fatalf("global spec missing merged content: %q", got)
	}
}
