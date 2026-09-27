package execution

import (
	"os"
	"path/filepath"
	"testing"

	"aiw/internal/session"
	"aiw/internal/workflow"
)

func TestExecutionReportEvidencePathUsesCanonicalRuntimeTaskDirectory(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	if err := os.Chdir(tmp); err != nil { t.Fatal(err) }
	defer os.Chdir(old)

	got, err := executionReportEvidencePath("task-1", "reports/executions/report.json")
	if err != nil { t.Fatal(err) }
	want := filepath.ToSlash(filepath.Join(".ai", "tasks", "task-1", "reports", "executions", "report.json"))
	if got != want {
		t.Fatalf("execution report path = %q, want %q", got, want)
	}
}

func TestSupervisorSessionOutcomeRejectsUnknownResult(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	if err := os.Chdir(tmp); err != nil { t.Fatal(err) }
	defer os.Chdir(old)
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
	if _, err := workflow.NewStore("").Create(state); err != nil { t.Fatal(err) }
	if _, err := session.NewStore("").Create("session-1", "session", tmp, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	request := &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", Workspace: "."}
	if _, err := recordSupervisorSessionOutcome(workflow.NewStore(""), request); err == nil { t.Fatal("expected incomplete result rejection") }
}

func TestSupervisorSessionOutcomeRequiresBinding(t *testing.T) {
	if _, err := recordSupervisorSessionOutcome(workflow.NewStore(t.TempDir()), &workflow.PreparedAgentRequest{}); err == nil {
		t.Fatal("expected missing binding rejection")
	}
}

func TestSupervisorObservesLateResultFromSameSessionTurn(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	if err := os.Chdir(tmp); err != nil { t.Fatal(err) }
	defer os.Chdir(old)
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
	store := workflow.NewStore("")
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	sessions := session.NewStore("")
	if _, err := sessions.Create("session-1", "session", tmp, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	request := &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", ExpectedSessionTurn: 1, DispatchedAt: "2026-09-27T00:00:00Z", Workspace: "."}
	if _, err := recordSupervisorSessionOutcome(store, request); err == nil { t.Fatal("incomplete Session result was accepted") }
	if _, err := sessions.Update("session-1", func(status *session.Status) error {
		status.Task = &session.ManagedExecutionRef{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1"}
		status.Session.LastTurn = 1
		status.Result.Status = "completed"
		status.Result.FinalOutputFile = "outputs/0001-final.txt"
		return nil
	}); err != nil { t.Fatal(err) }
	outputDir := filepath.Join(tmp, ".ai", "sessions", "session-1", "outputs")
	if err := os.MkdirAll(outputDir, 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(outputDir, "0001-final.txt"), []byte(`{"outcome":"completed","detail":"done"}`), 0o644); err != nil { t.Fatal(err) }
	outcome, err := recordSupervisorSessionOutcome(store, request)
	if err != nil { t.Fatal(err) }
	if outcome.Kind != workflow.SupervisedOutcomeCompleted || outcome.EvidenceReference == "" { t.Fatalf("late bound result not observed: %+v", outcome) }
}
