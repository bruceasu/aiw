package workflowadapter

import (
	"strings"

	task "aiw/internal/task"
	workflowcore "aiw/internal/workflow"
)

// WorkflowRuntimeFromMeta initializes only a missing local Workflow Core
// projection from durable task.toml compatibility metadata. Existing Core
// records remain authoritative; metadata is never used to overwrite their
// status, delivery, process ownership, Attempt, or write lease.
func WorkflowRuntimeFromMeta(meta task.TaskMeta) workflowcore.RuntimeState {
	return workflowcore.NewCompatibleRuntime(
		workflowcore.TaskReference{
			ID:        workflowcore.TaskID(meta.ID),
			Workspace: meta.Worktree,
			Kind:      workflowWorkspaceState(meta.WorkspaceKind),
		},
		workflowPlanningState(meta.Status),
		workflowDeliveryState(meta.Delivery),
	)
}

func workflowPlanningState(status string) workflowcore.PlanningState {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "TODO", "DRAFT", "":
		return workflowcore.PlanningDraft
	case "NEEDS_DECISION":
		return workflowcore.PlanningNeedsDecision
	default:
		return workflowcore.PlanningReady
	}
}

func workflowWorkspaceState(kind string) workflowcore.WorkspaceState {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "primary":
		return workflowcore.WorkspacePrimary
	case "isolated":
		return workflowcore.WorkspaceIsolated
	case "unassigned", "":
		return workflowcore.WorkspaceUnassigned
	default:
		return workflowcore.WorkspaceUnknown
	}
}

func workflowDeliveryState(delivery string) workflowcore.DeliveryState {
	switch strings.ToLower(strings.TrimSpace(delivery)) {
	case "pending":
		return workflowcore.DeliveryPending
	case "merged":
		return workflowcore.DeliveryMerged
	case "discarded":
		return workflowcore.DeliveryDiscarded
	default:
		return workflowcore.DeliveryUnmanaged
	}
}
