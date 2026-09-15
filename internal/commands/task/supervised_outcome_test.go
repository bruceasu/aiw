package task

import (
	"testing"

	"aiw/internal/session"
	"aiw/internal/workflow"
)

func TestParseSupervisedOutcomeRejectsOrdinaryProse(t *testing.T) {
	outcome := parseSupervisedOutcome("I could not run Git because the directory is unsafe.", "output.txt")
	if outcome.Kind != workflow.SupervisedOutcomeNoProgress || outcome.EvidenceReference != "output.txt" {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestParseSupervisedOutcomeAcceptsBlockedCategory(t *testing.T) {
	outcome := parseSupervisedOutcome(`{"outcome":"blocked","blocked_category":"dependency","detail":"waiting for 3.2"}`, "output.txt")
	if outcome.Kind != workflow.SupervisedOutcomeBlocked || outcome.BlockedCategory != workflow.BlockedOutcomeDependency {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestNormalizeSupervisorSessionOutcomeReservesWorkspaceAccessForPreflight(t *testing.T) {
	outcome := normalizeSupervisorSessionOutcome(workflow.SupervisedOutcome{
		Kind:            workflow.SupervisedOutcomeBlocked,
		BlockedCategory: workflow.BlockedOutcomeWorkspaceAccess,
		Detail:          "stale handoff said Git was unavailable",
	})
	if outcome.BlockedCategory != workflow.BlockedOutcomeUnknown {
		t.Fatalf("blocked category = %q", outcome.BlockedCategory)
	}
	if outcome.Detail == "stale handoff said Git was unavailable" {
		t.Fatalf("detail did not explain the Core-only classification: %q", outcome.Detail)
	}
}

func TestValidateSupervisorSessionResultRejectsPriorAttempt(t *testing.T) {
	request := &workflow.PreparedAgentRequest{SessionID: "session-1", AttemptID: "attempt-new", ExpectedSessionTurn: 9}
	status := session.Status{Session: session.SessionInfo{LastTurn: 8}, Task: &session.ManagedExecutionRef{AttemptID: "attempt-old"}}
	if err := validateSupervisorSessionResult(status, request); err == nil {
		t.Fatal("expected stale Session result to be rejected")
	}
}

func TestRecordSupervisorSessionOutcomeRejectsUndispatchedRequest(t *testing.T) {
	request := &workflow.PreparedAgentRequest{SessionID: "session-1", AttemptID: "attempt-1"}
	if _, err := recordSupervisorSessionOutcome(workflow.NewStore(t.TempDir()), request); err == nil {
		t.Fatal("expected undispatched request to be rejected")
	}
}
