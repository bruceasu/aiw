package workflow

import (
	"fmt"
	"strings"
	"time"
)

// RepairUndispatchedCompilePlan fills a legacy missing snapshot, without
// replacing an existing plan or changing the Attempt and model selection.
func (s *Store) RepairUndispatchedCompilePlan(id TaskID, attemptID AttemptID, plan RoutingPlan) (RuntimeState, error) {
	if err := plan.Validate(); err != nil {
		return RuntimeState{}, err
	}
	if plan.TaskID != id || !plan.Compile.Available {
		return RuntimeState{}, fmt.Errorf("repair requires an available compile plan for Task %s", id)
	}
	return s.UpdateWithEvent(id, Event{Type: "compile-plan.repaired", AttemptID: attemptID, Detail: plan.Reference().SHA256}, func(state *RuntimeState) error {
		request := state.Automation.PreparedRequest
		if request == nil || request.AttemptID != attemptID || request.DispatchedAt != "" ||
			state.WriteLease == nil || state.WriteLease.AttemptID != attemptID {
			return fmt.Errorf("compile plan repair requires an undispatched owning request")
		}
		supervisor := state.Automation.Supervisor
		if supervisor.LeaseID != "" {
			expires, err := time.Parse(time.RFC3339, supervisor.LeaseExpiresAt)
			if err != nil || expires.After(time.Now().UTC()) || !strings.HasSuffix(supervisor.Result, "paused") {
				return fmt.Errorf("supervisor must be paused with an expired lease before repair")
			}
		}
		if request.CompilerResult != nil {
			return fmt.Errorf("cannot repair a request with compiler evidence")
		}
		if compile := request.Compile; compile != nil && (compile.Plan != nil || compile.Request != nil || compile.Result != nil || compile.Failures != 0 || compile.RepairPending) {
			return fmt.Errorf("cannot replace a frozen or started compile plan")
		}
		request.Compile = &SupervisedCompileState{Plan: &plan.Compile, PlanReference: plan.Reference()}
		gateID := GateID("compile-plan-missing-" + string(request.WorkItemID))
		for index := range state.Gates {
			if state.Gates[index].ID == gateID && state.Gates[index].State == GateOpen {
				state.Gates[index].State = GateResolved
			}
		}
		state.Automation.Cursor = AutomationCursor{Result: "agent-request-prepared", Detail: string(request.WorkItemID), RecordedAt: time.Now().UTC().Format(time.RFC3339)}
		return nil
	})
}
