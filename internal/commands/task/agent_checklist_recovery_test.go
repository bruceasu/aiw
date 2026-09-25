package task

import (
	"os"
	"path/filepath"
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestStartManagedAttemptRefreshesAuthoredDependencies(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	meta := taskx.TaskMeta{ID: "recovery", Status: "TODO", Worktree: ".", WorkspaceKind: "primary", Session: "recovery", Delivery: "unmanaged"}
	if err := os.MkdirAll(taskx.TaskDir(meta.ID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath(meta.ID), meta); err != nil {
		t.Fatal(err)
	}
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(taskx.WorkflowRuntimeFromMeta(meta)); err != nil {
		t.Fatal(err)
	}
	// Acceptance was created first, before its implementation dependency was authored.
	before, err := store.ReconcileChecklist("recovery", []workflow.ChecklistCandidate{
		{Item: "3.1", Title: "Accept scope"},
		{Item: "1.1", Title: "Implement"},
	})
	if err != nil {
		t.Fatal(err)
	}
	checklist := "- [ ] 1.1 Implement\n- [ ] 3.1 Accept scope <!-- aiw:depends-on=1.1 -->\n"
	if err := os.WriteFile(filepath.Join(taskx.TaskDir(meta.ID), "tasks.md"), []byte(checklist), 0o644); err != nil {
		t.Fatal(err)
	}
	attemptID, err := startManagedAttempt(meta.ID, meta)
	if err != nil {
		t.Fatal(err)
	}
	after, err := store.Load("recovery")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Attempts) != 1 || after.Attempts[0].WorkItemID != before.WorkItems[1].ID {
		t.Fatalf("selected acceptance before implementation: %+v", after.Attempts)
	}
	if after.WriteLease == nil || after.WriteLease.AttemptID != attemptID {
		t.Fatalf("wrong lease owner: %+v", after.WriteLease)
	}
	if len(after.WorkItems[0].Dependencies) != 1 || after.WorkItems[0].Dependencies[0] != before.WorkItems[1].ID {
		t.Fatalf("missing acceptance dependency: %+v", after.WorkItems[0])
	}
}
