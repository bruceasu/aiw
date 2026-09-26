package cli

import (
	"aiw/internal/task"
	"aiw/internal/workflow"
)

// TaskAdapter is the only application-specific seam required by Workflow CLI.
// The standalone workflow program owns command orchestration; the root Task
// package supplies these operations without exposing its implementation.
type TaskAdapter interface {
	SafeID(string) bool
	ResolveWorkspaceKind(task.TaskMeta) string
	VerifiedTaskWorktree(task.TaskMeta) bool
	PrepareAutomatedWorkspace(*workflow.Store, string, task.TaskMeta) error
	PreflightSupervisedGitWorkspace(task.TaskMeta) ([]string, error)
	SupervisedWorkItemInstruction(string, workflow.WorkItemID, []string) (string, error)
	RunTaskAgentWithEnvironment([]string, []string) error
	AddTaskWorktree(string) error
	CreateTaskSession(string, string) error
	LocalMergeDelivery(string, task.TaskMeta, *workflow.Store, string) error
	PreflightForceCloseDelivery(task.TaskMeta, workflow.DeliveryState) error
	RunFocusedTest(task.TaskMeta, workflow.RuntimeState, *workflow.Store, workflow.AttemptID) (task.FocusedTestResult, error)
	RepairMetadata([]string) error
}
