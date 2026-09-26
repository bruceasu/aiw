package task

import "aiw/internal/task"

// preflightSupervisedGitWorkspace delegates binding checks to the Task adapter.
func preflightSupervisedGitWorkspace(meta task.TaskMeta) ([]string, error) {
	return task.PreflightSupervisedGitWorkspace(meta)
}
