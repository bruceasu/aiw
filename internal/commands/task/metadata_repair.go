package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

type metadataRepairOptions struct {
	ID     string
	DryRun bool
}

func parseMetadataRepairOptions(args []string) (metadataRepairOptions, error) {
	options := metadataRepairOptions{}
	for _, arg := range args {
		switch arg {
		case "--dry-run":
			if options.DryRun {
				return options, fmt.Errorf("duplicate repair-metadata option: %s", arg)
			}
			options.DryRun = true
		default:
			if options.ID != "" || !safeID(arg) {
				return options, fmt.Errorf("usage: task workflow repair-metadata [task-id] [--dry-run]")
			}
			options.ID = arg
		}
	}
	return options, nil
}

func repairHistoricalMetadata(options metadataRepairOptions) error {
	locations, err := taskx.DiscoverTaskLocations()
	if err != nil {
		return err
	}
	found, repaired, unchanged, skipped := false, 0, 0, 0
	for _, location := range locations {
		if options.ID != "" && location.ID != options.ID {
			continue
		}
		found = true
		result := repairHistoricalLocation(location, options.DryRun)
		switch result.kind {
		case metadataRepairRepaired:
			repaired++
		case metadataRepairUnchanged:
			unchanged++
		default:
			skipped++
		}
		fmt.Printf("metadata-repair task=%s location=%s result=%s: %s\n", location.ID, metadataRepairLocation(location), result.kind, result.detail)
	}
	if !found && options.ID != "" {
		return fmt.Errorf("task not found for metadata repair: %s", options.ID)
	}
	mode := "applied"
	if options.DryRun {
		mode = "preview"
	}
	fmt.Printf("metadata-repair %s: repaired=%d unchanged=%d skipped=%d\n", mode, repaired, unchanged, skipped)
	return nil
}

type metadataRepairResult struct {
	kind   string
	detail string
}

const (
	metadataRepairRepaired  = "repaired"
	metadataRepairUnchanged = "unchanged"
	metadataRepairSkipped   = "skipped"
)

func repairHistoricalLocation(location taskx.TaskLocation, dryRun bool) metadataRepairResult {
	if err := errors.Join(location.Problems...); err != nil {
		return metadataRepairResult{metadataRepairSkipped, "discovery conflict: " + err.Error()}
	}
	metaPath, err := taskx.MetadataPathInDirectory(location.RuntimeDir)
	if err != nil || metaPath == "" {
		if err == nil {
			err = errors.New("task metadata is missing")
		}
		return metadataRepairResult{metadataRepairSkipped, "identity evidence unavailable: " + err.Error()}
	}
	meta, err := taskx.ReadTaskMeta(metaPath)
	if err != nil || meta.ID != location.ID {
		if err == nil {
			err = fmt.Errorf("metadata id is %q", meta.ID)
		}
		return metadataRepairResult{metadataRepairSkipped, "identity evidence unavailable: " + err.Error()}
	}
	store := workflow.NewStore("")
	if dryRun {
		locked, lockErr := store.TaskLocked(workflow.TaskID(location.ID))
		if lockErr != nil {
			return metadataRepairResult{metadataRepairSkipped, "inspect Task lock: " + lockErr.Error()}
		}
		if locked {
			return metadataRepairResult{metadataRepairSkipped, "Task is occupied by a Workflow Core writer"}
		}
		return previewHistoricalMetadata(location, meta, metaPath)
	}
	release, err := store.AcquireTaskLock(workflow.TaskID(location.ID))
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return metadataRepairResult{metadataRepairSkipped, "Task is occupied by a Workflow Core writer"}
		}
		return metadataRepairResult{metadataRepairSkipped, "acquire Task lock: " + err.Error()}
	}
	defer func() { _ = release() }()
	return applyHistoricalMetadata(location, meta, metaPath)
}

func previewHistoricalMetadata(location taskx.TaskLocation, meta taskx.TaskMeta, metaPath string) metadataRepairResult {
	plan, reason := planHistoricalMetadataRepair(location, meta, metaPath)
	if reason != "" {
		return metadataRepairResult{metadataRepairSkipped, reason}
	}
	if !plan.Changed() {
		return metadataRepairResult{metadataRepairUnchanged, "already matches Core-derived metadata"}
	}
	return metadataRepairResult{metadataRepairRepaired, "would update " + metadataRepairDiff(plan)}
}

func applyHistoricalMetadata(location taskx.TaskLocation, meta taskx.TaskMeta, metaPath string) metadataRepairResult {
	var err error
	meta, err = taskx.ReadTaskMeta(metaPath)
	if err != nil || meta.ID != location.ID {
		if err == nil {
			err = fmt.Errorf("metadata id is %q", meta.ID)
		}
		return metadataRepairResult{metadataRepairSkipped, "re-read identity evidence under Task lock: " + err.Error()}
	}
	plan, reason := planHistoricalMetadataRepair(location, meta, metaPath)
	if reason != "" {
		return metadataRepairResult{metadataRepairSkipped, reason}
	}
	if !plan.Changed() {
		return metadataRepairResult{metadataRepairUnchanged, "already matches Core-derived metadata"}
	}
	backup, err := writeMetadataRepairBackup(location.RuntimeDir, plan.Original())
	if err != nil {
		return metadataRepairResult{metadataRepairSkipped, "write pre-repair backup: " + err.Error()}
	}
	if err := plan.Apply(); err != nil {
		return metadataRepairResult{metadataRepairSkipped, "apply metadata repair after backup " + backup + ": " + err.Error()}
	}
	return metadataRepairResult{metadataRepairRepaired, "updated " + metadataRepairDiff(plan) + "; backup=" + backup}
}

