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

func TestSkipFocusedTestWaivesEvidenceAndCompletesBlockedWorkItem(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0004", Title: "Run the authorized focused verification.", State: WorkItemBlocked}}
	state.Gates = []Gate{{ID: "focused-test-missing", WorkItemID: "wi-0004", Kind: GateAuthorization, State: GateOpen}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	updated, workItemID, err := store.SkipFocusedTest("task-1", "verification is optional for this Task")
	if err != nil {
		t.Fatal(err)
	}
	if workItemID != "wi-0004" || updated.WorkItems[0].State != WorkItemCompleted {
		t.Fatalf("skip result = item:%s state:%s", workItemID, updated.WorkItems[0].State)
	}
	if updated.Gates[0].State != GateWaived {
		t.Fatalf("gate state = %s", updated.Gates[0].State)
	}
	if len(updated.Evidence) != 1 || updated.Evidence[0].State != EvidenceWaived {
		t.Fatalf("evidence = %#v", updated.Evidence)
	}
}

func TestSkipFocusedTestIsNoOpAfterVerificationCompleted(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0004", Title: "Run the authorized focused verification.", State: WorkItemCompleted}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	updated, workItemID, err := store.SkipFocusedTest("task-1", "verification remains optional")
	if err != nil {
		t.Fatal(err)
	}
	if workItemID != "" || updated.LastEventSequence != state.LastEventSequence {
		t.Fatalf("completed verification must be a no-op: item=%q event=%d", workItemID, updated.LastEventSequence)
	}
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

func TestFocusedTestAuthorizationIsConsumedOnlyOnce(t *testing.T) {
	store := focusedTestStore(t)
	authorizeActiveFocusedPlan(t, store, focusedPlanDigestA)

	consumed, err := store.ConsumeFocusedTestAuthorization("task-1", focusedPlanDigestA)
	if err != nil {
		t.Fatal(err)
	}
	if got := consumed.FocusedTestAuthorizationState(focusedPlanDigestA); got != FocusedTestAuthorizationConsumed {
		t.Fatalf("authorization after consumption = %q, want consumed", got)
	}
	if _, err := store.ConsumeFocusedTestAuthorization("task-1", focusedPlanDigestA); err == nil {
		t.Fatal("second authorization consumption succeeded")
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

func TestOpenWorkspaceAccessGateRecordsOneFailureWithoutAnAttempt(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "work", State: WorkItemReady, NoProgressCount: 1}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}

	updated, err := store.OpenWorkspaceAccessGate("task-1", "Git rejects this worktree")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Attempts) != 0 {
		t.Fatalf("workspace preflight created attempts: %#v", updated.Attempts)
	}
	if len(updated.Gates) != 1 || updated.Gates[0].Kind != GateWorkspaceAccess || updated.Gates[0].State != GateOpen {
		t.Fatalf("workspace Gate = %#v", updated.Gates)
	}
	if len(updated.Evidence) != 1 || updated.Evidence[0].ID != WorkspaceAccessEvidenceID || updated.Evidence[0].State != EvidenceFailed {
		t.Fatalf("workspace failure evidence = %#v", updated.Evidence)
	}
	if got := DeriveSummary(updated).Execution; got != ExecutionBlocked {
		t.Fatalf("execution = %q, want blocked", got)
	}

	again, err := store.OpenWorkspaceAccessGate("task-1", "Git still rejects this worktree")
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Evidence) != 1 || len(again.Attempts) != 0 {
		t.Fatalf("repeated workspace preflight must preserve evidence and attempts: %#v", again)
	}
	if len(again.Gates) != 1 || again.Gates[0].ID != WorkspaceAccessGateID || again.WorkItems[0].NoProgressCount != 1 {
		t.Fatalf("repeated ownership failure duplicated the Gate or consumed a retry: %+v", again)
	}
	if next := NextRunnerOutcome(again); next.Kind != RunnerGate || next.Request != nil {
		t.Fatalf("ownership failure must stop Agent scheduling: %+v", next)
	}
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
	if _, err := store.RecordAutomation("task-1", "", AutomationCursor{Result: string(RunnerPrepared)}, &PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: "."}); err != nil {
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
	events, err := store.readEvents("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 || events[len(events)-1].Sequence != closed.LastEventSequence {
		t.Fatalf("event log does not match closed state: %+v", events)
	}
	if got := events[len(events)-1].Type; got != "task.force-closed" {
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

func TestSyncChecklistMapsExplicitDependenciesToWorkItemIDs(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	state, err := store.SyncChecklist("task-1", []ChecklistCandidate{
		{Item: "3.2", Title: "compiler"},
		{Item: "5.2", Title: "dependent", DependsOn: []string{"3.2"}},
	}, "dependencies")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.WorkItems[1].Dependencies) != 1 || state.WorkItems[1].Dependencies[0] != state.WorkItems[0].ID {
		t.Fatalf("dependencies=%#v", state.WorkItems[1].Dependencies)
	}
	if _, err := SelectReadyMappedWorkItem(state); err != nil {
		t.Fatal(err)
	}
	state.WorkItems[0].State = WorkItemBlocked
	item, err := SelectReadyMappedWorkItem(state)
	if err == nil || item.ID != "" {
		t.Fatalf("blocked predecessor was scheduled: item=%#v err=%v", item, err)
	}
}
