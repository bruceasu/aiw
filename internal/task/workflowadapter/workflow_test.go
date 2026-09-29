package workflowadapter_test

import (
	"bytes"
	"os"
	"os/exec"
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

func TestAcceptedWorkItemProjectsToNativePrimaryFD(t *testing.T) {
	t.Chdir(t.TempDir())
	if output, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil { t.Fatalf("git init: %v: %s", err, output) }
	id := "native-primary"
	meta := task.TaskMeta{ID: id, Status: "TODO", Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged"}
	if err := task.WriteTaskMeta(task.TaskMetaPath(id), meta); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(task.FeatureDesignDir, 0o755); err != nil { t.Fatal(err) }
	fd := "## Design Readiness\n\nFD_APPLIED\n\n## Work Items\n\n- [ ] 1.1 Implement feature\n"
	if err := os.WriteFile(task.FeatureDesignPath(id), []byte(fd), 0o644); err != nil { t.Fatal(err) }
	store := workflowcore.NewStore("")
	if _, err := store.Create(taskworkflow.WorkflowRuntimeFromMeta(meta)); err != nil { t.Fatal(err) }
	state, err := taskworkflow.SyncWorkflowChecklist(id, store)
	if err != nil || len(state.WorkItems) != 1 { t.Fatalf("plan: %#v %v", state.WorkItems, err) }
	state, err = store.CompleteWorkItem(workflowcore.TaskID(id), state.WorkItems[0].ID)
	if err != nil { t.Fatal(err) }
	if err := taskworkflow.ProjectAcceptedWorkItem(id, store, state, state.WorkItems[0].ID); err != nil { t.Fatal(err) }
	content, err := os.ReadFile(task.FeatureDesignPath(id))
	if err != nil || !strings.Contains(string(content), "- [x] 1.1 Implement feature") { t.Fatalf("FD projection: %s %v", content, err) }
}

func TestBlockedFDDoesNotMapWorkItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "features", "blocked.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte("## Design Readiness\n\nBLOCKED\n\n## Work Items\n- [ ] 1.1 Work\n"), 0o644); err != nil { t.Fatal(err) }
	if _, err := taskworkflow.SyncWorkflowChecklistAtPath("blocked", nil, path); err == nil || !strings.Contains(err.Error(), "BLOCKED") { t.Fatalf("expected design readiness block, got %v", err) }
}

func TestBindPrimaryAfterMergePreservesUnfinishedTaskLineage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.toml")
	meta := task.TaskMeta{ID: "unfinished", Status: "READY", Branch: "feature/unfinished", ParentBranch: "main", Worktree: "", WorkspaceKind: "unassigned", Delivery: "merged", Session: "unfinished-session"}
	if err := task.WriteTaskMeta(path, meta); err != nil { t.Fatal(err) }
	bound, err := taskworkflow.BindPrimaryAfterMerge(path, meta.ID)
	if err != nil { t.Fatal(err) }
	if bound.WorkspaceKind != "primary" || bound.Worktree != "." || bound.Branch != "main" || bound.Session != meta.Session || bound.Delivery != "merged" || bound.Status != "READY" {
		t.Fatalf("unfinished Task was not preserved in primary: %+v", bound)
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
