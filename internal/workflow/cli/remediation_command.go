package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"aiw/internal/session"
	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow"
)

func runRemediationResponse(args []string) error {
	return runRemediationResponseWithStore(args, workflow.NewStore(""))
}

func runRemediationResponseWithStore(args []string, store *workflow.Store) error {
	if len(args) != 2 || !taskAdapter.SafeID(args[1]) {
		return fmt.Errorf("usage: wf <continue|resume> <task-id>")
	}
	id := workflow.TaskID(args[1])
	_, report, err := store.ReadPendingRemediation(id)
	if err != nil {
		if state, loadErr := store.Load(id); loadErr == nil && state.Automation.Cursor.Result == "remediation-response-consumed" {
			fmt.Printf("remediation response already consumed for Task %s; no action repeated\n", id)
			return nil
		}
		return err
	}
	response, err := store.ReadRemediationResponse(id, report)
	if errors.Is(err, os.ErrNotExist) {
		printRemediationChoices(report)
		return nil
	}
	if err != nil {
		return fmt.Errorf("read human remediation response: %w", err)
	}
	var option workflow.RemediationOption
	for _, candidate := range report.Options {
		if candidate.ID == response.OptionID {
			option = candidate
			break
		}
	}
	if option.Action == workflow.RemediationActionHumanReview {
		fmt.Printf("human review selected; Workflow remains paused and the response is not consumed.\n")
		return nil
	}
	fmt.Printf("accepted remediation choice %s for problem %s\n", option.ID, report.ProblemID)
	fmt.Printf("risk: %s\n", option.Risk)
	fmt.Printf("scope: %s\n", option.Scope)
	if option.Action == workflow.RemediationActionRepairProjection || option.Action == workflow.RemediationActionRepairSession {
		var repairErr error
		if option.Action == workflow.RemediationActionRepairProjection {
			repairErr = repairRemediationProjection(args[1], store, report.Problem.WorkItemID)
		} else {
			repairErr = repairRemediationSession(args[1], store, report.Problem.AttemptID)
		}
		if repairErr != nil {
			return fmt.Errorf("remediation repair failed; response remains pending: %w", repairErr)
		}
		if _, err := store.ConsumeRemediationResponse(id, response); err != nil {
			return err
		}
		fmt.Printf("repair completed; next: aiw wf supervise %s start\n", args[1])
		return nil
	}
	if option.Action == workflow.RemediationActionResolveGate || option.Action == workflow.RemediationActionWaiveGate {
		updated, err := store.ApplyGateRemediationResponse(id, response)
		if err != nil {
			return fmt.Errorf("Gate decision failed; response remains pending: %w", err)
		}
		return startWorkflowSupervisor(args[1], "", "", store, updated)
	}
	if report.Problem.Category == "session-result-unknown" && option.Action == workflow.RemediationActionResumeSupervisor {
		if err := validateUnknownSessionResume(store, report); err != nil {
			return fmt.Errorf("Session result cannot be re-observed safely; response remains pending: %w", err)
		}
	}
	updated, err := store.ConsumeRemediationResponse(id, response)
	if err != nil {
		return err
	}
	switch option.Action {
	case workflow.RemediationActionRetryWorkItem, workflow.RemediationActionResumeSupervisor:
		return startWorkflowSupervisor(args[1], "", "", store, updated)
	case workflow.RemediationActionHumanReview, workflow.RemediationActionStop:
		fmt.Printf("Workflow remains paused; review the evidence before submitting another response.\n")
		return nil
	default:
		return fmt.Errorf("unsupported remediation action: %s", option.Action)
	}
}

func validateUnknownSessionResume(store *workflow.Store, report workflow.RemediationReport) error {
	state, err := store.Load(report.Problem.TaskID)
	if err != nil { return err }
	request := state.Automation.PreparedRequest
	problem := report.Problem
	if request == nil || request.DispatchedAt == "" || request.TaskID != problem.TaskID || request.WorkItemID != problem.WorkItemID || request.AttemptID != problem.AttemptID || request.SessionID == "" || request.SessionID != problem.SessionID || request.ExpectedSessionTurn != problem.SessionTurn || state.WriteLease == nil || state.WriteLease.AttemptID != request.AttemptID {
		return fmt.Errorf("the original dispatched Attempt, Session turn, or write lease is no longer current")
	}
	return nil
}

func repairRemediationProjection(id string, store *workflow.Store, workItemID workflow.WorkItemID) error {
	state, err := store.Load(workflow.TaskID(id))
	if err != nil {
		return err
	}
	target := "tasks.md/" + string(workItemID)
	var selected *workflow.ProjectionRepair
	for index := range state.Automation.ProjectionRepairs {
		repair := &state.Automation.ProjectionRepairs[index]
		if repair.ResolvedAt != "" || repair.Target != target {
			continue
		}
		if selected != nil {
			return fmt.Errorf("multiple pending projection repairs for %s", target)
		}
		selected = repair
	}
	if selected == nil {
		return fmt.Errorf("no pending projection repair for %s", target)
	}
	updated, err := taskworkflow.RetryChecklistProjection(id, store, *selected)
	if err != nil {
		return err
	}
	return projectWorkflowState(id, updated)
}

func repairRemediationSession(id string, store *workflow.Store, attemptID workflow.AttemptID) error {
	state, err := store.Load(workflow.TaskID(id))
	if err != nil {
		return err
	}
	if state.WriteLease == nil || state.WriteLease.AttemptID != attemptID || state.Automation.PreparedRequest != nil {
		return fmt.Errorf("Attempt %s is not eligible for missing-Session repair", attemptID)
	}
	for _, attempt := range state.Attempts {
		if attempt.ID != attemptID {
			continue
		}
		if attempt.SessionID == "" {
			return fmt.Errorf("Attempt %s has no Session binding", attemptID)
		}
		if _, err := session.NewStore("").Load(attempt.SessionID); !errors.Is(err, session.ErrSessionNotFound) {
			if err == nil {
				return fmt.Errorf("Session %s still exists; missing-Session repair is unsafe", attempt.SessionID)
			}
			return err
		}
		updated, err := store.RepairMissingSessionAttempt(workflow.TaskID(id), attemptID)
		if err != nil {
			return err
		}
		return projectWorkflowState(id, updated)
	}
	return fmt.Errorf("unknown Attempt %s", attemptID)
}

func printRemediationChoices(report workflow.RemediationReport) {
	fmt.Printf("remediation requires a human choice: problem=%s\n", report.ProblemID)
	dir := task.RuntimeTaskDir(string(report.Problem.TaskID))
	fmt.Printf("report: %s\n", filepath.Join(dir, filepath.FromSlash(report.ReportPath)))
	fmt.Printf("response: %s\n", filepath.Join(dir, filepath.FromSlash(report.HumanResponsePath)))
	if report.Diagnosis != nil {
		fmt.Printf("cause: %s (confidence %.2f)\n", report.Diagnosis.Summary, report.Diagnosis.Confidence)
		fmt.Printf("reason: %s\n", report.Diagnosis.Reason)
	}
	for index, option := range report.Options {
		fmt.Printf("%d. %s [%s]\n", index+1, option.ID, option.Summary)
		fmt.Printf("   scope: %s\n   risk: %s\n   resources: %s\n   external effect: %s\n   rollback: %s\n", option.Scope, option.Risk, option.ResourceImpact, option.ExternalEffect, option.Rollback)
	}
	fmt.Printf("edit the response file, set option_id/operator/risk_confirmed, then run: aiw wf continue %s\n", report.Problem.TaskID)
}
