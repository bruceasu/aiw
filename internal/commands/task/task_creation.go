package task

import "aiw/internal/task"

// authorizeTaskCreation is retained as a CLI-local spelling while the Task
// module owns the creation policy and implementation.
func authorizeTaskCreation(id string, allowUnrelatedDirty bool) error {
	return task.AuthorizeTaskCreation(id, allowUnrelatedDirty)
}
