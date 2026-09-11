package workflow

import "testing"

func TestApplyLegacyStatusRejectsFabricatedDone(t *testing.T) {
	state := NewCompatibleRuntime(
		TaskReference{ID: "task-1", Kind: WorkspacePrimary},
		PlanningReady,
		DeliveryUnmanaged,
	)
	if err := ApplyLegacyStatus(&state, "DONE"); err == nil {
		t.Fatal("expected incomplete Task to reject DONE")
	}
}

func TestApplyLegacyStatusMapsPlanningAliases(t *testing.T) {
	state := NewCompatibleRuntime(
		TaskReference{ID: "task-1", Kind: WorkspacePrimary},
		PlanningDraft,
		DeliveryUnmanaged,
	)
	if err := ApplyLegacyStatus(&state, "ready"); err != nil {
		t.Fatal(err)
	}
	if state.Summary.Status != TaskReady {
		t.Fatalf("got %s, want %s", state.Summary.Status, TaskReady)
	}
}

func TestLoadMigratesLegacyWorkItemRetryPolicyToDefault(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.Version = SchemaVersion - 1
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "legacy work", State: WorkItemReady}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}

	migrated, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if migrated.Version != SchemaVersion || migrated.WorkItems[0].RetryPolicy.MaxAttempts != DefaultRetryLimit || migrated.WorkItems[0].NoProgressCount != 0 {
		t.Fatalf("legacy retry state was not migrated: %+v", migrated)
	}
}
