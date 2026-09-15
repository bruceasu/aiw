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

func TestSupervisorDeliveryPreservesOpenGate(t *testing.T) {
	store := workflow.NewStore(t.TempDir())
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".wt/task-1", Kind: workflow.WorkspaceIsolated}, workflow.PlanningReady, workflow.DeliveryPending)
	state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", Checklist: workflow.ChecklistReference{Item: "1.1"}, Title: "accepted", State: workflow.WorkItemCompleted}}
	state.Evidence = []workflow.Evidence{{ID: "evidence-1", WorkItemID: "wi-0001", Kind: workflow.EvidenceStaticReview, State: workflow.EvidencePassed}}
	state.Gates = []workflow.Gate{{ID: "review", State: workflow.GateOpen}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if delivered, err := deliverCompletedSupervisorTask("task-1", store); err != nil || delivered {
		t.Fatalf("blocked delivery = %t, error = %v", delivered, err)
	}
	loaded, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Delivery != workflow.DeliveryPending || loaded.Gates[0].State != workflow.GateOpen {
		t.Fatal("blocked delivery changed the Task or Gate")
	}
}
