package workflow

import (
	"fmt"
	"strings"
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

// RecordAttemptOutcome is the compatibility path for unsupervised callers.
// Only a failed call consumes no-progress; a successful call releases the
// Work Item for its authoritative checklist projection.
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
				if succeeded {
					if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemReady); err != nil {
						return err
					}
					item.State = WorkItemReady
				} else {
					item.NoProgressCount++
					item.LastOutputReference = attemptOutputReference(*attempt, state.Automation.PreparedRequest)
					if item.NoProgressCount >= item.RetryPolicy.MaxAttempts {
						if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemBlocked); err != nil {
							return err
						}
						item.State = WorkItemBlocked
					} else if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemReady); err != nil {
						return err
					} else {
						item.State = WorkItemReady
					}
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

// RecordSupervisedOutcome closes a supervised Attempt from a typed Agent
// result. Completion remains an adapter/Core concern: this method deliberately
// returns the Work Item to ready so checklist synchronization is the only
// operation that can mark it completed.
func (s *Store) RecordSupervisedOutcome(id TaskID, attemptID AttemptID, outcome SupervisedOutcome) (RuntimeState, error) {
	if err := ValidateSupervisedOutcome(outcome); err != nil {
		return RuntimeState{}, err
	}
	updated, err := s.UpdateWithEvent(id, Event{Type: "attempt.outcome." + string(outcome.Kind), AttemptID: attemptID, Detail: outcome.Detail}, func(state *RuntimeState) error {
		for index := range state.Attempts {
			attempt := &state.Attempts[index]
			if attempt.ID != attemptID {
				continue
			}
			if attempt.State != AttemptRunning {
				return fmt.Errorf("attempt %s is %s, expected running", attemptID, attempt.State)
			}
			outcome.RecordedAt = time.Now().UTC().Format(time.RFC3339)
			attempt.Outcome = &outcome
			attempt.EndedAt = outcome.RecordedAt
			if outcome.Kind == SupervisedOutcomeCompleted {
				attempt.State = AttemptCompleted
			} else {
				attempt.State = AttemptFailed
			}
			if state.WriteLease != nil && state.WriteLease.AttemptID == attemptID {
				state.WriteLease = nil
			}
			if state.Automation.PreparedRequest != nil && state.Automation.PreparedRequest.AttemptID == attemptID {
				state.Automation.PreparedRequest = nil
			}
			for itemIndex := range state.WorkItems {
				item := &state.WorkItems[itemIndex]
				if item.ID != attempt.WorkItemID {
					continue
				}
				switch outcome.Kind {
				case SupervisedOutcomeCompleted:
					if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemReady); err != nil {
						return err
					}
					item.State = WorkItemReady
				case SupervisedOutcomeBlocked:
					if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemBlocked); err != nil {
						return err
					}
					item.State = WorkItemBlocked
					item.LastOutputReference = outcome.EvidenceReference
					openOutcomeGate(state, item.ID, outcome)
				case SupervisedOutcomeNoProgress:
					item.NoProgressCount++
					item.LastOutputReference = outcome.EvidenceReference
					if item.NoProgressCount >= item.RetryPolicy.MaxAttempts {
						if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemBlocked); err != nil {
							return err
						}
						item.State = WorkItemBlocked
					} else if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemReady); err != nil {
						return err
					} else {
						item.State = WorkItemReady
					}
				}
				return nil
			}
			return fmt.Errorf("attempt %s references unknown work item %s", attemptID, attempt.WorkItemID)
		}
		return fmt.Errorf("unknown attempt %s", attemptID)
	})
	if err != nil || outcome.Kind == SupervisedOutcomeCompleted {
		return updated, err
	}
	attempt, ok := findAttempt(updated.Attempts, attemptID)
	if !ok {
		return RuntimeState{}, fmt.Errorf("recorded outcome attempt %s is missing", attemptID)
	}
	if _, err := s.PersistFailureReport(updated, failureReportForOutcome(updated, attempt, outcome)); err != nil {
		return updated, err
	}
	return updated, nil
}

func ValidateSupervisedOutcome(outcome SupervisedOutcome) error {
	if outcome.Kind != SupervisedOutcomeCompleted && outcome.Kind != SupervisedOutcomeBlocked && outcome.Kind != SupervisedOutcomeNoProgress {
		return fmt.Errorf("unsupported supervised outcome: %s", outcome.Kind)
	}
	if strings.TrimSpace(outcome.EvidenceReference) == "" {
		return fmt.Errorf("supervised outcome evidence reference is required")
	}
	if outcome.Kind == SupervisedOutcomeBlocked && outcome.BlockedCategory == "" {
		return fmt.Errorf("blocked supervised outcome category is required")
	}
	if outcome.Kind != SupervisedOutcomeBlocked && outcome.BlockedCategory != "" {
		return fmt.Errorf("only blocked supervised outcomes may have a category")
	}
	return nil
}

func openOutcomeGate(state *RuntimeState, workItemID WorkItemID, outcome SupervisedOutcome) {
	gateID := GateID("supervised-" + string(outcome.BlockedCategory) + "-" + string(workItemID))
	for index := range state.Gates {
		if state.Gates[index].ID != gateID {
			continue
		}
		state.Gates[index].State = GateOpen
		state.Gates[index].Reason = outcome.Detail
		return
	}
	state.Gates = append(state.Gates, Gate{ID: gateID, WorkItemID: workItemID, Kind: outcomeGateKind(outcome.BlockedCategory), State: GateOpen, Reason: outcome.Detail})
}

func outcomeGateKind(category BlockedOutcomeCategory) GateKind {
	switch category {
	case BlockedOutcomeWorkspaceAccess:
		return GateWorkspaceAccess
	case BlockedOutcomeAuthorization:
		return GateAuthorization
	case BlockedOutcomeDependency:
		return GateDependency
	case BlockedOutcomeValidation:
		return GateValidation
	default:
		return GateDecision
	}
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
