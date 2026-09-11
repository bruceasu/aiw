package taskx

import (
	"fmt"

	"aiw/internal/workflow"
)

// WorkflowFieldOwner makes the durable/runtime boundary explicit for adapters.
// It is intentionally small: extending it requires deciding who may write the
// field rather than adding another bidirectional synchronization path.
type WorkflowFieldOwner string

const (
	OwnerTaskMetadata WorkflowFieldOwner = "task-metadata"
	OwnerOpenSpec     WorkflowFieldOwner = "openspec"
	OwnerWorkflowCore WorkflowFieldOwner = "workflow-core"
)

// workflowFieldOwnership identifies the single authoritative writer for the
// managed Task artifacts. OpenSpec checklist prose remains OpenSpec-owned;
// Workflow Core keeps derived progress in runtime state.
var workflowFieldOwnership = map[string]WorkflowFieldOwner{
	"task.toml.id":             OwnerTaskMetadata,
	"task.toml.branch":         OwnerTaskMetadata,
	"task.toml.parent_branch":  OwnerTaskMetadata,
	"task.toml.worktree":       OwnerTaskMetadata,
	"task.toml.workspace_kind": OwnerTaskMetadata,
	"task.toml.status":         OwnerTaskMetadata,
	"task.toml.delivery":       OwnerTaskMetadata,
	"tasks.md.prose":           OwnerOpenSpec,
	"tasks.md.checklist":       OwnerOpenSpec,
	"runtime.state":            OwnerWorkflowCore,
	"runtime.events":           OwnerWorkflowCore,
	"runtime.write_lease":      OwnerWorkflowCore,
}

// WorkflowOwner returns the authoritative writer for an artifact field.
func WorkflowOwner(field string) (WorkflowFieldOwner, bool) {
	owner, ok := workflowFieldOwnership[field]
	return owner, ok
}

// ProjectWorkflowSummary is a legacy pure mapper retained for compatibility.
// Managed commands do not persist its result: Workflow Core runtime storage is
// the authoritative durable source for execution-derived state.
func ProjectWorkflowSummary(meta TaskMeta, state workflow.RuntimeState) (TaskMeta, error) {
	if meta.ID == "" || state.Task.ID != workflow.TaskID(meta.ID) {
		return TaskMeta{}, fmt.Errorf("workflow projection Task ID mismatch")
	}
	if err := workflow.ValidateRuntimeState(state); err != nil {
		return TaskMeta{}, err
	}
	result := meta
	summary := workflow.DeriveSummary(state)
	result.Status = string(summary.Status)
	result.Delivery = string(summary.Delivery)
	result.Updated = Today()
	return result, nil
}
