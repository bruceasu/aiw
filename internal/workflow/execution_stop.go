package workflow

import "errors"

// RequestExecutionStop records the explicit CLI decision even if no foreground
// supervisor exists. It does not claim that an in-flight process has exited.
func (s *Store) RequestExecutionStop(id TaskID, revision uint64, reason, source string) (RuntimeState, error) {
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.stop.requested", Detail: reason}, func(state *RuntimeState) error {
		if state.Protocol == nil || reason == "" || source == "" { return errors.New("explicit Stop requires a reason and source") }
		ref, err := s.persistProtocolArtifactLocked(id, "stop-decision", struct { Reason string `json:"reason"`; Source string `json:"source"` }{reason, source})
		if err != nil { return err }
		state.Protocol.Stop = &ExecutionStop{Reason: reason, Reference: ref}
		return nil
	})
}