func metadataRepairDiff(plan taskx.WorkflowMetadataRepairPlan) string {
	parts := make([]string, 0, len(plan.Fields))
	for _, field := range plan.Fields {
		switch field {
		case "status":
			parts = append(parts, fmt.Sprintf("status=%q→%q", plan.Before.Status, plan.After.Status))
		case "delivery":
			parts = append(parts, fmt.Sprintf("delivery=%q→%q", plan.Before.Delivery, plan.After.Delivery))
		case "worktree":
			parts = append(parts, fmt.Sprintf("worktree=%q→%q", plan.Before.Worktree, plan.After.Worktree))
		case "workspace_kind":
			parts = append(parts, fmt.Sprintf("workspace_kind=%q→%q", plan.Before.WorkspaceKind, plan.After.WorkspaceKind))
		}
	}
	return strings.Join(parts, ", ")
}

func planHistoricalMetadataRepair(location taskx.TaskLocation, meta taskx.TaskMeta, metaPath string) (taskx.WorkflowMetadataRepairPlan, string) {
	state, err := workflow.LoadFromDirectory(location.RuntimeDir, workflow.TaskID(location.ID))
	if err != nil {
		return taskx.WorkflowMetadataRepairPlan{}, "Core state unavailable: " + err.Error()
	}
	if state.WriteLease != nil || hasActiveRepairAttempt(state) || hasActiveRepairWorkItem(state) {
		return taskx.WorkflowMetadataRepairPlan{}, "Task has active execution evidence"
	}
	summary := workflow.DeriveSummary(state)
	if summary.Status != workflow.TaskDone && summary.Status != workflow.TaskCancelled {
		return taskx.WorkflowMetadataRepairPlan{}, "Core state is not terminal: " + string(summary.Status)
	}
	unassign, reason := historicalRepairUnassignment(meta, summary)
	if reason != "" {
		return taskx.WorkflowMetadataRepairPlan{}, reason
	}
	plan, err := taskx.PlanWorkflowMetadataRepair(metaPath, state, unassign)
	if err != nil {
		return taskx.WorkflowMetadataRepairPlan{}, "derive metadata repair: " + err.Error()
	}
	return plan, ""
}

func hasActiveRepairAttempt(state workflow.RuntimeState) bool {
	for _, attempt := range state.Attempts {
		if attempt.State == workflow.AttemptCreated || attempt.State == workflow.AttemptRunning || attempt.State == workflow.AttemptPaused {
			return true
		}
	}
	return false
}

func hasActiveRepairWorkItem(state workflow.RuntimeState) bool {
	for _, item := range state.WorkItems {
		if item.State == workflow.WorkItemLeased || item.State == workflow.WorkItemRunning {
			return true
		}
	}
	return false
}

func historicalRepairUnassignment(meta taskx.TaskMeta, summary workflow.TaskSummary) (bool, string) {
	switch taskx.ResolvedWorkspaceKind(meta) {
	case "primary", "unassigned":
		return false, ""
	case "isolated":
		if summary.Delivery != workflow.DeliveryMerged && summary.Delivery != workflow.DeliveryDiscarded {
			return false, "isolated Task has no terminal delivery evidence"
		}
		if strings.TrimSpace(meta.Worktree) == "" {
			return false, "isolated Task has no historical worktree evidence"
		}
		if gitx.WorktreeRegistered(meta.Worktree) || fsx.Exists(meta.Worktree) {
			return false, "isolated worktree still exists or is registered"
		}
		primary, err := gitx.PrimaryWorktree()
		if err != nil {
			return false, "resolve primary worktree for branch evidence: " + err.Error()
		}
		branchExists, err := branchExistsAt(primary, meta.Branch)
		if err != nil {
			return false, "inspect historical Task branch: " + err.Error()
		}
		if branchExists {
			return false, "historical Task branch still exists"
		}
		return true, ""
	default:
		return false, "workspace binding is unknown"
	}
}

func writeMetadataRepairBackup(runtimeDir string, original []byte) (string, error) {
	dir := filepath.Join(runtimeDir, "artifacts", "metadata-repair")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z") + ".toml"
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	_, writeErr := file.Write(original)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return "", err
	}
	return path, nil
}

func metadataRepairLocation(location taskx.TaskLocation) string {
	if location.Archived {
		return "archived"
	}
	if filepath.Base(filepath.Clean(location.RuntimeDir)) == location.ID && strings.Contains(filepath.ToSlash(location.RuntimeDir), "/.ai/tasks/") {
		return "legacy"
	}
	return "active"
}
