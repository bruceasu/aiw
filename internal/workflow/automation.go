package workflow

import (
	"fmt"
	"strings"
	"time"
)

// RecordAutomation persists a bounded orchestrator result through the normal
// ordered event path. It intentionally has no adapter or external side effect.
func (s *Store) RecordAutomation(id TaskID, fingerprint string, cursor AutomationCursor, request *PreparedAgentRequest) (RuntimeState, error) {
	if strings.TrimSpace(cursor.Result) == "" {
		return RuntimeState{}, fmt.Errorf("automation cursor result is required")
	}
	if cursor.RecordedAt == "" {
		cursor.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if request != nil {
		if request.TaskID != id || request.WorkItemID == "" || request.AttemptID == "" || strings.TrimSpace(request.Workspace) == "" {
			return RuntimeState{}, fmt.Errorf("prepared agent request must bind the Task, Work Item, Attempt, and workspace")
		}
		if request.PreparedAt == "" {
			request.PreparedAt = cursor.RecordedAt
		}
		if request.SkillManifest != nil {
			if err := request.SkillManifest.Validate(); err != nil {
				return RuntimeState{}, fmt.Errorf("Skill manifest: %w", err)
			}
		}
		if selection := request.AISelection; selection != nil && (strings.TrimSpace(selection.Profile) == "" || strings.TrimSpace(selection.Provider) == "" || strings.TrimSpace(selection.Model) == "" || strings.TrimSpace(selection.Digest) == "") {
			return RuntimeState{}, fmt.Errorf("AI selection requires Profile, provider, model, and digest")
		}
	}
	return s.UpdateWithEvent(id, Event{Type: "automation.recorded", Detail: cursor.Result}, func(state *RuntimeState) error {
		state.Automation.PlanFingerprint = fingerprint
		state.Automation.Cursor = cursor
		state.Automation.PreparedRequest = request
		return nil
	})
}

// DispatchPreparedAgentRequest marks a prepared request as having crossed the
// managed-adapter boundary. The marker is durable so recovery never mistakes a
// merely prepared request for a completed Session result.
func (s *Store) DispatchPreparedAgentRequest(id TaskID, attemptID AttemptID) (RuntimeState, error) {
	if attemptID == "" {
		return RuntimeState{}, fmt.Errorf("prepared agent request attempt is required")
	}
	return s.UpdateWithEvent(id, Event{Type: "agent-request.dispatched", AttemptID: attemptID}, func(state *RuntimeState) error {
		request := state.Automation.PreparedRequest
		if request == nil || request.AttemptID != attemptID {
			return fmt.Errorf("prepared agent request does not match Attempt %s", attemptID)
		}
		if request.DispatchedAt == "" {
			request.DispatchedAt = time.Now().UTC().Format(time.RFC3339)
		}
		return nil
	})
}

// EnqueueProjectionRepair records a projection-only retry instruction after a
// committed event. Repeated failures for the same event and target are
// intentionally idempotent.
func (s *Store) EnqueueProjectionRepair(id TaskID, repair ProjectionRepair) (RuntimeState, error) {
	if repair.EventSequence == 0 || strings.TrimSpace(repair.Target) == "" || strings.TrimSpace(repair.Recommended) == "" {
		return RuntimeState{}, fmt.Errorf("projection repair requires event sequence, target, and recommended command")
	}
	return s.UpdateWithEvent(id, Event{Type: "projection-repair.enqueued", Detail: repair.Target}, func(state *RuntimeState) error {
		for _, existing := range state.Automation.ProjectionRepairs {
			if existing.EventSequence == repair.EventSequence && existing.Target == repair.Target {
				return nil
			}
		}
		state.Automation.ProjectionRepairs = append(state.Automation.ProjectionRepairs, repair)
		return nil
	})
}

// ResolveProjectionRepair marks only an existing projection repair as resolved.
// The caller owns retrying the projection itself before invoking this method.
func (s *Store) ResolveProjectionRepair(id TaskID, eventSequence uint64, target string) (RuntimeState, error) {
	if eventSequence == 0 || strings.TrimSpace(target) == "" {
		return RuntimeState{}, fmt.Errorf("projection repair event sequence and target are required")
	}
	return s.UpdateWithEvent(id, Event{Type: "projection-repair.resolved", Detail: target}, func(state *RuntimeState) error {
		for index := range state.Automation.ProjectionRepairs {
			repair := &state.Automation.ProjectionRepairs[index]
			if repair.EventSequence == eventSequence && repair.Target == target {
				if repair.ResolvedAt == "" {
					repair.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
				}
				return nil
			}
		}
		return fmt.Errorf("unknown projection repair for event %d target %s", eventSequence, target)
	})
}
