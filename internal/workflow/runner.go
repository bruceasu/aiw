package workflow

import "fmt"

// RunnerOutcomeKind identifies the one durable outcome a bounded Runner may
// report. Adapters decide whether a prepared request is only shown or consumed
// by an explicitly authorized agent handoff.
type RunnerOutcomeKind string

const (
	RunnerRepair   RunnerOutcomeKind = "repair-required"
	RunnerGate     RunnerOutcomeKind = "gate-required"
	RunnerPrepared RunnerOutcomeKind = "agent-request-prepared"
	RunnerBlocked  RunnerOutcomeKind = "blocked"
	RunnerNoWork   RunnerOutcomeKind = "no-executable-work"
)

// RunnerOutcome is a pure view of persisted workflow state. It never starts
// an Attempt or performs an external operation.
type RunnerOutcome struct {
	Kind    RunnerOutcomeKind
	Detail  string
	Repair  *ProjectionRepair
	Gate    *Gate
	Request *PreparedAgentRequest
}

// NextRunnerOutcome prioritizes durable blockers ahead of execution. The
// caller synchronizes checklist references before using this function.
func NextRunnerOutcome(state RuntimeState) RunnerOutcome {
	for index := range state.Automation.ProjectionRepairs {
		repair := &state.Automation.ProjectionRepairs[index]
		if repair.ResolvedAt == "" {
			return RunnerOutcome{Kind: RunnerRepair, Detail: repair.Target, Repair: repair}
		}
	}
	for index := range state.Gates {
		gate := &state.Gates[index]
		if gate.State == GateOpen {
			return RunnerOutcome{Kind: RunnerGate, Detail: string(gate.ID), Gate: gate}
		}
	}
	if request := state.Automation.PreparedRequest; request != nil && state.WriteLease != nil && state.WriteLease.AttemptID == request.AttemptID {
		return RunnerOutcome{Kind: RunnerPrepared, Detail: string(request.WorkItemID), Request: request}
	}
	if state.WriteLease != nil {
		return RunnerOutcome{Kind: RunnerBlocked, Detail: fmt.Sprintf("workspace write lease is active for %s", state.WriteLease.AttemptID)}
	}
	if _, err := SelectReadyMappedWorkItem(state); err != nil {
		return RunnerOutcome{Kind: RunnerNoWork, Detail: err.Error()}
	}
	return RunnerOutcome{}
}
