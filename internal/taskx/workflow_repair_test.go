package taskx_test

import (
	"os"
	"path/filepath"
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestRepairWorkflowChecklistPreservesAuthoredDependencies(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	meta := taskx.TaskMeta{ID: "recovery", Status: "TODO", Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged"}
	if err := os.MkdirAll(taskx.TaskDir(meta.ID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath(meta.ID), meta); err != nil {
		t.Fatal(err)
	}
	checklist := "- [ ] 1.1 Implement\n- [ ] 3.1 Accept scope <!-- aiw:depends-on=1.1 -->\n"
	if err := os.WriteFile(filepath.Join(taskx.TaskDir(meta.ID), "tasks.md"), []byte(checklist), 0o644); err != nil {
		t.Fatal(err)
	}
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(taskx.WorkflowRuntimeFromMeta(meta)); err != nil {
		t.Fatal(err)
	}
	before, err := store.ReconcileChecklist("recovery", []workflow.ChecklistCandidate{
		{Item: "3.1", Title: "Accept scope", DependsOn: []string{"1.1"}},
		{Item: "1.1", Title: "Implement"},
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := taskx.RepairWorkflowChecklist(meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.WorkItems[0].ID != before.WorkItems[0].ID || len(after.WorkItems[0].Dependencies) != 1 || after.WorkItems[0].Dependencies[0] != before.WorkItems[1].ID {
		t.Fatalf("repair changed identity or lost dependencies: %+v", after.WorkItems)
	}
	selected, err := workflow.SelectReadyMappedWorkItem(after)
	if err != nil || selected.ID != before.WorkItems[1].ID {
		t.Fatalf("repair made acceptance executable: %+v, %v", selected, err)
	}
	repeated, err := taskx.RepairWorkflowChecklist(meta.ID)
	if err != nil || repeated.LastEventSequence != after.LastEventSequence {
		t.Fatalf("one-time repair was repeated: %+v, %v", repeated, err)
	}
}
