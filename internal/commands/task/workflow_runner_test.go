package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestEnsureRunnerHandoffCreatesAndPreservesContext(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	request := &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: "."}
	if err := ensureRunnerHandoff("task-1", request); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(taskx.RuntimeTaskDir("task-1"), "artifacts", "handoff.md")
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), "Work Item: wi-0001") {
		t.Fatalf("generated handoff = %q, %v", content, err)
	}
	if err := os.WriteFile(path, []byte("operator context"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureRunnerHandoff("task-1", request); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(path)
	if err != nil || string(content) != "operator context" {
		t.Fatalf("preserved handoff = %q, %v", content, err)
	}
}

func TestEnsureRunnerHandoffRefreshesGeneratedWorkItem(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	first := &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: "."}
	second := &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0002", AttemptID: "attempt-2", Workspace: "."}
	if err := ensureRunnerHandoff("task-1", first); err != nil {
		t.Fatal(err)
	}
	if err := ensureRunnerHandoff("task-1", second); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(taskx.RuntimeTaskDir("task-1"), "artifacts", "handoff.md"))
	if err != nil || !strings.Contains(string(content), "Work Item: wi-0002") {
		t.Fatalf("refreshed handoff = %q, %v", content, err)
	}
}

func TestWorkflowChecklistPathUsesIsolatedTaskWorkspace(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	worktree := filepath.Join(tmp, "isolated")
	meta := taskx.TaskMeta{ID: "task-1", Worktree: worktree, WorkspaceKind: "isolated"}
	if err := os.MkdirAll(taskx.TaskDir("task-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("task-1"), meta); err != nil {
		t.Fatal(err)
	}
	got, err := workflowChecklistPath("task-1")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(worktree, taskx.TaskDir("task-1"), "tasks.md")
	if got != want {
		t.Fatalf("checklist path = %q, want %q", got, want)
	}
}

func TestRunWorkflowReusesPreparedAttempt(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.MkdirAll(taskx.TaskDir("task-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := taskx.TaskMeta{ID: "task-1", Status: "TODO", Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged", Session: "session-1"}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("task-1"), meta); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskx.TaskDir("task-1"), "tasks.md"), []byte("- [ ] 1.1 Implement runner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runWorkflow("task-1", false); err != nil {
		t.Fatal(err)
	}
	if err := runWorkflow("task-1", false); err != nil {
		t.Fatal(err)
	}
	state, err := workflow.NewStore("").Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Attempts) != 1 || state.Automation.PreparedRequest == nil {
		t.Fatalf("attempts=%d request=%#v", len(state.Attempts), state.Automation.PreparedRequest)
	}
}
