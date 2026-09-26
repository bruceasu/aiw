package cli

import (
	"errors"
	"fmt"
	"os"

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
	updated, err := store.ConsumeRemediationResponse(id, response)
	if err != nil {
		return err
	}
	switch option.Action {
	case workflow.RemediationActionRetryWorkItem, workflow.RemediationActionResumeSupervisor:
		return startWorkflowSupervisor(args[1], "", "", store, updated)
	case workflow.RemediationActionRepairProjection:
		return repairWorkflowState(args[1])
	case workflow.RemediationActionRepairSession:
		return repairWorkflowState(args[1])
	case workflow.RemediationActionHumanReview, workflow.RemediationActionStop:
		fmt.Printf("Workflow remains paused; review the evidence before submitting another response.\n")
		return nil
	default:
		return fmt.Errorf("unsupported remediation action: %s", option.Action)
	}
}

func printRemediationChoices(report workflow.RemediationReport) {
	fmt.Printf("remediation requires a human choice: problem=%s\n", report.ProblemID)
	fmt.Printf("report: %s\n", report.ReportPath)
	fmt.Printf("response: %s\n", report.HumanResponsePath)
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
