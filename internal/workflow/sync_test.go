package workflow

import "testing"

func TestSyncChecklistRetainsMappingAndGatesMissingItem(t *testing.T) {
	store := NewStore(t.TempDir())
	initial := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(initial); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.1", Title: "first"}}, "one"); err != nil {
		t.Fatal(err)
	}
	state, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.2", Title: "new"}}, "two")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.WorkItems) != 2 {
		t.Fatalf("work items = %d", len(state.WorkItems))
	}
	if len(state.Gates) != 1 || state.Gates[0].WorkItemID != "wi-0001" {
		t.Fatalf("missing mapping gate = %#v", state.Gates)
	}
	unchanged, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.2", Title: "new"}}, "two")
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.LastEventSequence != state.LastEventSequence {
		t.Fatal("unchanged plan appended an event")
	}
}

func TestSyncChecklistCompletesCheckedItem(t *testing.T) {
	store := NewStore(t.TempDir())
	initial := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(initial); err != nil {
		t.Fatal(err)
	}

	state, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.1", Title: "done", Completed: true}}, "one")
	if err != nil {
		t.Fatal(err)
	}
	if got := state.WorkItems[0].State; got != WorkItemCompleted {
		t.Fatalf("work item state = %q, want %q", got, WorkItemCompleted)
	}
}

func TestSyncChecklistReportsCompletedItemConflict(t *testing.T) {
	store := NewStore(t.TempDir())
	initial := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(initial); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.1", Title: "blocked"}}, "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateWithEvent("task-1", Event{Type: "gate.opened"}, func(state *RuntimeState) error {
		state.Gates = append(state.Gates, Gate{ID: "gate-1", WorkItemID: "wi-0001", Kind: GateDecision, State: GateOpen})
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.1", Title: "blocked", Completed: true}}, "two"); err == nil {
		t.Fatal("expected completion conflict")
	}
	state, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := state.WorkItems[0].State; got != WorkItemReady {
		t.Fatalf("work item state = %q, want %q", got, WorkItemReady)
	}
}

func TestSyncChecklistDoesNotReopenCompletedItem(t *testing.T) {
	store := NewStore(t.TempDir())
	initial := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(initial); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.1", Title: "done", Completed: true}}, "one"); err != nil {
		t.Fatal(err)
	}

	state, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.1", Title: "done"}}, "two")
	if err != nil {
		t.Fatal(err)
	}
	if got := state.WorkItems[0].State; got != WorkItemCompleted {
		t.Fatalf("work item state = %q, want %q", got, WorkItemCompleted)
	}
}

func TestSyncChecklistClosesPreparedAttemptForCompletedItem(t *testing.T) {
	store := NewStore(t.TempDir())
	initial := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(initial); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.1", Title: "done"}}, "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: "."}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordAutomation("task-1", "one", AutomationCursor{Result: "agent-request-prepared"}, &PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: "."}); err != nil {
		t.Fatal(err)
	}

	state, err := store.SyncChecklist("task-1", []ChecklistCandidate{{Item: "1.1", Title: "done", Completed: true}}, "two")
	if err != nil {
		t.Fatal(err)
	}
	if state.WorkItems[0].State != WorkItemCompleted || state.Attempts[0].State != AttemptCompleted || state.WriteLease != nil || state.Automation.PreparedRequest != nil {
		t.Fatalf("completed checklist left active execution state: %+v", state)
	}
}
