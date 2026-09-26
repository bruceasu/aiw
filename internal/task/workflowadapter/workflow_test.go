package workflowadapter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	workflowcore "aiw/internal/workflow"
)

func TestWorkflowRuntimeFromMetaDoesNotInferExecution(t *testing.T) {
	state := taskworkflow.WorkflowRuntimeFromMeta(task.TaskMeta{
		ID:            "existing-task",
		Status:        "RUNNING",
		Worktree:      ".",
		WorkspaceKind: "primary",
		Delivery:      "unmanaged",
	})
	if state.Planning != workflowcore.PlanningReady {
		t.Fatalf("got planning %q, want ready", state.Planning)
	}
	if len(state.Attempts) != 0 || state.WriteLease != nil {
		t.Fatalf("durable metadata inferred active execution: %+v", state)
	}
}

func TestProjectWorkflowSummaryChangesOnlyCoreOwnedFields(t *testing.T) {
	meta := task.TaskMeta{
		ID: "task-1", Branch: "main", ParentBranch: "main", Worktree: ".",
		WorkspaceKind: "primary", Delivery: "unmanaged", Status: "TODO",
	}
	state := workflowcore.NewCompatibleRuntime(
		workflowcore.TaskReference{ID: "task-1", Workspace: ".", Kind: workflowcore.WorkspacePrimary},
		workflowcore.PlanningReady,
		workflowcore.DeliveryUnmanaged,
	)
	state.WorkItems = []workflowcore.WorkItem{{ID: "wi-0001", State: workflowcore.WorkItemCompleted}}
	projected, err := taskworkflow.ProjectWorkflowSummary(meta, state)
	if err != nil {
		t.Fatal(err)
	}
	if projected.Status != string(workflowcore.TaskDone) {
		t.Fatalf("got status %q, want %q", projected.Status, workflowcore.TaskDone)
	}
	if projected.Branch != meta.Branch || projected.Worktree != meta.Worktree {
		t.Fatalf("projection modified Task-owned fields: %+v", projected)
	}
}

func TestProjectWorkflowSummaryDoesNotMarkOutstandingVerificationDone(t *testing.T) {
	base := workflowcore.NewCompatibleRuntime(
		workflowcore.TaskReference{ID: "task-1", Workspace: ".", Kind: workflowcore.WorkspacePrimary},
		workflowcore.PlanningReady,
		workflowcore.DeliveryMerged,
	)
	base.WorkItems = []workflowcore.WorkItem{{ID: "wi-0001", State: workflowcore.WorkItemCompleted}}

	for _, tc := range []struct {
		name string
		state workflowcore.RuntimeState
		want  workflowcore.TaskDisplayState
	}{
		{
			name: "done after completed work and no verification requirement",
			state: base,
			want:  workflowcore.TaskDone,
		},
		{
			name: "awaiting authorization",
			state: func() workflowcore.RuntimeState {
				state := base
				state.FocusedTestPlanDigest = strings.Repeat("a", 64)
				return state
			}(),
			want: workflowcore.TaskAwaitingAuthorization,
		},
		{
			name: "awaiting verification",
			state: func() workflowcore.RuntimeState {
				state := base
				state.Evidence = []workflowcore.Evidence{{ID: "verification", Kind: workflowcore.EvidenceCommand, State: workflowcore.EvidencePending}}
				return state
			}(),
			want: workflowcore.TaskAwaitingVerification,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.state.Summary = workflowcore.DeriveSummary(tc.state)
			projected, err := taskworkflow.ProjectWorkflowSummary(task.TaskMeta{ID: "task-1"}, tc.state)
			if err != nil {
				t.Fatal(err)
			}
			if projected.Status != string(tc.want) {
				t.Fatalf("status = %q, want %q", projected.Status, tc.want)
			}
			if projected.Delivery != string(workflowcore.DeliveryMerged) {
				t.Fatalf("delivery = %q, want merged", projected.Delivery)
			}
		})
	}
}

func TestWriteWorkflowSummaryPreservesUnknownFieldsAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.toml")
	before := []byte("id = \"task-1\"\nstatus = \"TODO\"\ndelivery = \"pending\"\noperator_tag = \"keep-me\"\n")
	if err := os.WriteFile(path, before, 0o644); err != nil {
		t.Fatal(err)
	}
	state := workflowcore.NewCompatibleRuntime(
		workflowcore.TaskReference{ID: "task-1", Workspace: ".", Kind: workflowcore.WorkspacePrimary},
		workflowcore.PlanningReady,
		workflowcore.DeliveryMerged,
	)
	state.WorkItems = []workflowcore.WorkItem{{ID: "wi-0001", State: workflowcore.WorkItemCompleted}}
	state.Summary = workflowcore.DeriveSummary(state)

	if _, err := taskworkflow.WriteWorkflowSummary(path, state); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"status = \"DONE\"", "delivery = \"merged\"", "operator_tag = \"keep-me\""} {
		if !strings.Contains(string(first), field) {
			t.Fatalf("projected metadata is missing %q: %s", field, first)
		}
	}
	if _, err := taskworkflow.WriteWorkflowSummary(path, state); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("repeated projection rewrote metadata:\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestWorkflowOwnerSeparatesDurableAndRuntimeFields(t *testing.T) {
	if owner, ok := taskworkflow.WorkflowOwner("tasks.md.prose"); !ok || owner != taskworkflow.OwnerOpenSpec {
		t.Fatalf("got owner %q, found %t", owner, ok)
	}
	if owner, ok := taskworkflow.WorkflowOwner("runtime.write_lease"); !ok || owner != taskworkflow.OwnerWorkflowCore {
		t.Fatalf("got owner %q, found %t", owner, ok)
	}
}
