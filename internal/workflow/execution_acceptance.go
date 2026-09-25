package workflow

import (
	"encoding/json"
	"errors"
	"time"
)

type AcceptanceCandidate struct {
	RequestID string `json:"request_id"`
	CurrentRequestID string `json:"current_request_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	Report ActorReference `json:"report"`
	Inputs ValidationInputs `json:"inputs"`
	CompileRunID string `json:"compile_run_id"`
	TestRunIDs []string `json:"test_run_ids"`
	TestsRequired bool `json:"tests_required"`
	NotApplicableRule ActorReference `json:"not_applicable_rule"`
	NotApplicableReason string `json:"not_applicable_reason"`
	Plan ActorReference `json:"plan"`
	Grant ActorReference `json:"grant"`
	Policy ActorReference `json:"policy"`
}

// RevalidateExecution returns to validation without asking Coder to rewrite
// applicable implementation. Completed facts remain immutable history.
func (s *Store) RevalidateExecution(id TaskID, revision uint64, itemID WorkItemID, phase ExecutionPhase, reason ActorReference) (RuntimeState, error) {
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.validation.invalidated", WorkItemID: itemID}, func(state *RuntimeState) error {
		item, err := executionItem(state, itemID)
		if err != nil { return err }
		if state.Protocol.Stop != nil || item.Phase == PhaseAccepted { return errors.New("stopped or accepted execution cannot be rewritten") }
		if phase != PhaseCompile && phase != PhaseTest { return errors.New("revalidation must target compile or controlled tests") }
		if item.Phase != PhaseAccept && item.Phase != PhaseTest && item.Phase != PhaseTester { return errors.New("execution is not awaiting validation") }
		if err := requireNoStageInFlight(*state); err != nil { return err }
		var proof json.RawMessage
		if err := s.ReadExecutionArtifact(id, reason, &proof); err != nil { return err }
		item.Phase = phase
		return nil
	})
}

// AcceptExecution is the sole schema-10 Work Item completion path. Its callback
// verifies the controlled run facts and current authorization; Core checks the
// current bytes again and publishes evidence before it releases dependencies.
func (s *Store) AcceptExecution(id TaskID, revision uint64, candidate AcceptanceCandidate) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	snapshot, snapshotErr := s.prepareVerifierSnapshot(id, revision, candidate)
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.execution.accepted", WorkItemID: candidate.WorkItemID, AttemptID: candidate.AttemptID}, func(state *RuntimeState) error {
		item, err := executionItem(state, candidate.WorkItemID)
		if err != nil { return err }
		if state.Protocol.Stop != nil { return errors.New("execution is stopped") }
		if err := requireNoStageInFlight(*state); err != nil { return err }
		if (item.Phase != PhaseAccept && ((item.Phase != PhaseTest && item.Phase != PhaseTester) || candidate.TestsRequired)) || item.AttemptID != candidate.AttemptID || item.CurrentRequest != candidate.CurrentRequestID { return errors.New("acceptance candidate does not match the current stage") }
		if !validProtocolReference(candidate.Report) || candidate.CompileRunID == "" || !validProtocolReference(candidate.Plan) || !validProtocolReference(candidate.Policy) { return errors.New("acceptance lacks report, compile or plan provenance") }
		if candidate.TestsRequired {
			if len(candidate.TestRunIDs) == 0 || !validProtocolReference(candidate.Grant) { return errors.New("required tests need passed controlled runs and a grant; waived is insufficient") }
		} else if candidate.NotApplicableReason == "" || !validProtocolReference(candidate.NotApplicableRule) { return errors.New("test N/A requires an explicit applicability rule") }
		if err := s.ExecutionServices.ValidateAcceptance(*state, candidate); err != nil { return err }
		toolchain, err := ValidationToolchainIdentity()
		if err != nil { return err }
		current, err := CaptureValidationInputs(executionWorkspace(*state), candidate.Inputs.Scope, candidate.Inputs.Bindings, toolchain)
		if err != nil { return err }
		if applicable, reason := EvidenceApplicability(&candidate.Inputs, current, true, true); !applicable { return errors.New("acceptance inputs changed: "+reason) }
		// Keep the full plan/grant/policy/N/A provenance separately from the E01
		// compatibility evidence consumed by downstream context preparation.
		candidateRef, err := s.persistProtocolArtifactLocked(id, "acceptance-candidate", candidate)
		if err != nil { return err }
		evidence := ExecutionEvidence{SchemaVersion: 1, WorkItemID: candidate.WorkItemID, RequestID: candidate.RequestID, Report: &candidate.Report, Inputs: &candidate.Inputs, CompileRunID: candidate.CompileRunID, TestRunIDs: candidate.TestRunIDs, TestsRequired: candidate.TestsRequired, TestsNotApplicableReason: candidate.NotApplicableReason, Controlled: true, Passed: true, Accepted: true}
		ref, err := s.persistProtocolArtifactLocked(id, "accepted-execution", evidence)
		if err != nil { return err }
		item.Phase, item.Accepted = PhaseAccepted, &ref
		item.AcceptanceCandidate = &candidateRef
		if snapshotErr == nil {
			item.VerifierSnapshot, item.VerifierGap = &snapshot, ""
		} else {
			item.VerifierGap = "unavailable: "+snapshotErr.Error()
		}
		for i := range state.WorkItems {
			if state.WorkItems[i].ID == candidate.WorkItemID { state.WorkItems[i].State, state.WorkItems[i].AcceptedReference = WorkItemCompleted, &ref }
		}
		for i := range state.Attempts {
			if state.Attempts[i].ID == candidate.AttemptID { state.Attempts[i].State, state.Attempts[i].EndedAt = AttemptCompleted, time.Now().UTC().Format(time.RFC3339) }
		}
		return nil
	})
}
