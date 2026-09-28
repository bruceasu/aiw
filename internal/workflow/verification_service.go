package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// VerificationHost is the trusted execution boundary, not an Agent response.
// Check must enforce stable target access, process/network/write restrictions,
// indirect toolchain identity and resource limits. VerifyReceipt independently
// reads the host's exact request/terminal journal; an Agent cannot attest itself.
// A provider which cannot guarantee any of these properties must return
// host-unavailable. Registration does not activate schema 10.
type VerificationHost interface {
	Identity() string
	Check(RuntimeState, StageRequest, *TestManifest) error
	VerifyReceipt(StageRequest, VerificationReceipt) error
}

// Observation evidence and recovery conditions are distinct from terminal
// receipts. They cannot be supplied by a process which merely timed out.
type VerificationObservationHost interface {
	VerifyObservation(RuntimeState, StageRequest, StageResult) error
}

type AssertionReviewAuthority interface {
	VerifyAssertions(RuntimeState, StageRequest, VerificationReceipt) error
}

type TestCheckReceipt struct {
	CheckID string `json:"check_id"`
	ExitCode int `json:"exit_code"`
	TimedOut bool `json:"timed_out"`
	Output ActorReference `json:"output"`
	OutputBytes int `json:"output_bytes"`
	OutputTruncated bool `json:"output_truncated"`
	StartedAt string `json:"started_at"`
	EndedAt string `json:"ended_at"`
}

type VerificationReceipt struct {
	SchemaVersion int `json:"schema_version"`
	Request StageRequest `json:"request"`
	Result StageResult `json:"result"`
	Inputs ValidationInputs `json:"inputs"`
	After ValidationInputs `json:"after"`
	Checks []TestCheckReceipt `json:"checks"`
	Inventory *TestCaseInventory `json:"inventory,omitempty"`
	TestPlan *ActorReference `json:"test_plan,omitempty"`
	// A Tester repair must explain assertion changes against the original
	// requirements. The trusted host verifies the independent review evidence.
	AssertionReview *ActorReference `json:"assertion_review,omitempty"`
	// Usage contains only normalized Provider-reported fields; raw response
	// fragments remain in the Session's bounded usage sidecar.
	Usage json.RawMessage `json:"provider_usage,omitempty"`
}

type VerificationService struct {
	Store *Store
	Host VerificationHost
}

// Connect supplies only E03 responsibilities. E04 budget/migration and the
// platform activation verifier remain mandatory and are never replaced here.
func (v *VerificationService) Connect(services *ExecutionServices) error {
	if v == nil || v.Store == nil || v.Host == nil || v.Host.Identity() == "" || services == nil { return errors.New("host-unavailable: controlled verification service is incomplete") }
	services.Authorize, services.ValidateResult, services.ValidateAcceptance = v.Authorize, v.ValidateResult, v.ValidateAcceptance
	return nil
}

func (v *VerificationService) Authorize(state RuntimeState, request StageRequest) error {
	if v == nil || v.Store == nil || v.Host == nil || state.Protocol == nil { return errors.New("host-unavailable: controlled verification service is absent") }
	if request.TaskID != state.Task.ID || request.Workspace != executionWorkspace(state) { return errors.New("stale: stage workspace binding") }
	if request.Phase == PhaseTest {
		manifest, err := v.Store.CheckTestManifest(state, request)
		if err != nil { return err }
		if err := v.requireCurrentCompile(state, request, manifest.Inputs); err != nil { return err }
		item, err := executionItem(&state, request.WorkItemID)
		if err != nil { return err }
		authored, err := stageRecord(&state, item.TestAuthoringRequest)
		if err != nil { return err }
		receipt, err := v.receipt(state, *authored)
		if err != nil { return err }
		if authored.Request.Phase != PhaseTester || receipt.Result.Status != "passed" || receipt.TestPlan == nil || *receipt.TestPlan != request.Plan || receipt.Inventory == nil { return errors.New("Runner plan is not the current independent Tester output") }
		for _, test := range receipt.Inventory.Cases {
			found := false
			for _, file := range manifest.Tests { if file == test.TestFile { found = true } }
			if !found { return errors.New("Runner discovery omits an authored unit test") }
		}
		return v.Host.Check(state, request, &manifest)
	}
	if request.Phase == PhaseTester {
		var input ExecutionInput
		if err := v.Store.ReadExecutionArtifact(state.Task.ID, request.Input, &input); err != nil { return err }
		if err := validateIndependentTesterInput(input, request); err != nil { return err }
		var policy TestExecutionPolicy
		if err := v.Store.readStrictArtifact(state.Task.ID, request.Policy, &policy); err != nil { return err }
		if policy.TaskID != state.Task.ID || policy.WorkspaceDigest != WorkspaceBindingDigest(state) || !equalJSON(policy.TestPaths, request.AllowedPaths) || !equalJSON(policy.InputScope, input.WorkspaceInputs.Scope) { return errors.New("Tester write/input scope does not match the current policy") }
		toolchain, err := ValidationToolchainIdentity()
		if err != nil { return err }
		current, err := CaptureValidationInputs(request.Workspace, input.WorkspaceInputs.Scope, input.WorkspaceInputs.Bindings, toolchain)
		if err != nil { return err }
		if !equalJSON(current, *input.WorkspaceInputs) { return errors.New("Tester context became stale before dispatch") }
		if err := v.requireCurrentCompile(state, request, *input.WorkspaceInputs); err != nil { return err }
		for _, record := range state.Protocol.Requests {
			if record.Request.Phase == PhaseCoder && record.Request.SessionID == request.SessionID { return errors.New("Tester must use a separate Session from Coder") }
		}
	}
	for _, ref := range []ActorReference{request.Plan, request.Policy} {
		var artifact map[string]any
		if err := v.Store.ReadExecutionArtifact(state.Task.ID, ref, &artifact); err != nil { return err }
	}
	return v.Host.Check(state, request, nil)
}

