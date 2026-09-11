package task

import (
	"os"
	"testing"

	"aiw/internal/session"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestWorkflowSupervisorRejectsInvalidInputBeforeRuntimeAccess(t *testing.T) {
	if err := runWorkflowSupervisor([]string{"supervise", "bad/id", "start"}); err == nil {
		t.Fatal("expected invalid Task ID rejection")
	}
	if err := runWorkflowSupervisor([]string{"supervise", "task-1", "unknown"}); err == nil {
		t.Fatal("expected invalid action rejection")
	}
}

func TestSupervisorSessionEvidenceRejectsUnknownResult(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	if err := os.Chdir(tmp); err != nil { t.Fatal(err) }
	defer os.Chdir(old)
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
	if _, err := workflow.NewStore("").Create(state); err != nil { t.Fatal(err) }
	if _, err := session.NewStore("").Create("session-1", "session", tmp, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	request := &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", Workspace: "."}
	if err := recordSupervisorSessionEvidence(workflow.NewStore(""), request); err == nil { t.Fatal("expected incomplete result rejection") }
}

func TestSupervisorSessionEvidenceRequiresBinding(t *testing.T) {
	if err := recordSupervisorSessionEvidence(workflow.NewStore(t.TempDir()), &workflow.PreparedAgentRequest{}); err == nil {
		t.Fatal("expected missing binding rejection")
	}
}

func TestSupervisorSessionEvidenceIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	if err := os.Chdir(tmp); err != nil { t.Fatal(err) }
	defer os.Chdir(old)
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
	state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", Title: "work", State: workflow.WorkItemReady}}
	store := workflow.NewStore("")
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	sessions := session.NewStore("")
	if _, err := sessions.Create("session-1", "session", tmp, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	if _, err := sessions.Update("session-1", func(status *session.Status) error {
		status.Result.Status, status.Result.FinalOutputFile = "completed", "outputs/0001-final.txt"
		return nil
	}); err != nil { t.Fatal(err) }
	request := &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", Workspace: "."}
	if err := recordSupervisorSessionEvidence(store, request); err != nil { t.Fatal(err) }
	if err := recordSupervisorSessionEvidence(store, request); err != nil { t.Fatal(err) }
	current, err := store.Load("task-1")
	if err != nil || len(current.Evidence) != 1 { t.Fatalf("evidence=%d err=%v", len(current.Evidence), err) }
}

func TestWorkflowSupervisorStatusDoesNotStartAgent(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	if err := os.Chdir(tmp); err != nil { t.Fatal(err) }
	defer os.Chdir(old)
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
	if _, err := workflow.NewStore("").Create(state); err != nil { t.Fatal(err) }
	if err := runWorkflowSupervisor([]string{"supervise", "task-1", "status"}); err != nil { t.Fatal(err) }
	if _, err := os.Stat(taskx.TaskDir("task-1")); !os.IsNotExist(err) {
		t.Fatal("status must not create Task artifacts or start an agent")
	}
}
