package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func goChangedPaths(paths []string) bool {
	for _, path := range paths { lower := strings.ToLower(path); if strings.HasSuffix(lower, ".go") && !strings.HasSuffix(lower, "_test.go") { return true } }
	return false
}

// PersistGoAcceptance freezes the evidence before the owning Attempt closes.
// Its artifact is a candidate until RecordSupervisedOutcome links it to a
// completed Work Item; an unlinked file grants no acceptance.
func (s *Store) PersistGoAcceptance(request PreparedAgentRequest, outcome SupervisedOutcome, tester TesterResult) (ActorReference, error) {
	if outcome.Kind != SupervisedOutcomeCompleted || request.InputReference == nil || request.Compile == nil || request.Compile.Plan == nil || request.Compile.Inputs == nil || request.Compile.Request == nil || request.Compile.Result == nil {
		return ActorReference{}, errors.New("Go acceptance requires a completed report and controlled compile")
	}
	compile := request.Compile
	if compile.Result.Status != ActorResultAccepted || compile.Result.RequestID != compile.Request.ID || compile.Request.AttemptID != request.AttemptID {
		return ActorReference{}, errors.New("Go acceptance compile does not match the Attempt")
	}
	if tester.Status != ActorResultAccepted || tester.WorkItemID != request.WorkItemID || tester.AttemptID != request.AttemptID || tester.StaticReview.Kind != "go-static-review" || tester.StaticReview.SHA256 == "" {
		return ActorReference{}, errors.New("Go acceptance requires an independent passed Tester review")
	}
	state, err := s.Load(request.TaskID)
	if err != nil { return ActorReference{}, err }
	var item *WorkItem
	for i := range state.WorkItems { if state.WorkItems[i].ID == request.WorkItemID { item = &state.WorkItems[i]; break } }
	if item == nil { return ActorReference{}, errors.New("Go acceptance work item is missing") }
	bindings, err := CompileInputBindings(compile.Plan)
	if err != nil { return ActorReference{}, err }
	toolchain, err := ValidationToolchainIdentity()
	if err != nil { return ActorReference{}, err }
	current, err := CaptureValidationInputs(request.Workspace, compile.Inputs.Scope, bindings, toolchain)
	if err != nil { return ActorReference{}, err }
	if applicable, reason := EvidenceApplicability(compile.Inputs, current, true, true); !applicable { return ActorReference{}, fmt.Errorf("Go compile evidence is stale: %s", reason) }
	review, err := s.readGoStaticReview(request.TaskID, tester.StaticReview)
	if err != nil { return ActorReference{}, err }
	if review.WorkItemID != request.WorkItemID || review.AttemptID != request.AttemptID || review.State != EvidencePassed || review.Inputs == nil || len(review.Sources) == 0 || review.InputSHA256 == "" || len(review.Inputs.Bindings) != 1 || review.Inputs.Bindings[0].SHA256 != review.InputSHA256 {
		return ActorReference{}, errors.New("Go static review is not bound to the current Attempt and interface inventory")
	}
	currentReview, err := CaptureValidationInputs(request.Workspace, review.Inputs.Scope, review.Inputs.Bindings, review.Inputs.Toolchain)
	if err != nil { return ActorReference{}, err }
	if applicable, reason := EvidenceApplicability(review.Inputs, currentReview, true, true); !applicable { return ActorReference{}, fmt.Errorf("Go static review is stale: %s", reason) }
	reviewPassed := false
	for _, evidence := range state.Evidence {
		if evidence.WorkItemID == request.WorkItemID && evidence.Kind == EvidenceStaticReview && evidence.State == EvidencePassed && evidence.Reference == tester.StaticReview.Path+"#sha256="+tester.StaticReview.SHA256 { reviewPassed = true; break }
	}
	if !reviewPassed { return ActorReference{}, errors.New("Go static review has no matching passed Core evidence") }
	reportPrefix := ".ai/tasks/" + string(request.TaskID) + "/"
	if !strings.HasPrefix(outcome.EvidenceReference, reportPrefix) { return ActorReference{}, errors.New("Go implementation report is not Task-owned") }
	reportPath := strings.TrimPrefix(outcome.EvidenceReference, reportPrefix)
	if !strings.HasPrefix(reportPath, "reports/executions/") { return ActorReference{}, errors.New("Go implementation report is not a preserved execution report") }
	reportContent, err := os.ReadFile(s.path(request.TaskID, filepath.FromSlash(reportPath)))
	if err != nil { return ActorReference{}, err }
	reportRef := ActorReference{Kind: "execution-report", Path: reportPath, SHA256: contentDigest(reportContent)}
	var report ExecutionReport
	if err := s.ReadExecutionArtifact(request.TaskID, reportRef, &report); err != nil { return ActorReference{}, err }
	if report.TaskID != request.TaskID || report.WorkItemID != request.WorkItemID || report.AttemptID != request.AttemptID || report.InputReference == nil || *report.InputReference != *request.InputReference { return ActorReference{}, errors.New("Go implementation report identity is stale") }
	testsRequired := item.Verification == "mixed"
	var testRuns []string
	if testsRequired {
		for _, evidence := range state.Evidence { if evidence.WorkItemID == item.ID && evidence.Kind == EvidenceCommand && evidence.State == EvidencePassed && evidence.Reference != "" { testRuns = append(testRuns, string(evidence.ID)) } }
		if len(testRuns) == 0 { return ActorReference{}, errors.New("mixed-language work requires passed non-Go test evidence") }
	}
	record := ExecutionEvidence{SchemaVersion: 1, WorkItemID: request.WorkItemID, RequestID: ExecutionRequestID(request), AttemptID: request.AttemptID, Report: &reportRef, Inputs: compile.Inputs, CompileRunID: compile.Request.ID, StaticReview: &tester.StaticReview, StaticReviewInputs: review.Inputs, GoSources: review.Sources, TestRunIDs: testRuns, TestsRequired: testsRequired, Controlled: true, Passed: true, Accepted: true}
	if !testsRequired { record.TestsNotApplicableReason = "Go static-review policy" }
	return s.persistExecutionArtifact(request, "accepted-execution", "accepted/"+string(request.AttemptID)+".json", record)
}