func validateIndependentTesterInput(input ExecutionInput, request StageRequest) error {
	if input.Actor != ActorTester || input.RequestID != request.ID || input.WorkspaceInputs == nil || input.WorkspaceInputs.Digest() != request.InputDigest || !equalJSON(input.AllowedPaths, request.AllowedPaths) || len(input.AllowedPaths) == 0 { return errors.New("Tester requires a frozen independent context and explicit test write paths") }
	kinds := map[string]bool{}
	for _, source := range input.Sources {
		if source.Kind == "optional-knowledge" && !source.Required && source.Status == "unavailable" && source.Reason != "" { continue }
		if source.Status != "loaded" || source.Content == "" || source.SHA256 != contentDigest([]byte(source.Content)) { return errors.New("Tester input body is missing or changed") }
		kinds[source.Kind] = true
	}
	for _, kind := range []string{"requirements", "acceptance", "implementation-report", "interfaces", "source", "test-inventory", "fixture-inventory", "test-policy", "allowed-paths"} {
		if !kinds[kind] { return fmt.Errorf("Tester context is missing %s body", kind) }
	}
	return nil
}

func (v *VerificationService) receipt(state RuntimeState, record StageRecord) (VerificationReceipt, error) {
	var result StageResult
	var receipt VerificationReceipt
	if !record.Consumed || record.Result == nil { return receipt, errors.New("controlled terminal result is absent") }
	if err := v.Store.ReadExecutionArtifact(state.Task.ID, *record.Result, &result); err != nil { return receipt, err }
	if len(result.Evidence) == 0 || result.Evidence[0].Kind != "verification-receipt" { return receipt, errors.New("host receipt is absent") }
	if err := v.Store.ReadExecutionArtifact(state.Task.ID, result.Evidence[0], &receipt); err != nil { return receipt, err }
	if err := v.validateReceipt(state, record.Request, result, receipt); err != nil { return receipt, err }
	return receipt, nil
}

func (v *VerificationService) requireCurrentCompile(state RuntimeState, request StageRequest, inputs ValidationInputs) error {
	for i := len(state.Protocol.Requests)-1; i >= 0; i-- {
		record := state.Protocol.Requests[i]
		if record.Request.WorkItemID != request.WorkItemID || record.Request.AttemptID != request.AttemptID || record.Request.Phase != PhaseCompile { continue }
		receipt, err := v.receipt(state, record)
		if err != nil { return err }
		if receipt.Result.Status != "passed" || !sameValidationContent(receipt.Inputs, inputs) { return errors.New("current implementation/tests require a new controlled compile") }
		return nil
	}
	return errors.New("independent Tester/Runner requires a current compile result")
}

// Compile and test plans have distinct binding references. Compare content,
// scope and toolchain here; each plan is independently checked at its own run.
func sameValidationContent(a, b ValidationInputs) bool {
	if a.SchemaVersion != 1 || b.SchemaVersion != 1 || len(a.Scope) == 0 || a.Toolchain == "" || strings.HasPrefix(a.Toolchain, "incomplete:") { return false }
	return equalJSON(a.Scope, b.Scope) && equalJSON(a.Files, b.Files) && a.Toolchain == b.Toolchain
}

