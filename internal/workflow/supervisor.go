package workflow

import (
	"fmt"
	"strings"
	"time"
)

const supervisorLeaseTTL = time.Minute

// StartSupervisor acquires the single durable supervisor lease for a Task.
func (s *Store) StartSupervisor(id TaskID, leaseID string) (RuntimeState, error) {
	if strings.TrimSpace(leaseID) == "" {
		return RuntimeState{}, fmt.Errorf("supervisor lease id is required")
	}
	now := time.Now().UTC()
	return s.UpdateWithEvent(id, Event{Type: "supervisor.started", Detail: leaseID}, func(state *RuntimeState) error {
		if state.Automation.Supervisor.LeaseID != "" && state.Automation.Supervisor.LeaseID != leaseID {
			if state.Automation.Supervisor.RetryAfter != "" {
				state.Automation.Supervisor.LeaseID = ""
			} else {
			expires, err := time.Parse(time.RFC3339, state.Automation.Supervisor.LeaseExpiresAt)
			if err == nil && now.Before(expires) {
				return fmt.Errorf("supervisor %s is already active", state.Automation.Supervisor.LeaseID)
			}
			}
		}
		state.Automation.Supervisor.LeaseID = leaseID
		state.Automation.Supervisor.StartedAt = now.Format(time.RFC3339)
		state.Automation.Supervisor.LeaseExpiresAt = now.Add(supervisorLeaseTTL).Format(time.RFC3339)
		state.Automation.Supervisor.StoppedAt = ""
		return nil
	})
}

// RenewSupervisorLease extends only the current owner's lease before an
// external operation starts.
func (s *Store) RenewSupervisorLease(id TaskID, leaseID string) (RuntimeState, error) {
	now := time.Now().UTC()
	return s.UpdateWithEvent(id, Event{Type: "supervisor.renewed", Detail: leaseID}, func(state *RuntimeState) error {
		if state.Automation.Supervisor.LeaseID != leaseID {
			return fmt.Errorf("supervisor lease does not own Task")
		}
		state.Automation.Supervisor.LeaseExpiresAt = now.Add(supervisorLeaseTTL).Format(time.RFC3339)
		return nil
	})
}

// StopSupervisor releases only its owning lease and preserves the last result.
func (s *Store) StopSupervisor(id TaskID, leaseID string) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "supervisor.stopped", Detail: leaseID}, func(state *RuntimeState) error {
		if state.Automation.Supervisor.LeaseID != leaseID {
			return fmt.Errorf("supervisor lease does not own Task")
		}
		state.Automation.Supervisor.LeaseID = ""
		state.Automation.Supervisor.LeaseExpiresAt = ""
		state.Automation.Supervisor.StoppedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	})
}

// RecordSupervisorObservation advances the durable cursor after one bounded
// supervisor evaluation. It cannot create an Attempt or resolve a Gate.
func (s *Store) RecordSupervisorObservation(id TaskID, leaseID string, observed uint64, result, detail string) (RuntimeState, error) {
	if strings.TrimSpace(result) == "" {
		return RuntimeState{}, fmt.Errorf("supervisor result is required")
	}
	return s.UpdateWithEvent(id, Event{Type: "supervisor.observed", Detail: result}, func(state *RuntimeState) error {
		if state.Automation.Supervisor.LeaseID != leaseID {
			return fmt.Errorf("supervisor lease does not own Task")
		}
		if observed < state.Automation.Supervisor.ObservedEvent {
			return fmt.Errorf("supervisor observation cannot move backwards")
		}
		// Account for the supervisor.observed event being appended by this
		// transition so the supervisor never wakes itself.
		state.Automation.Supervisor.ObservedEvent = state.LastEventSequence + 1
		state.Automation.Supervisor.Result = result
		state.Automation.Supervisor.Detail = detail
		return nil
	})
}

// SupervisorDue reports whether an active Supervisor has external durable work
// to inspect. Its own observation events are excluded by the stored cursor.
func SupervisorDue(state RuntimeState, now time.Time) bool {
	s := state.Automation.Supervisor
	if s.LeaseID == "" || s.StoppedAt != "" {
		return false
	}
	if s.RetryAfter != "" {
		deadline, err := time.Parse(time.RFC3339, s.RetryAfter)
		if err == nil && now.Before(deadline) {
			return false
		}
	}
	return state.LastEventSequence > s.ObservedEvent
}

// PauseSupervisor schedules a bounded future retry without changing execution
// state. A later supervisor invocation decides whether the deadline is due.
func (s *Store) PauseSupervisor(id TaskID, leaseID, result, detail string, retryAfter time.Time) (RuntimeState, error) {
	current, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	_, err = s.RecordSupervisorObservation(id, leaseID, current.LastEventSequence, result, detail)
	if err != nil {
		return RuntimeState{}, err
	}
	return s.UpdateWithEvent(id, Event{Type: "supervisor.paused", Detail: result}, func(current *RuntimeState) error {
		if current.Automation.Supervisor.LeaseID != leaseID {
			return fmt.Errorf("supervisor lease does not own Task")
		}
		current.Automation.Supervisor.RetryAfter = retryAfter.UTC().Format(time.RFC3339)
		return nil
	})
}
