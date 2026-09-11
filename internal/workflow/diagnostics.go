package workflow

import (
	"errors"
	"fmt"
	"os"
)

type Diagnostic struct {
	Code    string
	Message string
	Repair  string
}

// Diagnose is read-only. It supplies actionable recovery guidance without
// silently initializing runtime state or releasing a lease.
func (s *Store) Diagnose(id TaskID) ([]Diagnostic, error) {
	state, err := s.Load(id)
	if errors.Is(err, os.ErrNotExist) {
		return []Diagnostic{{Code: "runtime-missing", Message: "no local Workflow runtime projection exists", Repair: "initialize from durable task metadata through a managed Task command"}}, nil
	}
	if err != nil {
		return []Diagnostic{{Code: "runtime-unreadable", Message: fmt.Sprintf("read workflow runtime: %v", err), Repair: "preserve the state file and repair or restore it before retrying"}}, nil
	}
	diagnostics := s.eventDiagnostics(id, state)
	if state.FocusedTestPlanDigest != "" {
		authorization := state.FocusedTestAuthorizationState(state.FocusedTestPlanDigest)
		repair := "focused-test remains disabled until a human explicitly authorizes this exact Verification Plan digest"
		if authorization == FocusedTestAuthorizationStale {
			repair = "review the changed Verification Plan and obtain a new explicit human authorization for its current digest"
		}
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "focused-test-authorization-" + string(authorization),
			Message: fmt.Sprintf("focused-test profile uses Verification Plan digest %s; authorization is %s", state.FocusedTestPlanDigest, authorization),
			Repair:  repair,
		})
	}
	if state.WriteLease != nil {
		diagnostics = append(diagnostics, Diagnostic{Code: "write-lease-active", Message: fmt.Sprintf("attempt %s owns the workspace write lease", state.WriteLease.AttemptID), Repair: "resume or finish the owning Attempt; do not delete the lease file"})
	}
	if request := state.Automation.PreparedRequest; request != nil {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "agent-request-prepared",
			Message: fmt.Sprintf("work item %s has a prepared agent request for session %s", request.WorkItemID, request.SessionID),
			Repair:  "inspect the Session output, then resume the supervisor; do not create a second Attempt for the same work item",
		})
	}
	if state.Cancellation != nil {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "task-force-closed",
			Message: fmt.Sprintf("Task was force-closed with %s delivery: %s", state.Cancellation.Delivery, state.Cancellation.Reason),
			Repair:  "inspect the preserved Workflow event and evidence history before any follow-up Task",
		})
	}
	for _, item := range state.WorkItems {
		if item.State != WorkItemBlocked || item.NoProgressCount < item.RetryPolicy.MaxAttempts {
			continue
		}
		message := fmt.Sprintf("work item %s exhausted its retry limit (%d/%d)", item.ID, item.NoProgressCount, item.RetryPolicy.MaxAttempts)
		if item.LastOutputReference != "" {
			message += fmt.Sprintf("; last output: %s", item.LastOutputReference)
		}
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "retry-limit-exhausted",
			Message: message,
			Repair:  fmt.Sprintf("review the output, then run: aiw task workflow reopen %s %s <reason>", id, item.ID),
		})
	}
	for _, gate := range state.Gates {
		if gate.State != GateOpen {
			continue
		}
		reason := gate.Reason
		if reason == "" {
			reason = "no reason was recorded; inspect the Workflow event history"
		}
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "gate-open",
			Message: fmt.Sprintf("gate %s (%s) is open%s: %s", gate.ID, gate.Kind, diagnosticWorkItemSuffix(gate.WorkItemID), reason),
			Repair:  fmt.Sprintf("review the gate reason, then run: aiw task workflow gate %s %s resolved|waived", id, gate.ID),
		})
	}
	for _, repair := range state.Automation.ProjectionRepairs {
		if repair.ResolvedAt != "" {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{Code: "projection-repair-pending", Message: fmt.Sprintf("event %d requires %s projection repair", repair.EventSequence, repair.Target), Repair: repair.Recommended})
	}
	return diagnostics, nil
}

func diagnosticWorkItemSuffix(id WorkItemID) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf(" for work item %s", id)
}

func (s *Store) eventDiagnostics(id TaskID, state RuntimeState) []Diagnostic {
	var diagnostics []Diagnostic
	events, err := s.readEvents(id)
	if err != nil {
		return []Diagnostic{{Code: "event-log-unreadable", Message: fmt.Sprintf("read workflow events: %v", err), Repair: "preserve the event log and repair or restore it before retrying"}}
	}
	if state.PendingEvent != nil {
		pending := state.PendingEvent
		if pending.Sequence == state.LastEventSequence+1 && pending.Sequence != 0 && pending.Type != "" {
			diagnostics = append(diagnostics, Diagnostic{Code: "event-pending", Message: fmt.Sprintf("event %d is pending durable confirmation", pending.Sequence), Repair: "run the managed workflow recovery command; it will append only the persisted pending event"})
		} else {
			diagnostics = append(diagnostics, Diagnostic{Code: "event-pending-invalid", Message: fmt.Sprintf("pending event %d cannot follow confirmed sequence %d", pending.Sequence, state.LastEventSequence), Repair: "preserve state and event files; repair the pending event before another workflow transition"})
		}
	}
	last := uint64(0)
	seen := make(map[uint64]bool, len(events))
	for _, event := range events {
		if event.Type == "" || event.SchemaVersion < 1 || event.SchemaVersion > SchemaVersion {
			diagnostics = append(diagnostics, Diagnostic{Code: "event-malformed", Message: "event history contains a record with an unsupported schema or empty type", Repair: "preserve state and event files; repair the event history before another workflow transition"})
			continue
		}
		if event.Sequence == 0 {
			diagnostics = append(diagnostics, Diagnostic{Code: "event-unsequenced", Message: "event history contains a legacy unsequenced record", Repair: "preserve the legacy event record; use only managed transitions for new history"})
			continue
		}
		if seen[event.Sequence] {
			diagnostics = append(diagnostics, Diagnostic{Code: "event-duplicate", Message: fmt.Sprintf("event sequence %d appears more than once", event.Sequence), Repair: "preserve state and event files; repair the event history before another workflow transition"})
			continue
		}
		seen[event.Sequence] = true
		if event.Sequence != last+1 {
			diagnostics = append(diagnostics, Diagnostic{Code: "event-gap", Message: fmt.Sprintf("event sequence %d follows %d", event.Sequence, last), Repair: "preserve state and event files; repair the event history before another workflow transition"})
		}
		last = event.Sequence
	}
	if last != state.LastEventSequence {
		diagnostics = append(diagnostics, Diagnostic{Code: "event-sequence-mismatch", Message: fmt.Sprintf("state confirms event %d but log confirms %d", state.LastEventSequence, last), Repair: "preserve state and event files; repair the event history before another workflow transition"})
	}
	return diagnostics
}
