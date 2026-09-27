package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aiw/internal/workflow"
)

func TestContinueAndResumeShareSafePendingResponsePath(t *testing.T) {
	taskAdapter = testWorkflowAdapter()
	store := workflow.NewStore(t.TempDir())
	state := workflow.NewCompatibleRuntime(
		workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary},
		workflow.PlanningReady,
		workflow.DeliveryUnmanaged,
	)
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	problem := workflow.RemediationProblem{
		TaskID:            "task-1",
		WorkItemID:        "wi-0001",
		AttemptID:         "attempt-1",
		EventSequence:     7,
		Category:          "runner",
		Detail:            "operator decision is required",
		EvidenceReference: "sessions/session-1/output.json",
		FailureDigest:     "frozen-failure-digest",
	}
	report := workflow.NewRemediationReport(problem, nil, workflow.RemediationAwaitingHuman, 0, workflow.DefaultRemediationOptions(problem, true))
	if _, err := store.AwaitRemediation("task-1", report); err != nil {
		t.Fatal(err)
	}
	_, pending, err := store.ReadPendingRemediation("task-1")
	if err != nil {
		t.Fatal(err)
	}
	responsePath := filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(pending.HumanResponsePath))
	if err := os.Remove(responsePath); err != nil {
		t.Fatal(err)
	}
	before, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := runRemediationResponseWithStore([]string{"continue", "task-1"}, store); err != nil {
		t.Fatal(err)
	}
	if err := runRemediationResponseWithStore([]string{"resume", "task-1"}, store); err != nil {
		t.Fatal(err)
	}
	afterMissing, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if afterMissing.LastEventSequence != before.LastEventSequence || afterMissing.Automation.Cursor.Result != "awaiting-human" {
		t.Fatalf("missing response mutated state: before=%#v after=%#v", before.Automation.Cursor, afterMissing.Automation.Cursor)
	}

	response := workflow.RemediationResponse{
		SchemaVersion: workflow.RemediationSchemaVersion,
		ProblemID:     pending.ProblemID,
		ReportDigest:  pending.ReportDigest,
		OptionID:      "human-review",
		Operator:      "operator-1",
		RiskConfirmed: true,
		RecordedAt:    "2026-09-27T00:00:00Z",
	}
	content, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(responsePath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runRemediationResponseWithStore([]string{"continue", "task-1"}, store); err != nil {
		t.Fatal(err)
	}
	if err := runRemediationResponseWithStore([]string{"resume", "task-1"}, store); err != nil {
		t.Fatal(err)
	}
	afterHumanReview, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if afterHumanReview.LastEventSequence != afterMissing.LastEventSequence || afterHumanReview.Automation.Cursor.Result != "awaiting-human" {
		t.Fatalf("human-review response was consumed or changed state: %#v", afterHumanReview.Automation.Cursor)
	}
}

func TestUnknownSessionResumeRejectsStaleRequest(t *testing.T) {
	store := workflow.NewStore(t.TempDir())
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	report := workflow.RemediationReport{Problem: workflow.RemediationProblem{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", SessionTurn: 3, Category: "session-result-unknown"}}
	if err := validateUnknownSessionResume(store, report); err == nil { t.Fatal("missing dispatched request was accepted") }
}

func TestFailedRepairKeepsHumanResponsePending(t *testing.T) {
	taskAdapter = testWorkflowAdapter()
	store := workflow.NewStore(t.TempDir())
	state := workflow.NewCompatibleRuntime(
		workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary},
		workflow.PlanningReady,
		workflow.DeliveryUnmanaged,
	)
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	problem := workflow.RemediationProblem{
		TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1",
		EventSequence: 7, Category: "projection", Detail: "projection repair required",
		EvidenceReference: "tasks.md", FailureDigest: "frozen-failure-digest",
	}
	report := workflow.NewRemediationReport(problem, nil, workflow.RemediationAwaitingHuman, 0, workflow.DefaultRemediationOptions(problem, false))
	if _, err := store.AwaitRemediation("task-1", report); err != nil {
		t.Fatal(err)
	}
	_, pending, err := store.ReadPendingRemediation("task-1")
	if err != nil {
		t.Fatal(err)
	}
	response := workflow.RemediationResponse{
		SchemaVersion: workflow.RemediationSchemaVersion,
		ProblemID: pending.ProblemID,
		ReportDigest: pending.ReportDigest,
		OptionID: "repair-projection",
		Operator: "operator-1",
		RiskConfirmed: true,
		RecordedAt: "2026-09-27T00:00:00Z",
	}
	content, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	responsePath := filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(pending.HumanResponsePath))
	if err := os.WriteFile(responsePath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runRemediationResponseWithStore([]string{"continue", "task-1"}, store); err == nil {
		t.Fatal("expected repair to fail without a matching pending projection")
	}
	after, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if after.Automation.Cursor.Result != "awaiting-human" {
		t.Fatalf("failed repair consumed response: %#v", after.Automation.Cursor)
	}
}

func TestGateDecisionReopensOnlyItsBlockedWorkItem(t *testing.T) {
	store := workflow.NewStore(t.TempDir())
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
	state.WorkItems = []workflow.WorkItem{
		{ID: "wi-0001", Title: "blocked", State: workflow.WorkItemBlocked},
		{ID: "wi-0002", Title: "other", State: workflow.WorkItemReady},
	}
	state.Gates = []workflow.Gate{{ID: "review-wi-0001", WorkItemID: "wi-0001", Kind: workflow.GateDecision, State: workflow.GateOpen, Reason: "review required"}}
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	if _, err := store.SetRetryPolicy("task-1", "wi-0002", workflow.RetryPolicy{MaxAttempts: 3}); err != nil { t.Fatal(err) }
	state, err := store.Load("task-1")
	if err != nil { t.Fatal(err) }
	report, err := workflow.BuildRemediationReportForGate(state, state.Gates[0])
	if err != nil { t.Fatal(err) }
	if _, err := store.AwaitRemediation("task-1", report); err != nil { t.Fatal(err) }
	before, report, err := store.ReadPendingRemediation("task-1")
	if err != nil { t.Fatal(err) }
	response := workflow.RemediationResponse{SchemaVersion: workflow.RemediationSchemaVersion, ProblemID: report.ProblemID, ReportDigest: report.ReportDigest, OptionID: "resolve-gate", Operator: "operator-1", RiskConfirmed: true, Note: "root cause fixed", RecordedAt: "2026-09-27T00:00:00Z"}
	updated, err := store.ApplyGateRemediationResponse("task-1", response)
	if err != nil { t.Fatal(err) }
	if updated.Gates[0].State != workflow.GateResolved || updated.WorkItems[0].State != workflow.WorkItemReady || updated.WorkItems[1].State != workflow.WorkItemReady {
		t.Fatalf("Gate decision changed the wrong state: %+v", updated)
	}
	if updated.LastEventSequence != before.LastEventSequence+1 || updated.Automation.Cursor.Result != "remediation-response-consumed" {
		t.Fatalf("Gate response was not one transition: before=%d after=%+v", before.LastEventSequence, updated)
	}
	if err := runRemediationResponseWithStore([]string{"continue", "task-1"}, store); err != nil { t.Fatalf("repeat continue should be a no-op: %v", err) }
	again, err := store.Load("task-1")
	if err != nil { t.Fatal(err) }
	if again.LastEventSequence != updated.LastEventSequence { t.Fatal("repeat continue created another event") }
}
