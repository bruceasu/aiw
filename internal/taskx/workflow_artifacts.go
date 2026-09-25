package taskx

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"aiw/internal/gitx"
	"aiw/internal/workflow"
)

// SyncWorkflowChecklist adapts the Task checklist in its bound workspace.
func SyncWorkflowChecklist(id string, store *workflow.Store) (workflow.RuntimeState, error) {
	path, err := WorkflowChecklistPath(id)
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	return SyncWorkflowChecklistAtPath(id, store, path)
}

// SyncWorkflowChecklistAtPath adapts a checklist at a caller-resolved Task
// artifact path. Archive preflight uses the discovered change directory so a
// cleaned unassigned Task cannot silently read an unrelated primary checkout.
func SyncWorkflowChecklistAtPath(id string, store *workflow.Store, path string) (workflow.RuntimeState, error) {
	items, err := ReadWorkflowChecklist(path)
	if err != nil {
		var projectionErr *ChecklistProjectionError
		if errors.As(err, &projectionErr) {
			_, _ = store.OpenChecklistReconciliationGate(workflow.TaskID(id), "", projectionErr.Code, projectionErr.Error())
		}
		return workflow.RuntimeState{}, err
	}
	candidates := make([]workflow.ChecklistCandidate, len(items))
	for index, item := range items {
		candidates[index] = workflow.ChecklistCandidate{Item: item.Number, Title: item.Title, Completed: item.Completed, DependsOn: item.DependsOn}
	}
	return store.SyncChecklist(workflow.TaskID(id), candidates, ChecklistFingerprint(items))
}

// WorkflowChecklistPath selects authored artifacts in the bound Task workspace;
// isolated Tasks read their own worktree rather than the parent checkout.
func WorkflowChecklistPath(id string) (string, error) {
	meta, err := ReadTaskMeta(ResolveTaskMetaPath(id))
	if err != nil {
		return "", err
	}
	if ResolvedWorkspaceKind(meta) != "isolated" {
		return filepath.Join(TaskDir(id), "tasks.md"), nil
	}
	worktree := strings.TrimSpace(meta.Worktree)
	if worktree == "" {
		return "", fmt.Errorf("Task %s has no isolated worktree", id)
	}
	if !filepath.IsAbs(worktree) {
		root, err := gitx.PrimaryWorktree()
		if err != nil {
			return "", err
		}
		worktree = filepath.Join(root, filepath.FromSlash(worktree))
	}
	return filepath.Join(worktree, TaskDir(id), "tasks.md"), nil
}

// ProjectAcceptedWorkItem marks an accepted item in the isolated checklist and
// records a reconciliation Gate or projection repair when the write fails.
func ProjectAcceptedWorkItem(id string, store *workflow.Store, state workflow.RuntimeState, workItemID workflow.WorkItemID) error {
	var item *workflow.WorkItem
	for index := range state.WorkItems {
		if state.WorkItems[index].ID == workItemID {
			item = &state.WorkItems[index]
			break
		}
	}
	if item == nil || item.State != workflow.WorkItemCompleted || item.Checklist.Item == "" {
		return fmt.Errorf("completed work item %s has no checklist projection", workItemID)
	}
	meta, err := ReadTaskMeta(ResolveTaskMetaPath(id))
	if err == nil && ResolvedWorkspaceKind(meta) != "isolated" {
		err = fmt.Errorf("accepted Work Item projection requires an isolated Task worktree")
	}
	path := ""
	if err == nil {
		path, err = WorkflowChecklistPath(id)
	}
	if err == nil {
		err = ProjectWorkflowChecklistCompletion(path, item.Checklist.Item)
	}
	if err == nil {
		return nil
	}
	var projectionErr *ChecklistProjectionError
	if errors.As(err, &projectionErr) {
		_, gateErr := store.OpenChecklistReconciliationGate(workflow.TaskID(id), workItemID, projectionErr.Code, projectionErr.Error())
		if gateErr != nil {
			return fmt.Errorf("project accepted Work Item: %w; record reconciliation Gate: %v", err, gateErr)
		}
		return err
	}
	_, repairErr := store.EnqueueProjectionRepair(workflow.TaskID(id), workflow.ProjectionRepair{
		EventSequence: state.LastEventSequence,
		Target:        "tasks.md/" + string(workItemID),
		ErrorClass:    "checklist-projection-failed",
		Recommended:   fmt.Sprintf("aiw task workflow repair %s", id),
	})
	if repairErr != nil {
		return fmt.Errorf("project accepted Work Item: %w; enqueue projection repair: %v", err, repairErr)
	}
	return err
}

// RepairWorkflowChecklist repairs runtime projection from the Task's authored checklist.
func RepairWorkflowChecklist(id string) (workflow.RuntimeState, error) {
	meta, err := ReadTaskMeta(ResolveTaskMetaPath(id))
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(WorkflowRuntimeFromMeta(meta)); err != nil {
		return workflow.RuntimeState{}, err
	}
	path, err := WorkflowChecklistPath(id)
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	items, err := ReadWorkflowChecklist(path)
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	candidates := make([]workflow.ChecklistCandidate, len(items))
	for index, item := range items {
		candidates[index] = workflow.ChecklistCandidate{Item: item.Number, Title: item.Title, Completed: item.Completed, DependsOn: item.DependsOn}
	}
	return store.RepairChecklist(workflow.TaskID(id), candidates)
}
