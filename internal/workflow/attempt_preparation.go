package workflow

import (
	"fmt"
	"time"
)

// RepairMissingSessionAttempt repairs the legacy preparation failure before
// a Session or prepared request existed. The command adapter must first verify
// that the owning Session is absent. This does not report an Agent outcome.
func (s *Store) RepairMissingSessionAttempt(id TaskID, attemptID AttemptID) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "attempt.preparation-repaired", AttemptID: attemptID, Detail: "Session missing before request preparation"}, func(state *RuntimeState) error {
		if state.WriteLease == nil || state.WriteLease.AttemptID != attemptID || state.Automation.PreparedRequest != nil {
			return fmt.Errorf("attempt %s is not an unprepared lease owner", attemptID)
		}
		supervisor := state.Automation.Supervisor
		if supervisor.Result != "runner-paused" && supervisor.Result != "supervisor-paused" {
			return fmt.Errorf("supervisor must be paused before repairing preparation")
		}
		if supervisor.LeaseExpiresAt != "" {
			expires, err := time.Parse(time.RFC3339, supervisor.LeaseExpiresAt)
			if err != nil || expires.After(time.Now().UTC()) {
				return fmt.Errorf("supervisor lease has not verifiably expired")
			}
		}
		for index := range state.Attempts {
			attempt := &state.Attempts[index]
			if attempt.ID != attemptID {
				continue
			}
			if attempt.State != AttemptRunning || attempt.SessionID == "" || attempt.Handoff != (Handoff{}) || attempt.Outcome != nil {
				return fmt.Errorf("attempt %s is not an untouched preparation", attemptID)
			}
			for itemIndex := range state.WorkItems {
				item := &state.WorkItems[itemIndex]
				if item.ID != attempt.WorkItemID {
					continue
				}
				if item.State != WorkItemRunning {
					return fmt.Errorf("work item %s is not running", item.ID)
				}
				attempt.State = AttemptCancelled
				attempt.EndedAt = time.Now().UTC().Format(time.RFC3339)
				item.State = WorkItemReady
				state.WriteLease = nil
				state.Automation.Cursor = AutomationCursor{Result: "preparation-repaired", Detail: string(attemptID), RecordedAt: attempt.EndedAt}
				return nil
			}
			return fmt.Errorf("attempt %s has no Work Item", attemptID)
		}
		return fmt.Errorf("unknown attempt %s", attemptID)
	})
}
