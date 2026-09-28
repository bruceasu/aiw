package workflow

import (
	"encoding/json"
	"errors"
)

// ValidateStageReport advances the deterministic report check without another
// model call. A failed check leaves PhaseReport intact for its single read-only
// supplement; a second invalid report requires manual handling.
func (s *Store) ValidateStageReport(id TaskID, revision uint64, workItemID WorkItemID, reference ActorReference) (RuntimeState, error) {
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.report.validated", WorkItemID: workItemID}, func(state *RuntimeState) error {
		item, err := executionItem(state, workItemID)
		if err != nil { return err }
		if item.Phase != PhaseReport || state.Protocol.Stop != nil { return errors.New("report validation is not the current active phase") }
		if err := requireNoStageInFlight(*state); err != nil { return err }
		current, err := stageRecord(state, item.CurrentRequest)
		if err != nil || !current.Consumed { return errors.New("report requires a terminal generation") }
		origin := current.Request
		if origin.Phase == PhaseReport {
			found := false
			for i := len(state.Protocol.Requests)-1; i >= 0; i-- {
				prior := state.Protocol.Requests[i].Request
				if prior.WorkItemID == workItemID && prior.Phase == PhaseCoder { origin, found = prior, true; break }
			}
			if !found { return errors.New("report supplement has no original generation") }
		}
		var report ExecutionReport
		if reference.Kind != "execution-report" { return errors.New("report reference has the wrong kind") }
		if err := s.ReadExecutionArtifact(id, reference, &report); err != nil { return err }
		if report.RequestID != current.Request.ID || report.TaskID != id || report.WorkItemID != workItemID || report.AttemptID != item.AttemptID { return errors.New("report identity differs from current generation") }
		request := PreparedAgentRequest{TaskID: id, WorkItemID: workItemID, AttemptID: origin.AttemptID, SessionID: origin.SessionID, ExpectedSessionTurn: origin.Turn, Workspace: origin.Workspace, InputReference: &origin.Input}
		validated, err := ValidateImplementationReport(report.Output, request, origin.Workspace)
		if err != nil { return err }
		var input ExecutionInput
		if err := s.ReadExecutionArtifact(id, origin.Input, &input); err != nil { return err }
		if err := ValidateReportChanges(validated, input.WorkspaceInputs, origin.Workspace); err != nil { return err }
		item.Phase, item.ValidatedReport = PhaseCompile, &reference
		return nil
	})
}

type InvalidCoderReportRetry struct {
	RequestID string `json:"request_id"`
	Result ActorReference `json:"result"`
	Report ActorReference `json:"report"`
	Input ActorReference `json:"input"`
	WorkspaceDigest string `json:"workspace_digest"`
	Reason string `json:"reason"`
}

// RetryInvalidCoderReport records one explicit no-model recovery decision.
// It preserves the original terminal result and usage, and permits a new
// Coder request only if the failed turn left the workspace unchanged.
func (s *Store) RetryInvalidCoderReport(id TaskID, revision uint64, workItemID WorkItemID) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.report.retry-authorized", WorkItemID: workItemID}, func(state *RuntimeState) error {
		item, err := executionItem(state, workItemID)
		if err != nil { return err }
		if item.Phase != PhaseReport || item.ReportRetry != nil || state.Protocol.Stop != nil { return errors.New("report retry requires the first active, unstopped Coder report failure") }
		if err := requireNoStageInFlight(*state); err != nil { return err }
		for _, gate := range state.Gates { if gate.State == GateOpen && (gate.WorkItemID == "" || gate.WorkItemID == workItemID) { return errors.New("open Gate blocks report retry") } }
		if usageBudgetAuthorizationOpen(*state) { return errors.New("Task usage budget authorization is pending") }
		record, err := stageRecord(state, item.CurrentRequest)
		if err != nil { return err }
		if record.Request.Phase != PhaseCoder || record.Request.WorkItemID != workItemID || record.Request.AttemptID != item.AttemptID || !record.Consumed || record.Dispatch != "terminal" || record.Result == nil { return errors.New("report retry requires the consumed original Coder result") }
		var result StageResult
		if err := s.ReadExecutionArtifact(id, *record.Result, &result); err != nil { return err }
		if result.RequestID != record.Request.ID || result.Status != "passed" || len(result.Evidence) < 2 || result.Evidence[1].Kind != "execution-report" { return errors.New("report retry requires an exact passed Coder result and preserved report") }
		var report ExecutionReport
		if err := s.ReadExecutionArtifact(id, result.Evidence[1], &report); err != nil { return err }
		if report.RequestID != record.Request.ID || report.TaskID != id || report.WorkItemID != workItemID || report.AttemptID != item.AttemptID || report.SessionID != record.Request.SessionID || report.Turn != record.Request.Turn { return errors.New("preserved Coder report does not match the original request") }
		var envelope struct { Report json.RawMessage `json:"report"` }
		if err := json.Unmarshal([]byte(report.Output), &envelope); err == nil && len(envelope.Report) > 0 && string(envelope.Report) != "null" { return errors.New("report is structured; use normal validation or a separate review decision") }
		var input ExecutionInput
		if err := s.ReadExecutionArtifact(id, record.Request.Input, &input); err != nil { return err }
		if input.WorkspaceInputs == nil || input.WorkspaceInputs.Digest() != record.Request.InputDigest { return errors.New("original Coder workspace baseline is missing") }
		before := input.WorkspaceInputs
		after, err := CaptureValidationInputs(record.Request.Workspace, before.Scope, before.Bindings, before.Toolchain)
		if err != nil { return err }
		if before.Digest() != after.Digest() { return errors.New("Coder workspace changed; preserve the report and inspect files before authorizing another call") }
		decision := InvalidCoderReportRetry{RequestID: record.Request.ID, Result: *record.Result, Report: result.Evidence[1], Input: record.Request.Input, WorkspaceDigest: after.Digest(), Reason: "Coder returned no structured implementation report and left the workspace unchanged"}
		ref, err := s.persistProtocolArtifactLocked(id, "report-retry-decision", decision)
		if err != nil { return err }
		item.ReportRetry, item.Phase, item.ValidatedReport = &ref, PhaseCoder, nil
		return nil
	})
}
