package cli

import (
	"os"
	"strings"
	"testing"

	"aiw/internal/task"
	"aiw/internal/workflow"
)

func TestSupervisorCompileGuidanceDistinguishesRecovery(t *testing.T) {
	base := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
	for _, tc := range []struct {
		name, gateID, wantState, wantNext string
	}{
		{"missing plan", "compile-plan-missing-wi-0001", "plan-missing", "aiw wf diagnose task-1"},
		{"unavailable target", "compile-target-unavailable-wi-0001", "target-unavailable", "repair the compile target"},
		{"repair exhausted", "compiler-repair-limit-wi-0001", "repair-limit-reached", "repair manually"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := base
			state.Gates = []workflow.Gate{{ID: workflow.GateID(tc.gateID), State: workflow.GateOpen}}
			got, next, _ := supervisorCompileGuidance(state)
			if got != tc.wantState || !strings.Contains(next, tc.wantNext) {
				t.Fatalf("guidance = %q, %q; want %q containing %q", got, next, tc.wantState, tc.wantNext)
			}
		})
	}
	prepared := base
	prepared.Automation.PreparedRequest = &workflow.PreparedAgentRequest{Compile: &workflow.SupervisedCompileState{RepairPending: true}}
	if got, next, _ := supervisorCompileGuidance(prepared); got != "repair-prepared" || !strings.Contains(next, "supervise task-1 start") {
		t.Fatalf("prepared repair guidance = %q, %q", got, next)
	}
	unknown := base
	unknown.Automation.Supervisor.Result = "compiler-paused"
	if got, next, _ := supervisorCompileGuidance(unknown); got != "result-unknown" || !strings.Contains(next, "do not rerun") {
		t.Fatalf("unknown compiler guidance = %q, %q", got, next)
	}
}

func TestWorkflowSupervisorRejectsInvalidInputBeforeRuntimeAccess(t *testing.T) {
	if err := runWorkflowSupervisor([]string{"supervise", "bad/id", "start"}); err == nil {
		t.Fatal("expected invalid Task ID rejection")
	}
	if err := runWorkflowSupervisor([]string{"supervise", "task-1", "unknown"}); err == nil {
		t.Fatal("expected invalid action rejection")
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
	if _, err := os.Stat(task.TaskDir("task-1")); !os.IsNotExist(err) {
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
