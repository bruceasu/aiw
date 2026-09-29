package task

import "aiw/internal/task"

// CreateTaskFromIssue creates a native Task and FD without an OpenSpec change.
func CreateTaskFromIssue(id, issueID string, allowUnrelatedDirty bool) error {
	return task.CreateIssueTask(id, issueID, allowUnrelatedDirty)
}

// CreateTaskWithOpenSpec creates the Task through the verified OpenSpec
// backend. OpenSpec owns the change artifacts; AIW only reconciles its Task
// metadata after the delegated command returns.
func CreateTaskWithOpenSpec(id string, allowUnrelatedDirty bool) error {
	args := []string{id, "--backend", "openspec"}
	if allowUnrelatedDirty {
		args = append(args, "--allow-unrelated-dirty")
	}
	return DispatchTopLevel("new", args)
}

// EnsureChecklistMapping exposes the workflow projection seam used after a
// Requirement promotion without exposing task internals to req.
func EnsureChecklistMapping(id string) error {
	return ensureChecklistMapping(id)
}
