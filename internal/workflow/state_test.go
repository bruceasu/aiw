package workflow

import "testing"

func TestValidateWorkItemTransition(t *testing.T) {
	if err := ValidateWorkItemTransition("wi-0001", WorkItemPlanned, WorkItemReady); err != nil {
		t.Fatalf("planned -> ready: %v", err)
	}
	if err := ValidateWorkItemTransition("wi-0001", WorkItemCompleted, WorkItemReady); err == nil {
		t.Fatal("completed -> ready should be rejected")
	}
}

func TestDeriveSummaryUsesCompletedRetry(t *testing.T) {
	state := RuntimeState{
		SchemaVersion: SchemaVersion,
		Task:          TaskReference{ID: "task", Kind: WorkspacePrimary},
		Planning:      PlanningReady,
		Delivery:      DeliveryUnmanaged,
		WorkItems: []WorkItem{{
			ID:    "wi-0001",
			State: WorkItemCompleted,
		}},
		Attempts: []Attempt{
			{ID: "attempt-1", WorkItemID: "wi-0001", State: AttemptFailed},
			{ID: "attempt-2", WorkItemID: "wi-0001", State: AttemptCompleted},
		},
	}
	if err := ValidateRuntimeState(state); err != nil {
		t.Fatalf("validate state: %v", err)
	}
	summary := DeriveSummary(state)
	if summary.Status != TaskDone {
		t.Fatalf("status = %s, want %s", summary.Status, TaskDone)
	}
}

func TestDeriveSummaryRequiresValidationAuthorization(t *testing.T) {
	state := RuntimeState{
		SchemaVersion: SchemaVersion,
		Task:          TaskReference{ID: "task", Kind: WorkspacePrimary},
		Planning:      PlanningReady,
		Delivery:      DeliveryUnmanaged,
		WorkItems:     []WorkItem{{ID: "wi-0001", State: WorkItemCompleted}},
		Gates: []Gate{{
			ID:   "gate-1",
			Kind: GateAuthorization,
			State: GateOpen,
		}},
	}
	summary := DeriveSummary(state)
	if summary.Status != TaskAwaitingAuthorization {
		t.Fatalf("status = %s, want %s", summary.Status, TaskAwaitingAuthorization)
	}
}

func TestValidateRuntimeStateRejectsUnknownDependency(t *testing.T) {
	state := RuntimeState{
		SchemaVersion: SchemaVersion,
		WorkItems: []WorkItem{{
			ID:           "wi-0001",
			Dependencies: []WorkItemID{"wi-9999"},
		}},
	}
	if err := ValidateRuntimeState(state); err == nil {
		t.Fatal("unknown dependency should be rejected")
	}
}

func TestWorkItemIDRoundTrip(t *testing.T) {
	id := NewWorkItemID(12)
	if id != "wi-0012" {
		t.Fatalf("id = %s", id)
	}
	parsed, err := ParseWorkItemID(string(id))
	if err != nil || parsed != id {
		t.Fatalf("parse = %s, %v", parsed, err)
	}
	if _, err := ParseWorkItemID("task-12"); err == nil {
		t.Fatal("invalid identifier should be rejected")
	}
}