func (v *VerificationService) ValidateResult(state RuntimeState, request StageRequest, result StageResult) error {
	if result.Status == "unknown" || result.Status == "dispatched" || result.Status == "not-dispatched" || result.Status == "recovery-ready" {
		observer, ok := v.Host.(VerificationObservationHost)
		if !ok || len(result.Evidence) != 1 { return errors.New("host-unavailable: exact observation/recovery authority is missing") }
		var proof map[string]any
		if err := v.Store.ReadExecutionArtifact(state.Task.ID, result.Evidence[0], &proof); err != nil { return err }
		return observer.VerifyObservation(state, request, result)
	}
	if len(result.Evidence) == 0 || result.Evidence[0].Kind != "verification-receipt" { return errors.New("Agent status is not a controlled execution receipt") }
	var receipt VerificationReceipt
	if err := v.Store.ReadExecutionArtifact(state.Task.ID, result.Evidence[0], &receipt); err != nil { return err }
	return v.validateReceipt(state, request, result, receipt)
}

func (v *VerificationService) validateReceipt(state RuntimeState, request StageRequest, result StageResult, receipt VerificationReceipt) error {
	if v.Host == nil || receipt.SchemaVersion != 1 || !equalJSON(receipt.Request, request) || result.Executor != v.Host.Identity() || !result.Terminal || result.InputDigest != request.InputDigest || receipt.Inputs.Digest() != request.InputDigest { return errors.New("host receipt identity or input mismatch") }
	copyResult := result
	copyResult.Evidence = result.Evidence[1:]
	if !equalJSON(copyResult, receipt.Result) { return errors.New("stage result differs from the host receipt") }
	if err := v.Host.VerifyReceipt(request, receipt); err != nil { return err }
	if result.Status == "passed" {
		if result.FailureClass != "" { return errors.New("passed result contains a failure classification") }
	} else {
		switch result.FailureClass { case "implementation", "test", "infrastructure", "requirements", "unattributed": default: return errors.New("failure requires supported, evidenced attribution") }
		if len(receipt.Result.Evidence) == 0 { return errors.New("failure attribution requires preserved diagnostic evidence") }
	}
	for _, ref := range receipt.Result.Evidence {
		var evidence map[string]any
		if err := v.Store.ReadExecutionArtifact(state.Task.ID, ref, &evidence); err != nil { return err }
	}
	if request.Phase == PhaseTest {
		var manifest TestManifest
		if err := v.Store.ReadExecutionArtifact(state.Task.ID, request.Input, &manifest); err != nil { return err }
		if !equalJSON(manifest.Inputs, receipt.Inputs) || !sameValidationContent(receipt.Inputs, receipt.After) { return errors.New("test execution changed its frozen inputs") }
		if (result.Status == "passed" && len(receipt.Checks) != len(manifest.Invocations)) || len(receipt.Checks) > len(manifest.Invocations) { return errors.New("test result coverage is incomplete") }
		seen := map[string]bool{}
		for _, check := range receipt.Checks {
			var expected *TestInvocation
			for i := range manifest.Invocations { if manifest.Invocations[i].Check.CheckID == check.CheckID { expected = &manifest.Invocations[i]; break } }
			if expected == nil || seen[check.CheckID] || check.StartedAt == "" || check.EndedAt == "" || check.OutputBytes < 0 || check.OutputBytes > manifest.Limits.MaxOutputBytes { return errors.New("invalid controlled test check receipt") }
			seen[check.CheckID] = true
			var output struct { Data []byte `json:"data"` }
			if err := v.Store.ReadExecutionArtifact(state.Task.ID, check.Output, &output); err != nil { return err }
			if len(output.Data) != check.OutputBytes { return errors.New("test output byte count mismatch") }
			if result.Status == "passed" && (check.TimedOut || check.ExitCode != expected.Check.ExpectedExitCode) { return errors.New("failed/timed-out check cannot establish passed") }
		}
	}
	if request.Phase == PhaseTester {
		contract := ImplementationContract{Status: ImplementationContractReady, AllowedTestPaths: request.AllowedPaths}
		if err := ValidateTesterChangedPaths(contract, changedValidationPaths(receipt.Inputs, receipt.After)); err != nil { return err }
	}
	if request.Phase == PhaseTester && result.Status == "passed" {
		if receipt.Inventory == nil || receipt.TestPlan == nil || receipt.Inventory.WorkItemID != request.WorkItemID { return errors.New("Tester must provide unit cases and a concrete execution plan") }
		if err := receipt.Inventory.Validate(); err != nil { return err }
		var input ExecutionInput
		if err := v.Store.ReadExecutionArtifact(state.Task.ID, request.Input, &input); err != nil { return err }
		if err := validateIndependentTesterInput(input, request); err != nil { return err }
		contract := ImplementationContract{Status: ImplementationContractReady, AllowedTestPaths: request.AllowedPaths}
		changed := changedValidationPaths(receipt.Inputs, receipt.After)
		if err := ValidateTesterChangedPaths(contract, changed); err != nil { return err }
		for _, test := range receipt.Inventory.Cases {
			if err := ValidateTesterChangedPaths(contract, []string{test.TestFile}); err != nil { return err }
			found := false
			for _, file := range receipt.After.Files { if file.Path == test.TestFile { found = true } }
			if !found { return errors.New("authored test file is missing from the output manifest") }
		}
		var plan map[string]any
		if err := v.Store.ReadExecutionArtifact(state.Task.ID, *receipt.TestPlan, &plan); err != nil { return err }
		for _, prior := range state.Protocol.Requests {
			if prior.Request.ID == request.ID || prior.Request.WorkItemID != request.WorkItemID || prior.Request.Phase != PhaseTester || !prior.Consumed { continue }
			if receipt.AssertionReview == nil { return errors.New("Tester repair needs independent requirement-based assertion review") }
			var review map[string]any
			if err := v.Store.ReadExecutionArtifact(state.Task.ID, *receipt.AssertionReview, &review); err != nil { return err }
			authority, ok := v.Host.(AssertionReviewAuthority)
			if !ok { return errors.New("host-unavailable: independent assertion review is missing") }
			if err := authority.VerifyAssertions(state, request, receipt); err != nil { return err }
			break
		}
	}
	if request.Phase == PhaseCoder {
		// Coder must not erase or weaken tests to make a failure disappear.
		var policy TestExecutionPolicy
		if err := v.Store.readStrictArtifact(state.Task.ID, request.Policy, &policy); err != nil { return err }
		for _, path := range changedValidationPaths(receipt.Inputs, receipt.After) {
			if matchesTestPath(policy.TestPaths, path) || matchesTestPath(policy.FixturePaths, path) { return errors.New("Coder changed a protected test/fixture; return the repair to Tester") }
		}
	}
	return nil
}

