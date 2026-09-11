package workflow

import "fmt"

// SelectReadyMappedWorkItem returns the first checklist-mapped item that can
// safely receive a managed Attempt. It is deliberately pure so command and
// agent adapters share one readiness interpretation.
func SelectReadyMappedWorkItem(state RuntimeState) (WorkItem, error) {
	for _, item := range state.WorkItems {
		if item.Checklist.Item == "" || item.State != WorkItemReady {
			continue
		}
		if workItemBlocked(state, item) {
			continue
		}
		return item, nil
	}
	return WorkItem{}, fmt.Errorf("no ready mapped Work Item")
}

func HasMappedWorkItems(state RuntimeState) bool {
	for _, item := range state.WorkItems {
		if item.Checklist.Item != "" {
			return true
		}
	}
	return false
}

func workItemBlocked(state RuntimeState, item WorkItem) bool {
	for _, dependency := range item.Dependencies {
		for _, candidate := range state.WorkItems {
			if candidate.ID == dependency && candidate.State != WorkItemCompleted {
				return true
			}
		}
	}
	for _, gate := range state.Gates {
		if gate.State == GateOpen && (gate.WorkItemID == "" || gate.WorkItemID == item.ID) {
			return true
		}
	}
	return false
}
