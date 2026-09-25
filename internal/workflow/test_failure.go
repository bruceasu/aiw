package workflow

import "errors"

type FailureAttribution struct {
	SchemaVersion int `json:"schema_version"`
	RequestID string `json:"request_id"`
	Result ActorReference `json:"result"`
	FailureClass string `json:"failure_class"`
	Reason string `json:"reason"`
	ReadOnly bool `json:"read_only"`
	Evidence []ActorReference `json:"evidence"`
}

// AttributeVerificationFailure consumes the already reserved SW07 diagnosis.
// It preserves the original terminal result, and does not create a repair or
// spend a fresh model request. E04 accounts once under the original request ID.
func (s *Store) AttributeVerificationFailure(id TaskID, revision uint64, attribution FailureAttribution, verify func(RuntimeState, FailureAttribution) error) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.failure.attributed", Detail: attribution.RequestID}, func(state *RuntimeState) error {
		record, err := stageRecord(state, attribution.RequestID)
		if err != nil { return err }
		if state.Protocol.Stop != nil || !record.Consumed || record.Result == nil || *record.Result != attribution.Result || attribution.SchemaVersion != 1 || !attribution.ReadOnly || attribution.Reason == "" || len(attribution.Evidence) == 0 || verify == nil { return errors.New("attribution requires the exact failed result and independent read-only evidence") }
		reserved := false
		for _, key := range state.Protocol.Diagnoses { if key == attribution.RequestID { reserved = true } }
		if !reserved { return errors.New("reserve the original bounded diagnosis before dispatch") }
		var result StageResult
		if err := s.ReadExecutionArtifact(id, *record.Result, &result); err != nil { return err }
		if result.Status != "failed" || result.FailureClass != "unattributed" { return errors.New("only an unattributed test failure may be diagnosed") }
		switch attribution.FailureClass { case "implementation", "test", "infrastructure", "requirements", "unattributed": default: return errors.New("unsupported defect attribution") }
		for _, ref := range attribution.Evidence {
			var evidence map[string]any
			if err := s.ReadExecutionArtifact(id, ref, &evidence); err != nil { return err }
		}
		if err := verify(*state, attribution); err != nil { return err }
		if record.Attribution != nil {
			var prior FailureAttribution
			if err := s.ReadExecutionArtifact(id, *record.Attribution, &prior); err != nil { return err }
			if !equalJSON(prior, attribution) { return errors.New("the bounded diagnosis cannot be replaced") }
			return errProtocolNoChange
		}
		item, err := executionItem(state, record.Request.WorkItemID)
		if err != nil { return err }
		if item.CurrentRequest != record.Request.ID { return errors.New("late diagnosis cannot redirect another request") }
		ref, err := s.persistProtocolArtifactLocked(id, "verification-attribution", attribution)
		if err != nil { return err }
		if err := s.ExecutionServices.Budget(state, record.Request, "attribute:"+attribution.FailureClass); err != nil { return err }
		record.Attribution = &ref
		switch attribution.FailureClass {
		case "implementation": item.Phase = PhaseCoder
		case "test": item.Phase = PhaseTester
		// Environment/requirements/unresolved attribution wait for recovery or
		// an explicit decision. They are not model quality failures.
		}
		return nil
	})
}