func changedValidationPaths(before, after ValidationInputs) []string {
	old := map[string]string{}
	for _, file := range before.Files { old[file.Path] = file.SHA256 }
	changed := []string{}
	for _, file := range after.Files { if old[file.Path] != file.SHA256 { changed = append(changed, file.Path) }; delete(old, file.Path) }
	for path := range old { changed = append(changed, path) }
	return changed
}

func (v *VerificationService) ValidateAcceptance(state RuntimeState, candidate AcceptanceCandidate) error {
	if state.Protocol == nil || candidate.RequestID == "" { return errors.New("acceptance has no managed generation") }
	item, err := executionItem(&state, candidate.WorkItemID)
	if err != nil { return err }
	if item.ValidatedReport == nil || *item.ValidatedReport != candidate.Report { return errors.New("acceptance report is not the current validated generation report") }
	var report ExecutionReport
	if err := v.Store.ReadExecutionArtifact(state.Task.ID, candidate.Report, &report); err != nil { return err }
	if report.TaskID != state.Task.ID || report.WorkItemID != candidate.WorkItemID || report.AttemptID != candidate.AttemptID || (report.RequestID != candidate.RequestID && report.OriginalRequestID != candidate.RequestID) || report.OutputSHA256 != contentDigest([]byte(report.Output)) { return errors.New("acceptance report binding mismatch") }
	origin, err := stageRecord(&state, candidate.RequestID)
	if err != nil { return err }
	if origin.Request.Phase != PhaseCoder || !origin.Consumed { return errors.New("acceptance requires the terminal Coder generation") }
	prepared := PreparedAgentRequest{TaskID: state.Task.ID, WorkItemID: candidate.WorkItemID, AttemptID: candidate.AttemptID, SessionID: origin.Request.SessionID, ExpectedSessionTurn: origin.Request.Turn, Workspace: origin.Request.Workspace, InputReference: &origin.Request.Input}
	if _, err := ValidateImplementationReport(report.Output, prepared, prepared.Workspace); err != nil { return err }
	compile, err := stageRecord(&state, candidate.CompileRunID)
	if err != nil || compile.Request.Phase != PhaseCompile || compile.Request.WorkItemID != candidate.WorkItemID || compile.Request.AttemptID != candidate.AttemptID { return errors.New("acceptance compile binding mismatch") }
	compiled, err := v.receipt(state, *compile)
	if err != nil { return err }
	if compiled.Result.Status != "passed" || !sameValidationContent(compiled.Inputs, candidate.Inputs) { return errors.New("acceptance compile inputs are stale") }
	if candidate.TestsRequired {
		if len(candidate.TestRunIDs) != 1 { return errors.New("acceptance needs one complete controlled test manifest") }
		record, err := stageRecord(&state, candidate.TestRunIDs[0])
		if err != nil { return err }
		if record.Request.Phase != PhaseTest || record.Request.WorkItemID != candidate.WorkItemID || record.Request.AttemptID != candidate.AttemptID || record.Request.Plan != candidate.Plan || record.Request.Grant != candidate.Grant || record.Request.Policy != candidate.Policy { return errors.New("acceptance test provenance mismatch") }
		receipt, err := v.receipt(state, *record)
		if err != nil { return err }
		if receipt.Result.Status != "passed" || !equalJSON(receipt.Inputs, candidate.Inputs) { return errors.New("required tests are failed, waived, absent or stale") }
		_, err = v.Store.CheckTestManifest(state, record.Request)
		return err
	}
	if len(candidate.TestRunIDs) != 0 || candidate.NotApplicableRule != candidate.Policy || strings.TrimSpace(candidate.NotApplicableReason) == "" { return errors.New("N/A requires an explicit current applicability rule") }
	var policy TestExecutionPolicy
	if err := v.Store.readStrictArtifact(state.Task.ID, candidate.Policy, &policy); err != nil { return err }
	if policy.TaskID != state.Task.ID || policy.WorkspaceDigest != WorkspaceBindingDigest(state) { return errors.New("N/A policy binding is stale") }
	var input ExecutionInput
	if err := v.Store.ReadExecutionArtifact(state.Task.ID, origin.Request.Input, &input); err != nil { return err }
	if input.WorkspaceInputs == nil { return errors.New("N/A requires a complete pre-change manifest") }
	changed := changedValidationPaths(*input.WorkspaceInputs, candidate.Inputs)
	if len(changed) == 0 { return errors.New("N/A change scope is unknown") }
	for _, path := range changed {
		allowed := false
		for _, ref := range policy.DescriptiveDocuments {
			for _, file := range candidate.Inputs.Files { if ref.Kind == "descriptive-document" && ref.Path == path && ref.Path == file.Path && ref.SHA256 == file.SHA256 { allowed = true } }
		}
		if !allowed { return errors.New("N/A is limited to policy-reviewed descriptive document content") }
	}
	return nil
}

