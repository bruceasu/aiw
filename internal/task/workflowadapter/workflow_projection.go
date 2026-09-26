package workflowadapter

import (
	"fmt"
	"os"

	task "aiw/internal/task"
	workflowcore "aiw/internal/workflow"
)

// WorkflowMetadataRepairPlan is a Core-derived, metadata-only repair. The
// unexported byte slices preserve the original TOML representation, including
// fields that this package does not understand.
type WorkflowMetadataRepairPlan struct {
	Path     string
	Before   task.TaskMeta
	After    task.TaskMeta
	Fields   []string
	original []byte
	repaired []byte
}

// Changed reports whether applying the plan would change task.toml.
func (p WorkflowMetadataRepairPlan) Changed() bool { return len(p.Fields) > 0 }

// Original returns a copy suitable for a repair backup.
func (p WorkflowMetadataRepairPlan) Original() []byte { return append([]byte(nil), p.original...) }

// Apply atomically writes the planned metadata change. Callers are responsible
// for holding the Task's Workflow Core lock and creating any requested backup.
func (p WorkflowMetadataRepairPlan) Apply() error {
	if !p.Changed() {
		return nil
	}
	return task.WriteTaskMetaAtomically(p.Path, p.repaired)
}

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
	"task.toml.status":         OwnerWorkflowCore,
	"task.toml.delivery":       OwnerWorkflowCore,
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

// ProjectWorkflowSummary maps a Workflow Core summary into the compatible
// task.toml fields. It is pure so callers cannot use a metadata snapshot to
// overwrite an existing Core record.
func ProjectWorkflowSummary(meta task.TaskMeta, state workflowcore.RuntimeState) (task.TaskMeta, error) {
	if meta.ID == "" || state.Task.ID != workflowcore.TaskID(meta.ID) {
		return task.TaskMeta{}, fmt.Errorf("workflow projection Task ID mismatch")
	}
	if err := workflowcore.ValidateRuntimeState(state); err != nil {
		return task.TaskMeta{}, err
	}
	result := meta
	summary := workflowcore.DeriveSummary(state)
	result.Status = string(summary.Status)
	result.Delivery = string(summary.Delivery)
	result.Updated = task.Today()
	return result, nil
}

// WriteWorkflowSummary atomically persists only the Core-derived task.toml
// fields. It preserves unknown fields and avoids rewriting an already current
// snapshot, while leaving Workflow Core as the authoritative state source.
func WriteWorkflowSummary(path string, state workflowcore.RuntimeState) (task.TaskMeta, error) {
	meta, err := task.ReadTaskMeta(path)
	if err != nil {
		return task.TaskMeta{}, err
	}
	projected, err := ProjectWorkflowSummary(meta, state)
	if err != nil {
		return task.TaskMeta{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return task.TaskMeta{}, err
	}
	content, statusChanged := task.ReplaceTaskMetaField(content, "status", projected.Status)
	content, deliveryChanged := task.ReplaceTaskMetaField(content, "delivery", projected.Delivery)
	if statusChanged || deliveryChanged {
		if err := task.WriteTaskMetaAtomically(path, content); err != nil {
			return task.TaskMeta{}, err
		}
	}
	return task.ReadTaskMeta(path)
}

// PlanWorkflowMetadataRepair derives a repair only from an existing Workflow
// Core state. It never reads a task.toml snapshot back into Core. When
// unassignWorkspace is true, the plan also clears the current isolated binding
// while preserving historical branch, parent branch, Session, and unknown TOML
// fields.
func PlanWorkflowMetadataRepair(path string, state workflowcore.RuntimeState, unassignWorkspace bool) (WorkflowMetadataRepairPlan, error) {
	meta, err := task.ReadTaskMeta(path)
	if err != nil {
		return WorkflowMetadataRepairPlan{}, err
	}
	projected, err := ProjectWorkflowSummary(meta, state)
	if err != nil {
		return WorkflowMetadataRepairPlan{}, err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return WorkflowMetadataRepairPlan{}, err
	}
	repaired := original
	fields := make([]string, 0, 4)
	var changed bool
	if repaired, changed = task.ReplaceTaskMetaField(repaired, "status", projected.Status); changed {
		fields = append(fields, "status")
	}
	if repaired, changed = task.ReplaceTaskMetaField(repaired, "delivery", projected.Delivery); changed {
		fields = append(fields, "delivery")
	}
	if unassignWorkspace {
		if repaired, changed = task.ReplaceTaskMetaField(repaired, "worktree", ""); changed {
			fields = append(fields, "worktree")
		}
		if repaired, changed = task.ReplaceTaskMetaField(repaired, "workspace_kind", "unassigned"); changed {
			fields = append(fields, "workspace_kind")
		}
		projected.Worktree, projected.WorkspaceKind = "", "unassigned"
	}
	return WorkflowMetadataRepairPlan{
		Path: path, Before: meta, After: projected, Fields: fields,
		original: original, repaired: repaired,
	}, nil
}

// UnassignWorkspace records that an isolated worktree has been removed. It
// changes only the current binding fields, preserving the historical branch,
// parent branch, Session, delivery evidence, and unknown TOML fields for
// recovery and archive eligibility checks.
func UnassignWorkspace(path, id string) (task.TaskMeta, error) {
	meta, err := task.ReadTaskMeta(path)
	if err != nil {
		return task.TaskMeta{}, err
	}
	if meta.ID != id {
		return task.TaskMeta{}, fmt.Errorf("workspace unassignment Task ID mismatch: expected %s, got %s", id, meta.ID)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return task.TaskMeta{}, err
	}
	content, worktreeChanged := task.ReplaceTaskMetaField(content, "worktree", "")
	content, kindChanged := task.ReplaceTaskMetaField(content, "workspace_kind", "unassigned")
	if worktreeChanged || kindChanged {
		if err := task.WriteTaskMetaAtomically(path, content); err != nil {
			return task.TaskMeta{}, err
		}
	}
	return task.ReadTaskMeta(path)
}
