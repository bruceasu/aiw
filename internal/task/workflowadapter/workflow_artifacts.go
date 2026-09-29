package workflowadapter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/gitx"
	task "aiw/internal/task"
	workflowcore "aiw/internal/workflow"
)

// SyncWorkflowChecklist adapts the Task checklist in its bound workspace.
func SyncWorkflowChecklist(id string, store *workflowcore.Store) (workflowcore.RuntimeState, error) {
	path, err := WorkflowChecklistPath(id)
	if err != nil {
		return workflowcore.RuntimeState{}, err
	}
	return SyncWorkflowChecklistAtPath(id, store, path)
}

// SyncWorkflowChecklistAtPath adapts a checklist at a caller-resolved Task
// artifact path. Archive preflight uses the discovered change directory so a
// cleaned unassigned Task cannot silently read an unrelated primary checkout.
func SyncWorkflowChecklistAtPath(id string, store *workflowcore.Store, path string) (workflowcore.RuntimeState, error) {
	items, err := task.ReadWorkflowChecklist(path)
	if err != nil {
		var projectionErr *task.ChecklistProjectionError
		if errors.As(err, &projectionErr) {
			_, _ = store.OpenChecklistReconciliationGate(workflowcore.TaskID(id), "", projectionErr.Code, projectionErr.Error())
		}
		return workflowcore.RuntimeState{}, err
	}
	if filepath.Base(path) == id+".md" && filepath.Base(filepath.Dir(path)) == "features" {
		readiness, err := task.ReadFeatureDesignReadiness(path)
		if err != nil { return workflowcore.RuntimeState{}, err }
		if readiness == "BLOCKED" {
			return workflowcore.RuntimeState{}, fmt.Errorf("Task %s Feature Design is BLOCKED; resolve Design Readiness before mapping work items", id)
		}
	}
	candidates := make([]workflowcore.ChecklistCandidate, len(items))
	for index, item := range items {
		candidates[index] = workflowcore.ChecklistCandidate{Item: item.Number, Title: item.Title, Completed: item.Completed, DependsOn: item.DependsOn}
	}
	return store.SyncChecklist(workflowcore.TaskID(id), candidates, task.ChecklistFingerprint(items))
}

