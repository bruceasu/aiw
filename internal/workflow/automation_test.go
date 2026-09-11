package workflow

import "testing"

func TestProjectionRepairIsDeduplicatedAndResolvable(t *testing.T) {
	store := NewStore(t.TempDir())
	state := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	repair := ProjectionRepair{EventSequence: 1, Target: "tasks.md", Recommended: "workflow repair task-1"}
	if _, err := store.EnqueueProjectionRepair("task-1", repair); err != nil {
		t.Fatal(err)
	}
	current, err := store.EnqueueProjectionRepair("task-1", repair)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Automation.ProjectionRepairs) != 1 {
		t.Fatalf("repairs = %d", len(current.Automation.ProjectionRepairs))
	}
	current, err = store.ResolveProjectionRepair("task-1", 1, "tasks.md")
	if err != nil {
		t.Fatal(err)
	}
	if current.Automation.ProjectionRepairs[0].ResolvedAt == "" {
		t.Fatal("repair was not resolved")
	}
}
