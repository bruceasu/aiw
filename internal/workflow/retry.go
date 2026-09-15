package workflow

import (
	"fmt"
	"strings"
)

// SetRetryPolicy updates the automatic Attempt budget for one non-running
// Work Item. Lowering a ready item's budget below its accumulated no-progress
// count blocks it immediately instead of permitting one more Attempt.
func (s *Store) SetRetryPolicy(id TaskID, workItemID WorkItemID, policy RetryPolicy) (RuntimeState, error) {
	if workItemID == "" {
		return RuntimeState{}, fmt.Errorf("work item id is required")
	}
	if err := validateRetryPolicy(policy); err != nil {
		return RuntimeState{}, err
	}
	return s.UpdateWithEvent(id, Event{Type: "work-item.retry-policy-set", WorkItemID: workItemID, Detail: fmt.Sprintf("max_attempts=%d", policy.MaxAttempts)}, func(state *RuntimeState) error {
		for index := range state.WorkItems {
			item := &state.WorkItems[index]
			if item.ID != workItemID {
				continue
			}
			if item.State != WorkItemReady && item.State != WorkItemBlocked {
				return fmt.Errorf("work item %s must be ready or blocked to change its retry policy", workItemID)
			}
			item.RetryPolicy = policy
			if item.State == WorkItemReady && item.NoProgressCount >= policy.MaxAttempts {
				if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemBlocked); err != nil {
					return err
				}
				item.State = WorkItemBlocked
			}
			return nil
		}
		return fmt.Errorf("unknown work item %s", workItemID)
	})
}

// ReopenWorkItem explicitly returns a blocked Work Item to the ready queue
// after its relevant Gates have been resolved. An exhausted retry budget is
// reset; a blocked outcome that did not exhaust the budget preserves its
// count. The reason is retained in event history for operator auditability.
func (s *Store) ReopenWorkItem(id TaskID, workItemID WorkItemID, reason string) (RuntimeState, error) {
	if workItemID == "" {
		return RuntimeState{}, fmt.Errorf("work item id is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return RuntimeState{}, fmt.Errorf("reopen reason is required")
	}
	return s.UpdateWithEvent(id, Event{Type: "work-item.reopened", WorkItemID: workItemID, Detail: reason}, func(state *RuntimeState) error {
		for index := range state.WorkItems {
			item := &state.WorkItems[index]
			if item.ID != workItemID {
				continue
			}
			if item.State != WorkItemBlocked {
				return fmt.Errorf("work item %s is not blocked", workItemID)
			}
			if hasOpenRecoveryGate(*state, workItemID) {
				return fmt.Errorf("work item %s has an unresolved Gate; resolve it before reopening", workItemID)
			}
			if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemReady); err != nil {
				return err
			}
			item.State = WorkItemReady
			if item.NoProgressCount >= item.RetryPolicy.MaxAttempts {
				item.NoProgressCount = 0
			}
			return nil
		}
		return fmt.Errorf("unknown work item %s", workItemID)
	})
}

func hasOpenRecoveryGate(state RuntimeState, workItemID WorkItemID) bool {
	for _, gate := range state.Gates {
		if gate.State == GateOpen && (gate.WorkItemID == "" || gate.WorkItemID == workItemID) {
			return true
		}
	}
	return false
}
