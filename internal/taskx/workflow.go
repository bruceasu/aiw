package taskx

import (
	"strings"

	"aiw/internal/workflow"
)

// WorkflowRuntimeFromMeta initializes only a missing local Workflow Core
// projection from durable task.toml compatibility metadata. Existing Core
// records remain authoritative; metadata is never used to overwrite their
// status, delivery, process ownership, Attempt, or write lease.
func WorkflowRuntimeFromMeta(meta TaskMeta) workflow.RuntimeState {
	return workflow.NewCompatibleRuntime(
		workflow.TaskReference{
			ID:        workflow.TaskID(meta.ID),
			Workspace: meta.Worktree,
			Kind:      workflowWorkspaceState(meta.WorkspaceKind),
		},
		workflowPlanningState(meta.Status),
		workflowDeliveryState(meta.Delivery),
	)
}

func workflowPlanningState(status string) workflow.PlanningState {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "TODO", "DRAFT", "":
		return workflow.PlanningDraft
	case "NEEDS_DECISION":
		return workflow.PlanningNeedsDecision
	default:
		return workflow.PlanningReady
	}
}

func workflowWorkspaceState(kind string) workflow.WorkspaceState {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "primary":
		return workflow.WorkspacePrimary
	case "isolated":
		return workflow.WorkspaceIsolated
	case "unassigned", "":
		return workflow.WorkspaceUnassigned
	default:
		return workflow.WorkspaceUnknown
	}
}

func workflowDeliveryState(delivery string) workflow.DeliveryState {
	switch strings.ToLower(strings.TrimSpace(delivery)) {
	case "pending":
		return workflow.DeliveryPending
	case "merged":
		return workflow.DeliveryMerged
	case "discarded":
		return workflow.DeliveryDiscarded
	default:
		return workflow.DeliveryUnmanaged
	}
}
