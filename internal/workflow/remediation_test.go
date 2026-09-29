package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemediationProblemIdentityIsStableAndEvidenceBound(t *testing.T) {
	base := FailureReport{
		SchemaVersion:     FailureReportSchemaVersion,
		TaskID:            "task-1",
		EventSequence:     7,
		WorkItemID:        "wi-0001",
		AttemptID:         "attempt-1",
		Category:          "runner",
		Detail:            "agent exited before producing a result",
		EvidenceReference: "sessions/session-1/output.json",
		Retryable:         true,
		Owner:             "supervisor",
		RecommendedAction: "review the recorded evidence",
		RecordedAt:        "2026-09-27T00:00:00Z",
	}

	first, firstID, err := BuildRemediationProblem(base)
	if err != nil {
		t.Fatal(err)
	}
	second, secondID, err := BuildRemediationProblem(base)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || firstID != secondID {
		t.Fatalf("same failure produced different identity: %#v/%s vs %#v/%s", first, firstID, second, secondID)
	}

	changed := base
	changed.Detail = "agent exited with a different failure"
	_, changedID, err := BuildRemediationProblem(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedID == firstID {
		t.Fatal("changed failure detail reused the same problem identity")
	}
}

func TestGateRemediationRequiresExplicitHumanDecision(t *testing.T) {
	state := NewCompatibleRuntime(TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, PlanningReady, DeliveryUnmanaged)
	state.LastEventSequence = 7
	gate := Gate{ID: "review-wi-0001", WorkItemID: "wi-0001", Kind: GateDecision, State: GateOpen, Reason: "human decision required"}
	report, err := BuildRemediationReportForGate(state, gate)
	if err != nil { t.Fatal(err) }
	if report.Problem.GateID != gate.ID || report.Status != RemediationAwaitingHuman || report.ProblemID == "" {
		t.Fatalf("Gate report lost identity: %+v", report)
	}
	for _, option := range report.Options {
		if option.Action == RemediationActionResolveGate || option.Action == RemediationActionWaiveGate {
			if !option.RequiresApproval { t.Fatalf("Gate decision lacks explicit approval: %+v", option) }
		}
	}
}

func TestUnknownSessionRemediationBindsDispatchedTurn(t *testing.T) {
	state := NewCompatibleRuntime(TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, PlanningReady, DeliveryUnmanaged)
	state.LastEventSequence = 7
	request := PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", ExpectedSessionTurn: 3, DispatchedAt: "2026-09-27T00:00:00Z"}
	report, err := BuildRemediationReportForUnknownSession(state, request, "result incomplete")
	if err != nil { t.Fatal(err) }
	if report.Problem.SessionID != request.SessionID || report.Problem.SessionTurn != 3 || report.Problem.AttemptID != request.AttemptID || len(report.Options) != 2 { t.Fatalf("unknown result lost request binding: %+v", report) }
	request.DispatchedAt = ""
	if _, err := BuildRemediationReportForUnknownSession(state, request, "result incomplete"); err == nil { t.Fatal("undispatched request must not offer result re-observation") }
}

func TestRepeatedAwaitRemediationPreservesHumanResponse(t *testing.T) {
	store, _, report := newAwaitingRemediationFixture(t)
	response := []byte(`{"option_id":"human-review","operator":"operator-1"}`)
	writeRemediationResponse(t, store, report, response)
	restarted := NewRemediationReport(report.Problem, nil, RemediationAwaitingHuman, report.Round, report.Options)
	if _, err := store.AwaitRemediation("task-1", restarted); err != nil { t.Fatal(err) }
	path := filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(report.HumanResponsePath))
	content, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	if string(content) != string(response) { t.Fatalf("repeated await replaced human response: %q", content) }
	_, pending, err := store.ReadPendingRemediation("task-1")
	if err != nil || pending.ReportDigest != report.ReportDigest { t.Fatalf("report identity changed: %+v, %v", pending, err) }
}

func TestGateDecisionFinishesLegacyPartialTransition(t *testing.T) {
	store, report, response := gateRemediationFixture(t, false)
	if _, err := store.ResolveGate("task-1", report.Problem.GateID, GateResolved); err != nil { t.Fatal(err) }
	updated, err := store.ApplyGateRemediationResponse("task-1", response)
	if err != nil { t.Fatal(err) }
	if updated.Gates[0].State != GateResolved || updated.WorkItems[0].State != WorkItemReady || updated.Automation.Cursor.Result != "remediation-response-consumed" {
		t.Fatalf("legacy partial Gate decision not reconciled: %+v", updated)
	}
}

