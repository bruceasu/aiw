package workflow

import "testing"

func TestNextRunnerOutcomePrioritizesRepairThenGate(t *testing.T) {
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Checklist: ChecklistReference{Item: "1.1"}, State: WorkItemReady}}
	state.Gates = []Gate{{ID: "gate-1", WorkItemID: "wi-0001", Kind: GateDecision, State: GateOpen}}
	state.Automation.ProjectionRepairs = []ProjectionRepair{{EventSequence: 2, Target: "tasks.md", Recommended: "repair"}}
	if outcome := NextRunnerOutcome(state); outcome.Kind != RunnerRepair || outcome.Detail != "tasks.md" {
		t.Fatalf("repair outcome = %#v", outcome)
	}
	state.Automation.ProjectionRepairs[0].ResolvedAt = "now"
	if outcome := NextRunnerOutcome(state); outcome.Kind != RunnerGate || outcome.Detail != "gate-1" {
		t.Fatalf("gate outcome = %#v", outcome)
	}
}

func TestNextRunnerOutcomeReportsPreparedAndNoWork(t *testing.T) {
	state := compatibleState()
	state.Automation.PreparedRequest = &PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: "."}
	state.WriteLease = &WriteLease{AttemptID: "attempt-1", Workspace: "."}
	if outcome := NextRunnerOutcome(state); outcome.Kind != RunnerPrepared || outcome.Request != state.Automation.PreparedRequest {
		t.Fatalf("prepared outcome = %#v", outcome)
	}
	state.Automation.PreparedRequest = nil
	state.WriteLease = nil
	if outcome := NextRunnerOutcome(state); outcome.Kind != RunnerNoWork {
		t.Fatalf("no-work outcome = %#v", outcome)
	}
}