func (s *Store) AttachGoAcceptance(id TaskID, request PreparedAgentRequest, reference ActorReference) (RuntimeState, error) {
	if reference.Kind != "accepted-execution" || reference.SHA256 == "" { return RuntimeState{}, errors.New("Go acceptance reference is invalid") }
	var record ExecutionEvidence
	if err := s.ReadExecutionArtifact(id, reference, &record); err != nil { return RuntimeState{}, err }
	if record.WorkItemID != request.WorkItemID || record.AttemptID != request.AttemptID || record.RequestID != ExecutionRequestID(request) || !record.Accepted { return RuntimeState{}, errors.New("Go acceptance reference does not match the prepared request") }
	return s.UpdateWithEvent(id, Event{Type: "go-acceptance.prepared", WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Detail: reference.Path}, func(state *RuntimeState) error {
		current := state.Automation.PreparedRequest
		if current == nil || current.AttemptID != request.AttemptID || current.WorkItemID != request.WorkItemID || ExecutionRequestID(*current) != ExecutionRequestID(request) { return errors.New("Go acceptance lost its prepared request") }
		current.GoAcceptance = &reference
		return nil
	})
}

func (s *Store) requireCurrentGoAcceptance(state RuntimeState, item WorkItem) error {
	if item.Verification != "go" && item.Verification != "mixed" && !item.RequiresGoEvidence { return nil }
	if item.AcceptedReference == nil || item.AcceptedReference.Kind != "accepted-execution" { return fmt.Errorf("Go work item %s lacks accepted execution evidence", item.ID) }
	var record ExecutionEvidence
	if err := s.ReadExecutionArtifact(state.Task.ID, *item.AcceptedReference, &record); err != nil { return err }
	if record.WorkItemID != item.ID || record.AttemptID == "" || record.Report == nil || record.Inputs == nil || record.StaticReview == nil || record.StaticReviewInputs == nil || len(record.GoSources) == 0 || !record.Accepted || !record.Controlled || !record.Passed { return errors.New("Go acceptance artifact is incomplete") }
	var report ExecutionReport
	if err := s.ReadExecutionArtifact(state.Task.ID, *record.Report, &report); err != nil { return err }
	if report.WorkItemID != item.ID || report.AttemptID != record.AttemptID { return errors.New("Go acceptance report identity is stale") }
	review, err := s.readGoStaticReview(state.Task.ID, *record.StaticReview)
	if err != nil { return err }
	if review.WorkItemID != item.ID || review.AttemptID != record.AttemptID || review.State != EvidencePassed || review.Inputs == nil || review.Inputs.Digest() != record.StaticReviewInputs.Digest() { return errors.New("Go acceptance static review identity is stale") }
	root := state.Task.Workspace
	bindings := record.Inputs.Bindings
	toolchain, err := ValidationToolchainIdentity()
	if err != nil { return err }
	current, err := CaptureValidationInputs(root, record.Inputs.Scope, bindings, toolchain)
	if err != nil { return err }
	if decision := AssessExecutionReuse(record, current, false, true); decision.Action != "reuse" { return fmt.Errorf("Go acceptance is stale: %s", decision.Reason) }
	currentReview, err := CaptureValidationInputs(root, record.StaticReviewInputs.Scope, record.StaticReviewInputs.Bindings, record.StaticReviewInputs.Toolchain)
	if err != nil { return err }
	if applicable, reason := EvidenceApplicability(record.StaticReviewInputs, currentReview, true, true); !applicable { return fmt.Errorf("Go static review is stale: %s", reason) }
	if item.Verification == "mixed" && (!record.TestsRequired || len(record.TestRunIDs) == 0) { return errors.New("mixed-language acceptance lacks non-Go test evidence") }
	return nil
}