func TestGateDecisionKeepsItemBlockedForAnotherOpenGate(t *testing.T) {
	store, _, response := gateRemediationFixture(t, true)
	updated, err := store.ApplyGateRemediationResponse("task-1", response)
	if err != nil { t.Fatal(err) }
	if updated.Gates[0].State != GateResolved || updated.Gates[1].State != GateOpen || updated.WorkItems[0].State != WorkItemBlocked || updated.Automation.Cursor.Result != "remediation-response-consumed" {
		t.Fatalf("Gate decision bypassed another open Gate: %+v", updated)
	}
}

func TestInterruptedGateDecisionRecoversCoupledStateOnce(t *testing.T) {
	store, report, _ := gateRemediationFixture(t, false)
	before, err := store.Load("task-1")
	if err != nil { t.Fatal(err) }
	pending := Event{SchemaVersion: before.SchemaVersion, Sequence: before.LastEventSequence + 1, Type: "remediation.gate-decision", WorkItemID: "wi-0001", Detail: string(report.Problem.GateID) + ":resolve-gate", At: "2026-09-27T00:00:00Z"}
	if _, err := store.update("task-1", func(state *RuntimeState) error {
		state.Gates[0].State = GateResolved
		state.WorkItems[0].State = WorkItemReady
		state.Automation.Cursor = AutomationCursor{Result: "remediation-response-consumed", Detail: remediationCursorKey(report), RecordedAt: pending.At}
		state.Automation.Supervisor.Result = "remediation-response-consumed"
		state.PendingEvent = &pending
		return nil
	}); err != nil { t.Fatal(err) }
	recovered, err := store.RecoverPendingEvent("task-1")
	if err != nil { t.Fatal(err) }
	if recovered.PendingEvent != nil || recovered.LastEventSequence != pending.Sequence || recovered.Gates[0].State != GateResolved || recovered.WorkItems[0].State != WorkItemReady || recovered.Automation.Cursor.Result != "remediation-response-consumed" {
		t.Fatalf("interrupted decision lost coupled state: %+v", recovered)
	}
	again, err := store.RecoverPendingEvent("task-1")
	if err != nil || again.LastEventSequence != recovered.LastEventSequence { t.Fatalf("recovery duplicated event: %+v, %v", again, err) }
}

func gateRemediationFixture(t *testing.T, anotherGate bool) (*Store, RemediationReport, RemediationResponse) {
	t.Helper()
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "blocked", State: WorkItemBlocked}, {ID: "wi-0002", Title: "other", State: WorkItemReady}}
	state.Gates = []Gate{{ID: "review-wi-0001", WorkItemID: "wi-0001", Kind: GateDecision, State: GateOpen, Reason: "review required"}}
	if anotherGate { state.Gates = append(state.Gates, Gate{ID: "second-wi-0001", WorkItemID: "wi-0001", Kind: GateDecision, State: GateOpen, Reason: "second review"}) }
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	if _, err := store.SetRetryPolicy("task-1", "wi-0002", RetryPolicy{MaxAttempts: 3}); err != nil { t.Fatal(err) }
	state, err := store.Load("task-1")
	if err != nil { t.Fatal(err) }
	report, err := BuildRemediationReportForGate(state, state.Gates[0])
	if err != nil { t.Fatal(err) }
	if _, err := store.AwaitRemediation("task-1", report); err != nil { t.Fatal(err) }
	_, report, err = store.ReadPendingRemediation("task-1")
	if err != nil { t.Fatal(err) }
	response := RemediationResponse{SchemaVersion: RemediationSchemaVersion, ProblemID: report.ProblemID, ReportDigest: report.ReportDigest, OptionID: "resolve-gate", Operator: "operator-1", RiskConfirmed: true, Note: "root cause fixed", RecordedAt: "2026-09-27T00:00:00Z"}
	return store, report, response
}

func TestPersistRemediationReportIsIdempotentAndPreservesConflicts(t *testing.T) {
	store, state, report := newAwaitingRemediationFixture(t)

	first, err := store.PersistRemediationReport(state, report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.PersistRemediationReport(state, report)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("idempotent report write changed reference: %#v vs %#v", first, second)
	}

	conflict := report
	conflict.Problem.Detail = "conflicting history must not overwrite the original"
	if _, err := store.PersistRemediationReport(state, conflict); err == nil || !strings.Contains(err.Error(), "conflicts with preserved problem history") {
		t.Fatalf("expected preserved-history conflict, got %v", err)
	}
}