// WorkflowChecklistPath selects authored artifacts in the bound Task workspace;
// isolated Tasks read their own worktree rather than the parent checkout.
func WorkflowChecklistPath(id string) (string, error) {
	meta, err := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	if err != nil {
		return "", err
	}
	if task.ResolvedWorkspaceKind(meta) != "isolated" {
		if _, err := os.Stat(task.FeatureDesignPath(id)); err == nil {
			return task.FeatureDesignPath(id), nil
		} else if !os.IsNotExist(err) { return "", err }
		return filepath.Join(task.TaskDir(id), "tasks.md"), nil
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
	fdPath := filepath.Join(worktree, task.FeatureDesignPath(id))
	if _, err := os.Stat(fdPath); err == nil {
		return fdPath, nil
	} else if !os.IsNotExist(err) { return "", err }
	if _, err := os.Stat(task.FeatureDesignPath(id)); err == nil {
		return "", fmt.Errorf("Task %s FD is not in the isolated worktree; commit the planning artifact first", id)
	} else if !os.IsNotExist(err) { return "", err }
	return filepath.Join(worktree, task.TaskDir(id), "tasks.md"), nil
}

// ProjectAcceptedWorkItem marks an accepted item in the isolated checklist and
// records a reconciliation Gate or projection repair when the write fails.
func ProjectAcceptedWorkItem(id string, store *workflowcore.Store, state workflowcore.RuntimeState, workItemID workflowcore.WorkItemID) error {
	var item *workflowcore.WorkItem
	for index := range state.WorkItems {
		if state.WorkItems[index].ID == workItemID {
			item = &state.WorkItems[index]
			break
		}
	}
	if item == nil || item.State != workflowcore.WorkItemCompleted || item.Checklist.Item == "" {
		return fmt.Errorf("completed work item %s has no checklist projection", workItemID)
	}
	meta, err := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	if err == nil && task.ResolvedWorkspaceKind(meta) == "primary" {
		primary, _, primaryErr := gitx.IsPrimaryWorktree()
		if primaryErr != nil { err = primaryErr } else if !primary || meta.Worktree != "." {
			err = fmt.Errorf("accepted Work Item projection requires the bound primary worktree")
		}
	} else if err == nil && task.ResolvedWorkspaceKind(meta) != "isolated" {
		err = fmt.Errorf("accepted Work Item projection requires a bound primary or isolated worktree")
	}
	path := ""
	if err == nil {
		path, err = WorkflowChecklistPath(id)
	}
	if err == nil {
		err = task.ProjectWorkflowChecklistCompletion(path, item.Checklist.Item)
	}
	if err == nil {
		return nil
	}
	var projectionErr *task.ChecklistProjectionError
	if errors.As(err, &projectionErr) {
		_, gateErr := store.OpenChecklistReconciliationGate(workflowcore.TaskID(id), workItemID, projectionErr.Code, projectionErr.Error())
		if gateErr != nil {
			return fmt.Errorf("project accepted Work Item: %w; record reconciliation Gate: %v", err, gateErr)
		}
		return err
	}
	_, repairErr := store.EnqueueProjectionRepair(workflowcore.TaskID(id), workflowcore.ProjectionRepair{
		EventSequence: state.LastEventSequence,
		Target:        "tasks.md/" + string(workItemID),
		ErrorClass:    "checklist-projection-failed",
		Recommended:   fmt.Sprintf("aiw wf repair %s", id),
	})
	if repairErr != nil {
		return fmt.Errorf("project accepted Work Item: %w; enqueue projection repair: %v", err, repairErr)
	}
	return err
}

// RetryChecklistProjection retries only the recorded accepted Work Item writeback.
// A successful retry closes that exact repair; failures leave it pending.
func RetryChecklistProjection(id string, store *workflowcore.Store, repair workflowcore.ProjectionRepair) (workflowcore.RuntimeState, error) {
	if !strings.HasPrefix(repair.Target, "tasks.md/") {
		return workflowcore.RuntimeState{}, fmt.Errorf("unsupported projection repair target %q", repair.Target)
	}
	state, err := store.Load(workflowcore.TaskID(id))
	if err != nil {
		return workflowcore.RuntimeState{}, err
	}
	pending := false
	for _, existing := range state.Automation.ProjectionRepairs {
		if existing.EventSequence == repair.EventSequence && existing.Target == repair.Target && existing.ResolvedAt == "" {
			pending = true
			break
		}
	}
	if !pending {
		return workflowcore.RuntimeState{}, fmt.Errorf("projection repair is no longer pending: %s", repair.Target)
	}
	workItemID := workflowcore.WorkItemID(strings.TrimPrefix(repair.Target, "tasks.md/"))
	if err := ProjectAcceptedWorkItem(id, store, state, workItemID); err != nil {
		return workflowcore.RuntimeState{}, err
	}
	return store.ResolveProjectionRepair(workflowcore.TaskID(id), repair.EventSequence, repair.Target)
}

// RepairWorkflowChecklist repairs runtime projection from the Task's authored checklist.
func RepairWorkflowChecklist(id string) (workflowcore.RuntimeState, error) {
	meta, err := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	if err != nil {
		return workflowcore.RuntimeState{}, err
	}
	store := workflowcore.NewStore("")
	if _, err := store.EnsureCompatible(WorkflowRuntimeFromMeta(meta)); err != nil {
		return workflowcore.RuntimeState{}, err
	}
	path, err := WorkflowChecklistPath(id)
	if err != nil {
		return workflowcore.RuntimeState{}, err
	}
	items, err := task.ReadWorkflowChecklist(path)
	if err != nil {
		return workflowcore.RuntimeState{}, err
	}
	candidates := make([]workflowcore.ChecklistCandidate, len(items))
	for index, item := range items {
		candidates[index] = workflowcore.ChecklistCandidate{Item: item.Number, Title: item.Title, Completed: item.Completed, DependsOn: item.DependsOn}
	}
	return store.RepairChecklist(workflowcore.TaskID(id), candidates)
}
