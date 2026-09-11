package workflow

import (
	"strings"
	"testing"
)

const (
	focusedPlanDigestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	focusedPlanDigestB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func focusedTestStore(t *testing.T) *Store {
	t.Helper()
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "work", State: WorkItemCompleted}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	return store
}

func authorizeActiveFocusedPlan(t *testing.T, store *Store, digest string) RuntimeState {
	t.Helper()
	if _, err := store.ActivateFocusedTestPlan("task-1", digest); err != nil {
		t.Fatal(err)
	}
	state, err := store.AuthorizeFocusedTest("task-1", digest, "reviewer@example.test")
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestFocusedTestAuthorizationLifecycle(t *testing.T) {
	store := focusedTestStore(t)

	awaiting, err := store.ActivateFocusedTestPlan("task-1", focusedPlanDigestA)
	if err != nil {
		t.Fatal(err)
	}
	if awaiting.FocusedTestAuthorizationState(focusedPlanDigestA) != FocusedTestAuthorizationDisabled {
		t.Fatalf("authorization = %q, want disabled", awaiting.FocusedTestAuthorizationState(focusedPlanDigestA))
	}
	if awaiting.Summary.Validation != ValidationAwaitingAuthorization || awaiting.Summary.Status != TaskAwaitingAuthorization {
		t.Fatalf("missing authorization summary = %+v", awaiting.Summary)
	}
	if len(awaiting.Gates) != 1 || awaiting.Gates[0].ID != FocusedTestAuthorizationGateID || awaiting.Gates[0].State != GateOpen {
		t.Fatalf("missing open focused-test gate: %+v", awaiting.Gates)
	}

	authorized, err := store.AuthorizeFocusedTest("task-1", focusedPlanDigestA, "reviewer@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if authorized.FocusedTestAuthorizationState(focusedPlanDigestA) != FocusedTestAuthorizationAuthorized {
		t.Fatalf("authorization = %q, want authorized", authorized.FocusedTestAuthorizationState(focusedPlanDigestA))
	}
	if authorized.FocusedTestAuthorization == nil || authorized.FocusedTestAuthorization.Approver != "reviewer@example.test" || authorized.FocusedTestAuthorization.ApprovedAt == "" {
		t.Fatalf("authorization record = %+v", authorized.FocusedTestAuthorization)
	}
	if authorized.Gates[0].State != GateResolved {
		t.Fatalf("gate state = %q, want resolved", authorized.Gates[0].State)
	}
	if authorized.Summary.Validation != ValidationPending {
		t.Fatalf("authorized validation = %q, want pending", authorized.Summary.Validation)
	}
}

func TestFocusedTestPlanChangeInvalidatesAuthorization(t *testing.T) {
	store := focusedTestStore(t)
	authorizeActiveFocusedPlan(t, store, focusedPlanDigestA)

	stale, err := store.ActivateFocusedTestPlan("task-1", focusedPlanDigestB)
	if err != nil {
		t.Fatal(err)
	}
	if stale.FocusedTestAuthorizationState(focusedPlanDigestB) != FocusedTestAuthorizationStale {
		t.Fatalf("authorization = %q, want stale", stale.FocusedTestAuthorizationState(focusedPlanDigestB))
	}
	if stale.Summary.Validation != ValidationAwaitingAuthorization || stale.Summary.Status != TaskAwaitingAuthorization {
		t.Fatalf("stale authorization summary = %+v", stale.Summary)
	}
	if stale.Gates[0].State != GateOpen || !strings.Contains(stale.Gates[0].Reason, focusedPlanDigestA) || !strings.Contains(stale.Gates[0].Reason, focusedPlanDigestB) {
		t.Fatalf("stale authorization gate = %+v", stale.Gates[0])
	}
}

func TestFocusedTestEvidenceDerivesPendingFailedAndPassedWithoutRepair(t *testing.T) {
	cases := []struct {
		name       string
		evidence   EvidenceState
		validation ValidationState
	}{
		{name: "pending", evidence: EvidencePending, validation: ValidationPending},
		{name: "failed", evidence: EvidenceFailed, validation: ValidationFailed},
		{name: "passed", evidence: EvidencePassed, validation: ValidationPassed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := focusedTestStore(t)
			before := authorizeActiveFocusedPlan(t, store, focusedPlanDigestA)
			after, err := store.RecordEvidence("task-1", Evidence{
				ID:         EvidenceID("focused-" + tc.name),
				WorkItemID: "wi-0001",
				Kind:       EvidenceCommand,
				State:      tc.evidence,
				Reference:  "artifacts/focused-" + tc.name + ".json",
			})
			if err != nil {
				t.Fatal(err)
			}
			if after.Summary.Validation != tc.validation {
				t.Fatalf("validation = %q, want %q", after.Summary.Validation, tc.validation)
			}
			if tc.evidence == EvidenceFailed {
				if len(after.Attempts) != len(before.Attempts) || len(after.Gates) != len(before.Gates) || after.Automation.PreparedRequest != before.Automation.PreparedRequest || len(after.Automation.ProjectionRepairs) != len(before.Automation.ProjectionRepairs) || after.Automation.Cursor != before.Automation.Cursor {
					t.Fatalf("failed evidence started repair or changed workflow ownership: before=%+v after=%+v", before, after)
				}
				if after.WorkItems[0].State != WorkItemCompleted {
					t.Fatalf("failed evidence changed work item state to %q", after.WorkItems[0].State)
				}
			}
		})
	}
}

