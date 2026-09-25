package task

import "aiw/internal/taskx"

// preflightSupervisedGitWorkspace delegates binding checks to the Task adapter.
func preflightSupervisedGitWorkspace(meta taskx.TaskMeta) ([]string, error) {
	return taskx.PreflightSupervisedGitWorkspace(meta)
}
