package execution

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"aiw/internal/ai"
	"aiw/internal/session"
	"aiw/internal/task"
	"aiw/internal/workflow"
)

// recordSupervisorSessionOutcome archives the exactly bound response before
// interpreting it. Saving a report does not accept or complete the Work Item.
func recordSupervisorSessionOutcome(store *workflow.Store, request *workflow.PreparedAgentRequest) (workflow.SupervisedOutcome, error) {
	if request == nil || request.SessionID == "" {
		return workflow.SupervisedOutcome{}, fmt.Errorf("managed Session binding is required")
	}
	if request.DispatchedAt == "" {
		return workflow.SupervisedOutcome{}, fmt.Errorf("session-result-not-dispatched: Session %s was prepared but was not dispatched for Attempt %s", request.SessionID, request.AttemptID)
	}
	status, err := session.NewStore("").Load(request.SessionID)
	if err != nil {
		return workflow.SupervisedOutcome{}, err
	}
	if status.Result.Status != "completed" || status.Result.FinalOutputFile == "" {
		return workflow.SupervisedOutcome{}, fmt.Errorf("managed Session result is incomplete")
	}
	if err := validateSupervisorSessionResult(status, request); err != nil {
		return workflow.SupervisedOutcome{}, err
	}
	reference := ".ai/sessions/" + request.SessionID + "/" + status.Result.FinalOutputFile
	output, err := session.NewStore("").ReadText(request.SessionID, status.Result.FinalOutputFile)
	if err != nil {
		return workflow.SupervisedOutcome{}, err
	}
	if store == nil {
		return workflow.SupervisedOutcome{}, fmt.Errorf("workflow Store is required to preserve the execution report")
	}
	report, err := store.PersistExecutionReport(*request, reference, output)
	if err != nil {
		return workflow.SupervisedOutcome{}, fmt.Errorf("preserve execution report: %w", err)
	}
	archived, err := executionReportEvidencePath(request.TaskID, report.Path)
	if err != nil {
		return workflow.SupervisedOutcome{}, fmt.Errorf("resolve execution report reference: %w", err)
	}
	provider, model, profile, level, intensity, parameterDigest := status.Backend.Name, status.Backend.Model, "unknown", 0, "unknown", ""
	if request.AISelection != nil {
		if request.AISelection.Provider != "" { provider = request.AISelection.Provider }
		if request.AISelection.Model != "" { model = request.AISelection.Model }
		if strings.TrimSpace(request.AISelection.Profile) != "" { profile = request.AISelection.Profile }
		level, intensity, parameterDigest = request.AISelection.Level, request.AISelection.ReasoningIntensity, request.AISelection.Digest
		if intensity == "" { intensity = "unknown" }
	}
	if provider == "" { provider = "unknown" }
	if model == "" { model = "unknown" }
	usage, err := session.NewStore("").ReadTurnUsage(request.SessionID, request.ExpectedSessionTurn, provider, model)
	if err != nil { return workflow.SupervisedOutcome{}, fmt.Errorf("read managed Session usage: %w", err) }
	usageJSON, err := json.Marshal(ai.UsageWithoutRawEvidence(usage))
	if err != nil { return workflow.SupervisedOutcome{}, fmt.Errorf("encode managed Session usage: %w", err) }
	reportedOutcome := parseSupervisedOutcome(output, archived)
	taskState, err := store.Load(request.TaskID)
	if err != nil { return workflow.SupervisedOutcome{}, err }
	if taskState.SchemaVersion == workflow.DurableSchemaVersion && taskState.Protocol != nil {
		usageEvent := workflow.UsageEvent{
			TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID,
			SessionID: request.SessionID, SessionTurn: request.ExpectedSessionTurn,
			Provider: provider, Model: model, Profile: profile, DifficultyLevel: level,
			ReasoningIntensity: intensity, ParameterDigest: parameterDigest,
			Outcome: string(reportedOutcome.Kind), Usage: usageJSON,
		}
		if request.AISelection != nil {
			usageEvent.RequestedLevel = request.AISelection.RequestedLevel
			usageEvent.AdjustmentReason = request.AISelection.AdjustmentReason
			usageEvent.PreviousProfile = request.AISelection.PreviousProfile
			usageEvent.PreviousProvider = request.AISelection.PreviousProvider
			usageEvent.PreviousModel = request.AISelection.PreviousModel
			usageEvent.PreviousLevel = request.AISelection.PreviousLevel
			usageEvent.PreviousReasoningIntensity = request.AISelection.PreviousReasoningIntensity
		}
		if _, err := store.RecordUsageEvent(request.TaskID, usageEvent); err != nil { return workflow.SupervisedOutcome{}, fmt.Errorf("record managed Session usage: %w", err) }
	}
	if request.InputReference != nil {
		var reported agentSupervisedOutcome
		if err := json.Unmarshal([]byte(output), &reported); err != nil { return workflow.SupervisedOutcome{}, &reportValidationError{reason: err.Error()} }
		if err := workflow.ValidateSupervisedOutcome(workflow.SupervisedOutcome{Kind: reported.Outcome, BlockedCategory: reported.BlockedCategory, Detail: reported.Detail, EvidenceReference: archived}); err != nil { return workflow.SupervisedOutcome{}, &reportValidationError{reason: err.Error()} }
		root := request.Workspace
		if !filepath.IsAbs(root) { root = filepath.Join(filepath.Dir(store.Root), filepath.FromSlash(root)) }
		structured, err := workflow.ValidateImplementationReport(output, *request, root)
		if err != nil {
			return workflow.SupervisedOutcome{}, &reportValidationError{reason: err.Error()}
		}
		origin := request
		if request.ReportOrigin != nil { origin = request.ReportOrigin }
		var input workflow.ExecutionInput
		if err := store.ReadExecutionArtifact(request.TaskID, *origin.InputReference, &input); err != nil { return workflow.SupervisedOutcome{}, err }
		if err := workflow.ValidateReportChanges(structured, input.WorkspaceInputs, root); err != nil { return workflow.SupervisedOutcome{}, &reportValidationError{reason: err.Error()} }
	}
	return normalizeSupervisorSessionOutcome(reportedOutcome), nil
}

