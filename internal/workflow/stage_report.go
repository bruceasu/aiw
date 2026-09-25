package workflow

import "errors"

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
