package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
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
	if err := taskworkflow.EnsureRunnerHandoff("task-1", request, false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(task.RuntimeTaskDir("task-1"), "artifacts", "handoff.md")
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), "Work Item: wi-0001") {
		t.Fatalf("generated handoff = %q, %v", content, err)
	}
	if err := os.WriteFile(path, []byte("operator context"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := taskworkflow.EnsureRunnerHandoff("task-1", request, false); err != nil {
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
	if err := taskworkflow.EnsureRunnerHandoff("task-1", first, false); err != nil {
		t.Fatal(err)
	}
	if err := taskworkflow.EnsureRunnerHandoff("task-1", second, false); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(task.RuntimeTaskDir("task-1"), "artifacts", "handoff.md"))
	if err != nil || !strings.Contains(string(content), "Work Item: wi-0002") {
		t.Fatalf("refreshed handoff = %q, %v", content, err)
	}
}

func TestEnsureRunnerHandoffDefersSupervisedValidation(t *testing.T) {
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
	if err := taskworkflow.EnsureRunnerHandoff("task-1", request, true); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(task.RuntimeTaskDir("task-1"), "artifacts", "handoff.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Do not run compiler commands or tests") || !strings.Contains(string(content), "supervisor runs the frozen Compile Plan") {
		t.Fatalf("supervised handoff = %q", content)
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
	meta := task.TaskMeta{ID: "task-1", Worktree: worktree, WorkspaceKind: "isolated"}
	if err := os.MkdirAll(task.TaskDir("task-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := task.WriteTaskMeta(task.TaskMetaPath("task-1"), meta); err != nil {
		t.Fatal(err)
	}
	got, err := workflowChecklistPath("task-1")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(worktree, task.TaskDir("task-1"), "tasks.md")
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
	if err := os.MkdirAll(task.TaskDir("task-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := task.TaskMeta{ID: "task-1", Status: "TODO", Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged", Session: "session-1"}
	if err := task.WriteTaskMeta(task.TaskMetaPath("task-1"), meta); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(task.TaskDir("task-1"), "tasks.md"), []byte("- [ ] 1.1 Implement runner\n"), 0o644); err != nil {
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

func TestProjectWorkflowStatePreservesCoreWhenSnapshotWriteFails(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	meta := task.TaskMeta{ID: "task-1", Status: "TODO", Worktree: ".", WorkspaceKind: "primary", Delivery: "pending"}
	if err := os.MkdirAll(task.TaskDir("task-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := task.WriteTaskMeta(task.TaskMetaPath("task-1"), meta); err != nil {
		t.Fatal(err)
	}
	state := workflow.NewCompatibleRuntime(
		workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary},
		workflow.PlanningReady,
		workflow.DeliveryMerged,
	)
	state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemCompleted}}
	state.Summary = workflow.DeriveSummary(state)
	store := workflow.NewStore("")
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(task.TaskMetaPath("task-1")); err != nil {
		t.Fatal(err)
	}
	if err := projectWorkflowState("task-1", state); err == nil {
		t.Fatal("missing metadata snapshot was accepted")
	}
	persisted, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Delivery != workflow.DeliveryMerged || workflow.DeriveSummary(persisted).Status != workflow.TaskDone {
		t.Fatalf("snapshot failure changed Core: %+v", persisted)
	}

	if err := task.WriteTaskMeta(task.TaskMetaPath("task-1"), meta); err != nil {
		t.Fatal(err)
	}
	if err := projectWorkflowState("task-1", persisted); err != nil {
		t.Fatalf("projection retry: %v", err)
	}
	projected, err := task.ReadTaskMeta(task.TaskMetaPath("task-1"))
	if err != nil {
		t.Fatal(err)
	}
	if projected.Status != string(workflow.TaskDone) || projected.Delivery != string(workflow.DeliveryMerged) {
		t.Fatalf("retry did not project Core: %+v", projected)
	}
}
