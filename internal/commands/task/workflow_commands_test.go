package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestValidateWorkflowArgsRejectsInvalidInputBeforeMutation(t *testing.T) {
	tests := [][]string{
		{"unknown", "task-1"},
		{"complete", "task-1", "not-a-work-item"},
		{"attempt", "task-1", "checkpoint", "", "running"},
		{"evidence", "task-1", "e-1", "wi-0001", "unknown", "passed"},
		{"gate", "task-1", "gate-1", "open"},
	}
	for _, args := range tests {
		if err := validateWorkflowArgs(args[0], args); err == nil {
			t.Fatalf("expected validation error for %#v", args)
		}
	}
}

func TestValidateWorkflowArgsAcceptsControlledOperations(t *testing.T) {
	tests := [][]string{
		{"plan", "task-1"},
		{"attempt", "task-1", "start", "wi-0001", "attempt-1"},
		{"attempt", "task-1", "checkpoint", "attempt-1", "paused"},
		{"evidence", "task-1", "e-1", "wi-0001", "static-review", "passed"},
		{"gate", "task-1", "gate-1", "resolved"},
		{"complete", "task-1", "wi-0001"},
	}
	for _, args := range tests {
		if err := validateWorkflowArgs(args[0], args); err != nil {
			t.Fatalf("unexpected validation error for %#v: %v", args, err)
		}
	}
}

func TestParseWorkflowRunArgs(t *testing.T) {
	for _, test := range []struct {
		args    []string
		execute bool
		valid   bool
	}{
		{args: []string{"run", "task-1"}, valid: true},
		{args: []string{"run", "task-1", "--execute"}, execute: true, valid: true},
		{args: []string{"run", "task-1", "--unknown"}},
	} {
		execute, primary, err := parseWorkflowRunArgs(test.args)
		if test.valid && (err != nil || execute != test.execute || primary) {
			t.Fatalf("parse %#v = execute:%t primary:%t, %v", test.args, execute, primary, err)
		}
		if !test.valid && err == nil {
			t.Fatalf("expected parse failure for %#v", test.args)
		}
	}
}

func TestParseWorkflowRunArgsPrimaryRequiresExecution(t *testing.T) {
	if _, _, err := parseWorkflowRunArgs([]string{"run", "task-1", "--primary"}); err == nil {
		t.Fatal("expected --primary without --execute to fail")
	}
}

func TestParseWorkflowRunArgsSupportsPrimaryOptOut(t *testing.T) {
	execute, primary, err := parseWorkflowRunArgs([]string{"run", "task-1", "--execute", "--primary"})
	if err != nil || !execute || !primary {
		t.Fatalf("parse returned execute=%v primary=%v err=%v", execute, primary, err)
	}
}

func TestEnsureAutomatedWorkspaceRejectsUnassignedTask(t *testing.T) {
	_, err := ensureAutomatedWorkspace("task-1", taskx.TaskMeta{ID: "task-1", WorkspaceKind: "unassigned"}, false)
	if err == nil || !strings.Contains(err.Error(), "workspace is unassigned") {
		t.Fatalf("expected unassigned workspace rejection, got %v", err)
	}
}

func TestEnsureAutomatedWorkspaceRejectsPrimaryOptOutForIsolatedTask(t *testing.T) {
	_, err := ensureAutomatedWorkspace("task-1", taskx.TaskMeta{ID: "task-1", WorkspaceKind: "isolated"}, true)
	if err == nil || !strings.Contains(err.Error(), "requires Task task-1") {
		t.Fatalf("expected primary opt-out rejection, got %v", err)
	}
}

func TestWorkflowSyncCompletesCheckedItemWithoutProjectingArtifacts(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	const id = "task-1"
	if err := os.MkdirAll(taskx.TaskDir(id), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := taskx.TaskMeta{ID: id, Type: "task", Worktree: ".", WorkspaceKind: "primary"}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath(id), meta); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(taskx.TaskDir(id), "tasks.md")
	if err := os.WriteFile(path, []byte("# Tasks\n\n- [x] 1.1 done\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeTasks, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeMeta, err := os.ReadFile(taskx.TaskMetaPath(id))
	if err != nil {
		t.Fatal(err)
	}

	if err := runWorkflowCommand([]string{"sync", id}); err != nil {
		t.Fatal(err)
	}
	state, err := workflow.NewStore("").Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if got := state.WorkItems[0].State; got != workflow.WorkItemCompleted {
		t.Fatalf("work item state = %q, want %q", got, workflow.WorkItemCompleted)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(beforeTasks) {
		t.Fatalf("tasks.md was projected: got %q, want %q", content, beforeTasks)
	}
	afterMeta, err := os.ReadFile(taskx.TaskMetaPath(id))
	if err != nil {
		t.Fatal(err)
	}
	if string(afterMeta) != string(beforeMeta) {
		t.Fatalf("task.toml was projected: got %q, want %q", afterMeta, beforeMeta)
	}
}
