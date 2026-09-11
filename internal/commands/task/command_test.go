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

func TestInitWorkspaceRunsOfficialSetupAfterBaseInitialization(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	previousSetup := runOfficialSetupFn
	runOfficialSetupFn = func(baseAgentsCreated bool) {
		if !baseAgentsCreated {
			t.Fatal("expected setup to receive base AGENTS creation state")
		}
		if _, err := os.Stat(taskx.ChangesDir); err != nil {
			t.Fatalf("base initialization did not complete before setup: %v", err)
		}
	}
	t.Cleanup(func() { runOfficialSetupFn = previousSetup })

	if err := initWorkspace(InitOptions{}); err != nil {
		t.Fatalf("initialize workspace: %v", err)
	}
}

func TestParseInitOptionsAcceptsNoSetup(t *testing.T) {
	opts, err := parseInitOptions([]string{"--no-setup"})
	if err != nil || !opts.SkipSetup {
		t.Fatalf("unexpected no-setup options: %+v (%v)", opts, err)
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
	if opts.Push || !opts.CleanupWT || !opts.DeleteBranch || !opts.Finalize {
		t.Fatalf("expected finalize to enable all flags, got %+v", opts)
	}
}

func TestParseNewArgsAllowsExplicitDirtyWorkspace(t *testing.T) {
	id, allowUnrelatedDirty, err := parseNewArgs([]string{"TASK-1", "--allow-unrelated-dirty"})
	if err != nil || id != "TASK-1" || !allowUnrelatedDirty {
		t.Fatalf("unexpected parse result: id=%q allowUnrelatedDirty=%v err=%v", id, allowUnrelatedDirty, err)
	}
}

func TestParseNewArgsRejectsRetiredDirtyBypass(t *testing.T) {
	_, _, err := parseNewArgs([]string{"TASK-1", "--allow-dirty"})
	if err == nil {
		t.Fatal("expected retired dirty bypass to be rejected")
	}
}

func TestDispatchTopLevelUsesCanonicalAgentCommands(t *testing.T) {
	for _, name := range []string{"turn", "chat"} {
		err := DispatchTopLevel(name, nil)
		if err == nil || !strings.Contains(err.Error(), "aiw turn|chat") {
			t.Fatalf("DispatchTopLevel(%q) error = %v, want canonical usage", name, err)
		}
	}
	if err := DispatchTopLevel("agent", []string{"next"}); err == nil {
		t.Fatal("expected retired task agent command to be rejected")
	}
}

func TestParsePromoteArgsAllowsUnrelatedDirtyByDefault(t *testing.T) {
	requirementID, taskID, allowUnrelatedDirty, err := parsePromoteArgs([]string{"requirement", "--task", "TASK-1"})
	if err != nil || requirementID != "requirement" || taskID != "TASK-1" || !allowUnrelatedDirty {
		t.Fatalf("unexpected promote arguments: requirement=%q task=%q allow=%v err=%v", requirementID, taskID, allowUnrelatedDirty, err)
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
