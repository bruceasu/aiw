package workflow

import (
	"fmt"
	"time"
)

// StartAttempt claims a ready Work Item and its Task write lease as one
// runtime update. Session is intentionally represented only by its stable ID;
// the Session package remains independent of Workflow Core.
func (s *Store) StartAttempt(id TaskID, attempt Attempt) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "attempt.started", WorkItemID: attempt.WorkItemID, AttemptID: attempt.ID}, func(state *RuntimeState) error {
		if attempt.ID == "" || attempt.WorkItemID == "" || attempt.Workspace == "" {
			return fmt.Errorf("attempt id, work item id, and workspace are required")
		}
		if state.WriteLease != nil {
			return fmt.Errorf("workspace write lease is held by attempt %s", state.WriteLease.AttemptID)
		}
		for _, existing := range state.Attempts {
			if existing.ID == attempt.ID {
				return fmt.Errorf("attempt already exists: %s", attempt.ID)
			}
		}
		for index := range state.WorkItems {
			item := &state.WorkItems[index]
			if item.ID != attempt.WorkItemID {
				continue
			}
			if item.NoProgressCount >= item.RetryPolicy.MaxAttempts {
				return fmt.Errorf("work item %s exhausted its retry limit; reopen it before starting another attempt", item.ID)
			}
			if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemLeased); err != nil {
				return err
			}
			item.State = WorkItemLeased
			if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemRunning); err != nil {
				return err
			}
			item.State = WorkItemRunning
			attempt.State = AttemptRunning
			if attempt.StartedAt == "" {
				attempt.StartedAt = time.Now().UTC().Format(time.RFC3339)
			}
			state.Attempts = append(state.Attempts, attempt)
			state.WriteLease = &WriteLease{AttemptID: attempt.ID, Workspace: attempt.Workspace, AcquiredAt: attempt.StartedAt}
			return nil
		}
		return fmt.Errorf("attempt %s references unknown work item %s", attempt.ID, attempt.WorkItemID)
	})
}

// RecordAttemptOutcome closes an Attempt after a Session turn. An Attempt
// outcome does not complete its Work Item, so either outcome consumes that
// Work Item's no-progress budget until checklist completion is recorded.
func (s *Store) RecordAttemptOutcome(id TaskID, attemptID AttemptID, succeeded bool) (RuntimeState, error) {
	outcome := "failed"
	if succeeded {
		outcome = "completed"
	}
	return s.UpdateWithEvent(id, Event{Type: "attempt." + outcome, AttemptID: attemptID}, func(state *RuntimeState) error {
		for index := range state.Attempts {
			attempt := &state.Attempts[index]
			if attempt.ID != attemptID {
				continue
			}
			if attempt.State != AttemptRunning {
				return TransitionError{Entity: "attempt", ID: string(attempt.ID), From: string(attempt.State), To: "outcome"}
			}
			if succeeded {
				attempt.State = AttemptCompleted
			} else {
				attempt.State = AttemptFailed
			}
			attempt.EndedAt = time.Now().UTC().Format(time.RFC3339)
			for itemIndex := range state.WorkItems {
				item := &state.WorkItems[itemIndex]
				if item.ID != attempt.WorkItemID {
					continue
				}
				item.NoProgressCount++
				item.LastOutputReference = attemptOutputReference(*attempt, state.Automation.PreparedRequest)
				if item.NoProgressCount >= item.RetryPolicy.MaxAttempts {
					if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemBlocked); err != nil {
						return err
					}
					item.State = WorkItemBlocked
				} else {
					if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemReady); err != nil {
						return err
					}
					item.State = WorkItemReady
				}
				break
			}
			if state.WriteLease != nil && state.WriteLease.AttemptID == attemptID {
				state.WriteLease = nil
			}
			if request := state.Automation.PreparedRequest; request != nil && request.AttemptID == attemptID {
				state.Automation.PreparedRequest = nil
			}
			return nil
		}
		return fmt.Errorf("unknown attempt %s", attemptID)
	})
}

func attemptOutputReference(attempt Attempt, request *PreparedAgentRequest) string {
	if attempt.Handoff.ArtifactPath != "" {
		return attempt.Handoff.ArtifactPath
	}
	if request != nil && request.AttemptID == attempt.ID {
		return request.Handoff
	}
	return ""
}
