package main

import (
	taskcmd "aiw/internal/commands/task"
	"aiw/internal/fsx"
	"aiw/internal/issue"
	"aiw/internal/task"
	taskadapter "aiw/internal/task/workflowadapter"
	workflowcli "aiw/internal/workflow/cli"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func promoteIssue(args []string) error {
	reqID, taskID, allowUnrelatedDirty, err := parsePromoteArgs(args)
	if err != nil {
		return err
	}
	meta, err := issue.Read(reqID)
	if err != nil {
		return err
	}
	if (meta.Status != "APPROVED" && meta.Status != "PROMOTED") || meta.Approval.Status != "APPROVED" {
		return errors.New("requirement is not approved")
	}
	if meta.Promotion.TaskID != "" && meta.Promotion.TaskID != taskID {
		return fmt.Errorf("requirement already links to task %s", meta.Promotion.TaskID)
	}
	if meta.Promotion.Status == "SPEC_DRAFTED" || meta.Promotion.Status == "FD_READY" {
		if !fsx.Exists(task.RuntimeTaskDir(taskID)) {
			return fmt.Errorf("promoted Issue task not found: %s", taskID)
		}
		fmt.Printf("issue %s already links to task %s\n", reqID, taskID)
		return nil
	}
	if meta.Promotion.TaskID == "" && !fsx.Exists(task.RuntimeTaskDir(taskID)) {
		if err := taskcmd.CreateTaskFromIssue(taskID, reqID, allowUnrelatedDirty); err != nil {
			return err
		}
	}
	if !fsx.Exists(task.RuntimeTaskDir(taskID)) {
		return fmt.Errorf("promoted task not found: %s", taskID)
	}
	if meta.Promotion.TaskID == "" {
		meta, _, err = issue.StartPromotion(reqID, taskID)
		if err != nil {
			return err
		}
	}
	snapshot, err := issue.ArtifactSnapshot(reqID)
	if err != nil {
		return err
	}
	if err := writeRequirementHandoff(meta, snapshot); err != nil {
		return err
	}
	checklistPath := task.FeatureDesignPath(taskID)
	if !fsx.Exists(checklistPath) {
		checklistPath = filepath.Join(task.TaskDir(taskID), "tasks.md")
	}
	items, err := task.ReadWorkflowChecklist(checklistPath)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(items) == 0) {
		fmt.Printf("issue %s linked to task %s; awaiting Feature Design work items\n", reqID, taskID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("read Feature Design work items: %w", err)
	}
	if checklistPath == task.FeatureDesignPath(taskID) {
		readiness, err := task.ReadFeatureDesignReadiness(checklistPath)
		if err != nil {
			return err
		}
		if readiness == "BLOCKED" {
			fmt.Printf("issue %s linked to task %s; awaiting Feature Design readiness\n", reqID, taskID)
			return nil
		}
	}
	if err := taskcmd.EnsureChecklistMapping(taskID); err != nil {
		return fmt.Errorf("map Feature Design work items: %w", err)
	}
	if err := completePromotion(reqID, taskID); err != nil {
		return err
	}
	fmt.Printf("issue %s promoted to task %s from Feature Design\n", reqID, taskID)
	return nil
}

func completePromotion(requirementID, taskID string) error {
	// Advisory routing is intentionally non-blocking. recommend-routing falls
	// back to deterministic defaults when no LLM is configured; an unexpected
	// routing persistence failure remains visible without downgrading accepted
	// artifact content back to an incomplete promotion.
	if err := workflowcli.RunWorkflowCommand(taskadapter.New(taskadapter.DefaultOperations{}),
		[]string{"recommend-routing", taskID}); err != nil {
		fmt.Fprintf(os.Stderr, "routing recommendation unavailable for task %s: %v\n", taskID, err)
	}
	_, err := issue.CompletePromotion(requirementID, taskID)
	return err
}

func parsePromoteArgs(args []string) (string, string, bool, error) {
	if (len(args) != 3 && len(args) != 4) || args[1] != "--task" || !issue.ValidID(args[2]) || (len(args) == 4 && args[3] != "--allow-unrelated-dirty") {
		usage, _ := issueSubcommandUsage("promote")
		return "", "", false, errors.New(usage)
	}
	return args[0], args[2], true, nil
}

func writeRequirementHandoff(meta issue.Meta, artifacts []issue.Artifact) error {
	dir := filepath.Join(task.RuntimeTaskDir(meta.Promotion.TaskID), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "requirement-handoff.md")
	if fsx.Exists(path) {
		return nil
	}
	var rows []string
	for _, artifact := range artifacts {
		rows = append(rows, fmt.Sprintf("| %s | %s | %s |", artifact.Kind, artifact.Path, artifact.Digest))
	}
	content := "# Issue Handoff\n\n## Source\n- Issue ID: " + meta.ID + "\n- Issue revision: " + fmt.Sprint(meta.Revision) + "\n- Approved by: " + meta.Approval.By + "\n- Approved at: " + meta.Approval.At + "\n\n## Referenced Artifacts\n| Artifact | Path | Digest |\n|---|---|---|\n" + strings.Join(rows, "\n") + "\n\n## Approved Scope\n- Use the approved Issue Plan and referenced artifacts as the authoritative scope.\n\n## Open Decisions Carried Into Engineering\n%% NEEDS_INPUT: Review approved artifacts before finalizing the Feature Design.\n\n## Suggested Next Workflow Action\nComplete docs/features/" + meta.Promotion.TaskID + ".md with decisions and ordered work items, then run aiw wf plan " + meta.Promotion.TaskID + ". Update stable OpenSpec specs only when their requirements change.\n"
	return os.WriteFile(path, []byte(content), 0o644)
}