func TestCommandEvidenceRequiresResolvedAuthorization(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "work", State: WorkItemReady}}
	state.Gates = []Gate{{ID: "gate-auth", WorkItemID: "wi-0001", Kind: GateAuthorization, State: GateOpen}}
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	if _, err := store.RecordEvidence("task-1", Evidence{ID: "e-1", WorkItemID: "wi-0001", Kind: EvidenceCommand, State: EvidencePassed}); err == nil { t.Fatal("expected authorization rejection") }
	if _, err := store.ResolveGate("task-1", "gate-auth", GateResolved); err != nil { t.Fatal(err) }
	if _, err := store.RecordEvidence("task-1", Evidence{ID: "e-1", WorkItemID: "wi-0001", Kind: EvidenceCommand, State: EvidencePassed}); err != nil { t.Fatal(err) }
}

func TestSetDeliveryRequiresDeliveryGate(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.Gates = []Gate{{ID: "gate-delivery", Kind: GateDelivery, State: GateOpen}}
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	if _, err := store.SetDelivery("task-1", DeliveryMerged); err == nil { t.Fatal("expected delivery authorization rejection") }
	if _, err := store.ResolveGate("task-1", "gate-delivery", GateResolved); err != nil { t.Fatal(err) }
	updated, err := store.SetDelivery("task-1", DeliveryMerged)
	if err != nil { t.Fatal(err) }
	if updated.Delivery != DeliveryMerged { t.Fatalf("got delivery %q", updated.Delivery) }
}

func TestDiagnoseMissingRuntimeIsReadOnly(t *testing.T) {
	store := NewStore(t.TempDir())
	diagnostics, err := store.Diagnose("task-1")
	if err != nil { t.Fatal(err) }
	if len(diagnostics) != 1 || diagnostics[0].Code != "runtime-missing" { t.Fatalf("unexpected diagnostics: %#v", diagnostics) }
}

func TestDiagnoseReportsPreparedRequestAndOpenGate(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.Gates = []Gate{{ID: "gate-auth", WorkItemID: "wi-0001", Kind: GateAuthorization, State: GateOpen}}
	state.Automation.PreparedRequest = &PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1"}
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	diagnostics, err := store.Diagnose("task-1")
	if err != nil { t.Fatal(err) }
	if len(diagnostics) != 2 { t.Fatalf("unexpected diagnostics: %#v", diagnostics) }
	if diagnostics[0].Code != "agent-request-prepared" { t.Fatalf("first diagnostic: %#v", diagnostics[0]) }
	if diagnostics[1].Code != "gate-open" { t.Fatalf("second diagnostic: %#v", diagnostics[1]) }
	if diagnostics[1].Repair != "review the gate reason, then run: aiw task workflow gate task-1 gate-auth resolved|waived" { t.Fatalf("gate repair: %q", diagnostics[1].Repair) }
}

func TestForceCloseCancelsActiveAttemptAndClearsExecutionOwnership(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "work", State: WorkItemReady}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: ".", State: AttemptCreated}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordAutomation("task-1", "", AutomationCursor{}, &PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1"}); err != nil {
		t.Fatal(err)
	}

	closed, err := store.ForceClose("task-1", "superseded by a replacement", DeliveryDiscarded)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Attempts[0].State != AttemptCancelled || closed.WorkItems[0].State != WorkItemCancelled {
		t.Fatalf("force-close left active work: %+v", closed)
	}
	if closed.WriteLease != nil || closed.Automation.PreparedRequest != nil {
		t.Fatalf("force-close retained execution ownership: %+v", closed.Automation)
	}
	if closed.Cancellation == nil || closed.Cancellation.Reason != "superseded by a replacement" || closed.Cancellation.Delivery != DeliveryDiscarded || closed.Delivery != DeliveryPending {
		t.Fatalf("force-close terminal record is incomplete: %+v", closed)
	}
	if got := closed.Events[len(closed.Events)-1].Type; got != "task.force-closed" {
		t.Fatalf("last event = %q, want task.force-closed", got)
	}
}

func TestReconcileChecklistKeepsWorkItemIDsAcrossRenameAndReorder(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileChecklist("task-1", []ChecklistCandidate{
		{Item: "1.1", Title: "first"},
		{Item: "1.2", Title: "second"},
	}); err != nil {
		t.Fatal(err)
	}
	state, err := store.ReconcileChecklist("task-1", []ChecklistCandidate{
		{Item: "1.2", Title: "renamed second"},
		{Item: "1.1", Title: "renamed first"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.WorkItems[0].ID != "wi-0001" || state.WorkItems[0].Title != "renamed first" {
		t.Fatalf("first mapping changed unexpectedly: %#v", state.WorkItems[0])
	}
	if state.WorkItems[1].ID != "wi-0002" || state.WorkItems[1].Title != "renamed second" {
		t.Fatalf("second mapping changed unexpectedly: %#v", state.WorkItems[1])
	}
}