// executionReportEvidencePath returns a repository-relative path for an
// execution report. Runtime Task directories are canonicalized under
// .ai/tasks/<id>; constructing this from RuntimeTaskDir keeps the reference in
// sync with Store.path instead of relying on the removed legacy .ai/<id> layout.
func executionReportEvidencePath(taskID workflow.TaskID, reportPath string) (string, error) {
	reportFile := filepath.Join(task.RuntimeTaskDir(string(taskID)), filepath.FromSlash(reportPath))
	root, err := filepath.Abs(task.RuntimeRoot())
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, reportFile)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relative), nil
}

type reportValidationError struct { reason string }
func (e *reportValidationError) Error() string { return "implementation report invalid: " + e.reason }

// validateSupervisorSessionResult rejects a different execution, including a
// later turn in the same Session. Session latest is not a request identity.
func validateSupervisorSessionResult(status session.Status, request *workflow.PreparedAgentRequest) error {
	if request == nil || request.TaskID == "" || request.WorkItemID == "" || request.AttemptID == "" || request.SessionID == "" {
		return fmt.Errorf("session-result-stale: exact managed request binding is required")
	}
	if status.Task == nil || status.Task.TaskID != string(request.TaskID) || status.Task.WorkItemID != string(request.WorkItemID) || status.Task.AttemptID != string(request.AttemptID) || status.Session.ID != request.SessionID {
		return fmt.Errorf("session-result-stale: Session %s has no completed result for Attempt %s", request.SessionID, request.AttemptID)
	}
	if request.ExpectedSessionTurn <= 0 || status.Session.LastTurn != request.ExpectedSessionTurn {
		return fmt.Errorf("session-result-stale: Session %s is at turn %d, expected turn %d for Attempt %s", request.SessionID, status.Session.LastTurn, request.ExpectedSessionTurn, request.AttemptID)
	}
	if status.Result.FinalOutputFile != fmt.Sprintf("outputs/%04d-final.txt", request.ExpectedSessionTurn) {
		return fmt.Errorf("session-result-stale: final output does not name the expected turn")
	}
	return nil
}

// normalizeSupervisorSessionOutcome reserves workspace-access classification for Core preflight.
func normalizeSupervisorSessionOutcome(outcome workflow.SupervisedOutcome) workflow.SupervisedOutcome {
	// workspace-access is Core-owned evidence from the scoped Git preflight.
	// An Agent can report its observation but cannot establish that the current
	// process lacks that access, especially when its handoff is stale.
	if outcome.Kind == workflow.SupervisedOutcomeBlocked && outcome.BlockedCategory == workflow.BlockedOutcomeWorkspaceAccess {
		outcome.BlockedCategory = workflow.BlockedOutcomeUnknown
		outcome.Detail = "Agent reported workspace access after supervisor preflight; inspect the recorded session output and current scoped Git evidence: " + outcome.Detail
	}
	return outcome
}
