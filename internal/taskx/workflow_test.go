package taskx_test

import (
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestWorkflowRuntimeFromMetaDoesNotInferExecution(t *testing.T) {
	state := taskx.WorkflowRuntimeFromMeta(taskx.TaskMeta{
		ID:            "existing-task",
		Status:        "RUNNING",
		Worktree:      ".",
		WorkspaceKind: "primary",
		Delivery:      "unmanaged",
	})
	if state.Planning != workflow.PlanningReady {
		t.Fatalf("got planning %q, want ready", state.Planning)
	}
	if len(state.Attempts) != 0 || state.WriteLease != nil {
		t.Fatalf("durable metadata inferred active execution: %+v", state)
	}
}

func TestProjectWorkflowSummaryChangesOnlyCoreOwnedFields(t *testing.T) {
	meta := taskx.TaskMeta{
		ID: "task-1", Branch: "main", ParentBranch: "main", Worktree: ".",
		WorkspaceKind: "primary", Delivery: "unmanaged", Status: "TODO",
	}
	state := workflow.NewCompatibleRuntime(
		workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary},
		workflow.PlanningReady,
		workflow.DeliveryUnmanaged,
	)
	state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemCompleted}}
	projected, err := taskx.ProjectWorkflowSummary(meta, state)
	if err != nil {
		t.Fatal(err)
	}
	if projected.Status != string(workflow.TaskDone) {
		t.Fatalf("got status %q, want %q", projected.Status, workflow.TaskDone)
	}
	if projected.Branch != meta.Branch || projected.Worktree != meta.Worktree {
		t.Fatalf("projection modified Task-owned fields: %+v", projected)
	}
}

func TestWorkflowOwnerSeparatesDurableAndRuntimeFields(t *testing.T) {
	if owner, ok := taskx.WorkflowOwner("tasks.md.prose"); !ok || owner != taskx.OwnerOpenSpec {
		t.Fatalf("got owner %q, found %t", owner, ok)
	}
	if owner, ok := taskx.WorkflowOwner("runtime.write_lease"); !ok || owner != taskx.OwnerWorkflowCore {
		t.Fatalf("got owner %q, found %t", owner, ok)
	}
}
