package taskx_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestProjectWorkflowSummaryDoesNotMarkOutstandingVerificationDone(t *testing.T) {
	base := workflow.NewCompatibleRuntime(
		workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary},
		workflow.PlanningReady,
		workflow.DeliveryMerged,
	)
	base.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemCompleted}}

	for _, tc := range []struct {
		name string
		state workflow.RuntimeState
		want  workflow.TaskDisplayState
	}{
		{
			name: "done after completed work and no verification requirement",
			state: base,
			want:  workflow.TaskDone,
		},
		{
			name: "awaiting authorization",
			state: func() workflow.RuntimeState {
				state := base
				state.FocusedTestPlanDigest = strings.Repeat("a", 64)
				return state
			}(),
			want: workflow.TaskAwaitingAuthorization,
		},
		{
			name: "awaiting verification",
			state: func() workflow.RuntimeState {
				state := base
				state.Evidence = []workflow.Evidence{{ID: "verification", Kind: workflow.EvidenceCommand, State: workflow.EvidencePending}}
				return state
			}(),
			want: workflow.TaskAwaitingVerification,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.state.Summary = workflow.DeriveSummary(tc.state)
			projected, err := taskx.ProjectWorkflowSummary(taskx.TaskMeta{ID: "task-1"}, tc.state)
			if err != nil {
				t.Fatal(err)
			}
			if projected.Status != string(tc.want) {
				t.Fatalf("status = %q, want %q", projected.Status, tc.want)
			}
			if projected.Delivery != string(workflow.DeliveryMerged) {
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
	state := workflow.NewCompatibleRuntime(
		workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary},
		workflow.PlanningReady,
		workflow.DeliveryMerged,
	)
	state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemCompleted}}
	state.Summary = workflow.DeriveSummary(state)

	if _, err := taskx.WriteWorkflowSummary(path, state); err != nil {
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
	if _, err := taskx.WriteWorkflowSummary(path, state); err != nil {
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
	if owner, ok := taskx.WorkflowOwner("tasks.md.prose"); !ok || owner != taskx.OwnerOpenSpec {
		t.Fatalf("got owner %q, found %t", owner, ok)
	}
	if owner, ok := taskx.WorkflowOwner("runtime.write_lease"); !ok || owner != taskx.OwnerWorkflowCore {
		t.Fatalf("got owner %q, found %t", owner, ok)
	}
}
