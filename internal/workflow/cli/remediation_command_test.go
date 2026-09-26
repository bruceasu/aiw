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
