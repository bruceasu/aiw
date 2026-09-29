package workflow

import (
	"fmt"
)

// ExecutionEvidence is written by the controlled execution/acceptance owner,
// never inferred from an Agent report, checkbox, or old waived test result.
type ExecutionEvidence struct {
	SchemaVersion int `json:"schema_version"`
	WorkItemID WorkItemID `json:"work_item_id"`
	RequestID string `json:"request_id"`
	Report *ActorReference `json:"report,omitempty"`
	Inputs *ValidationInputs `json:"inputs,omitempty"`
	CompileRunID string `json:"compile_run_id"`
	TestRunIDs []string `json:"test_run_ids"`
	TestsRequired bool `json:"tests_required"`
	TestsNotApplicableReason string `json:"tests_not_applicable_reason,omitempty"`
	Controlled bool `json:"controlled"`
	Passed bool `json:"passed"`
	Accepted bool `json:"accepted"`
}

type ExecutionReuse struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// AssessExecutionReuse never creates authorization, refunds consumed grants,
// resets a budget, or asks Coder to rewrite already valid implementation.
func AssessExecutionReuse(evidence ExecutionEvidence, current ValidationInputs, authorized, downstream bool) ExecutionReuse {
	if evidence.SchemaVersion != 1 || evidence.RequestID == "" { return ExecutionReuse{"reconcile", "legacy execution identity is unknown"} }
	if downstream && !evidence.Accepted { return ExecutionReuse{"wait", "downstream requires an accepted result"} }
	applicable, reason := EvidenceApplicability(evidence.Inputs, current, evidence.Controlled, evidence.Passed)
	if !applicable || evidence.CompileRunID == "" || (evidence.TestsRequired && len(evidence.TestRunIDs) == 0) || (!evidence.TestsRequired && evidence.TestsNotApplicableReason == "") {
		if !authorized { return ExecutionReuse{"authorization-required", "current controlled validation is missing; no applicable authorization"} }
		return ExecutionReuse{"validate", "current controlled validation is required: " + reason}
	}
	if evidence.Report == nil || evidence.Report.SHA256 == "" { return ExecutionReuse{"report-only", "reuse applicable validation and request the original generation's one report supplement"} }
	return ExecutionReuse{"reuse", "report and controlled evidence remain applicable"}
}

func (s *Store) PersistValidationInputs(request PreparedAgentRequest, inputs ValidationInputs) (ActorReference, error) {
	if inputs.SchemaVersion != 1 || len(inputs.Scope) == 0 || inputs.Toolchain == "" { return ActorReference{}, fmt.Errorf("validation manifest is incomplete") }
	return s.persistExecutionArtifact(request, "validation-inputs", "validation/"+inputs.Digest()+".json", inputs)
}

// ReadAcceptedExecution consumes an immutable acceptance reference supplied
// by Core. E02/E03 own producing acceptance; E01 cannot manufacture it from
// WorkItemCompleted or a successful Session.
func (s *Store) ReadAcceptedExecution(id TaskID, item WorkItem, reference ActorReference, root string) (ExecutionEvidence, error) {
	var evidence ExecutionEvidence
	if item.State != WorkItemCompleted || reference.Kind != "accepted-execution" { return evidence, fmt.Errorf("dependency is not accepted") }
	if err := s.ReadExecutionArtifact(id, reference, &evidence); err != nil { return evidence, err }
	if evidence.WorkItemID != item.ID || evidence.Inputs == nil { return evidence, fmt.Errorf("dependency lacks input provenance") }
	state, err := s.Load(id)
	if err != nil { return evidence, err }
	var bindings []ActorReference
	if state.Automation.PreparedRequest == nil || state.Automation.PreparedRequest.Compile == nil { return evidence, fmt.Errorf("current validation plan is unavailable") }
	bindings, err = CompileInputBindings(state.Automation.PreparedRequest.Compile.Plan)
	if err != nil { return evidence, err }
	toolchain, err := ValidationToolchainIdentity()
	if err != nil { return evidence, err }
	current, err := CaptureValidationInputs(root, evidence.Inputs.Scope, bindings, toolchain)
	if err != nil { return evidence, err }
	decision := AssessExecutionReuse(evidence, current, false, true)
	if decision.Action != "reuse" { return evidence, fmt.Errorf("dependency %s: %s", decision.Action, decision.Reason) }
	var report ExecutionReport
	if err := s.ReadExecutionArtifact(id, *evidence.Report, &report); err != nil { return evidence, err }
	if report.WorkItemID != item.ID || report.RequestID != evidence.RequestID { return evidence, fmt.Errorf("dependency report identity mismatch") }
	return evidence, nil
}