func TestReadRemediationResponseStrictlyValidatesDigestOptionAndJSON(t *testing.T) {
	store, _, report := newAwaitingRemediationFixture(t)
	option := report.Options[0]
	valid := RemediationResponse{
		SchemaVersion: RemediationSchemaVersion,
		ProblemID:     report.ProblemID,
		ReportDigest:  report.ReportDigest,
		OptionID:      option.ID,
		Operator:      "operator-1",
		RiskConfirmed: true,
		RecordedAt:    "2026-09-27T00:00:00Z",
	}

	writeRemediationResponse(t, store, report, marshalJSON(t, valid))
	if _, err := store.ReadRemediationResponse("task-1", report); err != nil {
		t.Fatal(err)
	}

	unknownField := marshalJSON(t, valid)
	unknownField = []byte(strings.TrimSuffix(string(unknownField), "\n"))
	unknownField = append(unknownField[:len(unknownField)-1], []byte(`,"unexpected":true}`)...)
	writeRemediationResponse(t, store, report, unknownField)
	if _, err := store.ReadRemediationResponse("task-1", report); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}

	writeRemediationResponse(t, store, report, append(marshalJSON(t, valid), []byte("{}")...))
	if _, err := store.ReadRemediationResponse("task-1", report); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("expected trailing-data rejection, got %v", err)
	}

	invalid := valid
	invalid.ReportDigest = "stale-digest"
	writeRemediationResponse(t, store, report, marshalJSON(t, invalid))
	if _, err := store.ReadRemediationResponse("task-1", report); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected stale-digest rejection, got %v", err)
	}

	invalid = valid
	invalid.OptionID = "not-an-option"
	writeRemediationResponse(t, store, report, marshalJSON(t, invalid))
	if _, err := store.ReadRemediationResponse("task-1", report); err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("expected unknown-option rejection, got %v", err)
	}
}

func TestConsumeRemediationResponseIsOneShotAndRetainsHistory(t *testing.T) {
	store, _, report := newAwaitingRemediationFixture(t)
	response := RemediationResponse{
		SchemaVersion: RemediationSchemaVersion,
		ProblemID:     report.ProblemID,
		ReportDigest:  report.ReportDigest,
		OptionID:      report.Options[0].ID,
		Operator:      "operator-1",
		RiskConfirmed: true,
		RecordedAt:    "2026-09-27T00:00:00Z",
	}
	writeRemediationResponse(t, store, report, marshalJSON(t, response))

	updated, err := store.ConsumeRemediationResponse("task-1", response)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Automation.Cursor.Result != "remediation-response-consumed" {
		t.Fatalf("cursor result = %q", updated.Automation.Cursor.Result)
	}
	if _, err := store.ConsumeRemediationResponse("task-1", response); err == nil || !strings.Contains(err.Error(), "no human remediation response is pending") {
		t.Fatalf("expected duplicate response rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(report.ReportPath))); err != nil {
		t.Fatalf("remediation report was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(report.HumanResponsePath))); err != nil {
		t.Fatalf("response history was removed: %v", err)
	}
}

func newAwaitingRemediationFixture(t *testing.T) (*Store, RuntimeState, RemediationReport) {
	t.Helper()
	store := NewStore(t.TempDir())
	state := compatibleState()
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	problem := RemediationProblem{
		TaskID:            "task-1",
		WorkItemID:        "wi-0001",
		AttemptID:         "attempt-1",
		EventSequence:     7,
		Category:          "runner",
		Detail:            "agent result is unavailable",
		EvidenceReference: "sessions/session-1/output.json",
		FailureDigest:     "frozen-failure-digest",
	}
	report := NewRemediationReport(problem, nil, RemediationAwaitingHuman, 0, DefaultRemediationOptions(problem, true))
	if _, err := store.AwaitRemediation("task-1", report); err != nil {
		t.Fatal(err)
	}
	_, report, err := store.ReadPendingRemediation("task-1")
	if err != nil {
		t.Fatal(err)
	}
	return store, state, report
}

func writeRemediationResponse(t *testing.T, store *Store, report RemediationReport, content []byte) {
	t.Helper()
	path := filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(report.HumanResponsePath))
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func marshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