func (v *VerificationService) AcceptanceCandidate(state RuntimeState, itemID WorkItemID, reportRef ActorReference, policyRef ActorReference, notApplicableReason string) (AcceptanceCandidate, error) {
	c := AcceptanceCandidate{WorkItemID: itemID, Report: reportRef, Policy: policyRef, TestsRequired: notApplicableReason == "", NotApplicableReason: notApplicableReason}
	item, err := executionItem(&state, itemID)
	if err != nil { return c, err }
	c.AttemptID, c.CurrentRequestID = item.AttemptID, item.CurrentRequest
	for i := len(state.Protocol.Requests)-1; i >= 0; i-- {
		r := state.Protocol.Requests[i]
		if r.Request.WorkItemID != itemID || r.Request.AttemptID != item.AttemptID || !r.Consumed { continue }
		switch r.Request.Phase {
		case PhaseCoder: if c.RequestID == "" { c.RequestID = r.Request.ID }
		case PhaseCompile:
			if c.CompileRunID == "" {
				c.CompileRunID = r.Request.ID
				if !c.TestsRequired { receipt, err := v.receipt(state, r); if err != nil { return c, err }; c.Inputs, c.Plan, c.NotApplicableRule = receipt.Inputs, r.Request.Plan, policyRef }
			}
		case PhaseTest:
			if c.TestsRequired && len(c.TestRunIDs) == 0 { receipt, err := v.receipt(state, r); if err != nil { return c, err }; c.Inputs, c.Plan, c.Grant, c.Policy = receipt.Inputs, r.Request.Plan, r.Request.Grant, r.Request.Policy; c.TestRunIDs = []string{r.Request.ID} }
		}
	}
	return c, v.ValidateAcceptance(state, c)
}
