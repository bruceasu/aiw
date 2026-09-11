package workflow

import (
	"fmt"
	"strings"
)

// ApplyLegacyStatus maps the small supported compatibility surface of the
// legacy Task status command to Workflow Core state. It never fabricates an
// Attempt, evidence, completed Work Item, or delivery outcome merely to make a
// requested display string true.
func ApplyLegacyStatus(state *RuntimeState, requested string) error {
	if state == nil {
		return fmt.Errorf("workflow state is required")
	}
	if err := ValidateRuntimeState(*state); err != nil {
		return err
	}
	switch strings.ToUpper(strings.TrimSpace(requested)) {
	case "TODO", "DRAFT":
		state.Planning = PlanningDraft
	case "READY":
		state.Planning = PlanningReady
	case "NEEDS_DECISION":
		state.Planning = PlanningNeedsDecision
	case "DONE":
		if summary := DeriveSummary(*state); summary.Status != TaskDone {
			return fmt.Errorf("cannot set DONE: Workflow Core derives %s; complete Work Items and required validation first", summary.Status)
		}
	default:
		return fmt.Errorf("unsupported legacy Task status %q; use a Workflow Core transition instead", requested)
	}
	state.Summary = DeriveSummary(*state)
	return nil
}
