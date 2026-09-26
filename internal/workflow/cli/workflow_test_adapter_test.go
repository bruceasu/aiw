package cli

import (
	"fmt"

	"aiw/internal/gitx"
	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow"
)

// These helpers keep Workflow CLI regression tests focused on behavior.
// Production code uses Task seams and the standalone Workflow program directly.
type testWorkflowTaskAdapter struct{}

func testWorkflowAdapter() TaskAdapter { return testWorkflowTaskAdapter{} }

func (testWorkflowTaskAdapter) SafeID(id string) bool { return taskworkflow.SafeID(id) }
func (testWorkflowTaskAdapter) ResolveWorkspaceKind(meta task.TaskMeta) string { return task.ResolvedWorkspaceKind(meta) }
func (testWorkflowTaskAdapter) VerifiedTaskWorktree(meta task.TaskMeta) bool { return taskworkflow.VerifiedTaskWorktree(meta) }
func (a testWorkflowTaskAdapter) PrepareAutomatedWorkspace(_ *workflow.Store, id string, meta task.TaskMeta) error {
	kind := a.ResolveWorkspaceKind(meta)
	if kind == "isolated" {
		if !a.VerifiedTaskWorktree(meta) { return fmt.Errorf("Task %s has an invalid isolated worktree binding", id) }
		return nil
	}
	if kind != "primary" { return fmt.Errorf("Task %s workspace is %s; repair or explicitly bind it before automated execution", id, kind) }
	isPrimary, primaryPath, err := gitx.IsPrimaryWorktree()
	if err != nil { return err }
	if !isPrimary { return fmt.Errorf("automatic isolation must start from the primary worktree: %s", primaryPath) }
	return taskworkflow.AddTaskWorktree(id)
}
func (testWorkflowTaskAdapter) PreflightSupervisedGitWorkspace(meta task.TaskMeta) ([]string, error) { return task.PreflightSupervisedGitWorkspace(meta) }
func (testWorkflowTaskAdapter) SupervisedWorkItemInstruction(id string, workItemID workflow.WorkItemID, environment []string) (string, error) { return taskworkflow.SupervisedWorkItemInstruction(id, workItemID, environment) }
func (testWorkflowTaskAdapter) RunTaskAgentWithEnvironment(args, environment []string) error { return taskworkflow.RunTaskAgentWithEnvironment(args, environment) }
func (testWorkflowTaskAdapter) AddTaskWorktree(id string) error { return taskworkflow.AddTaskWorktree(id) }
func (testWorkflowTaskAdapter) CreateTaskSession(id, worktree string) error { return taskworkflow.CreateTaskSession(id, worktree) }
func (testWorkflowTaskAdapter) LocalMergeDelivery(id string, meta task.TaskMeta, store *workflow.Store, message string) error { return taskworkflow.LocalMergeDelivery(id, meta, store, message) }
func (testWorkflowTaskAdapter) PreflightForceCloseDelivery(meta task.TaskMeta, delivery workflow.DeliveryState) error { return taskworkflow.PreflightForceCloseDelivery(meta, delivery) }
func (testWorkflowTaskAdapter) RunFocusedTest(meta task.TaskMeta, state workflow.RuntimeState, store *workflow.Store, attemptID workflow.AttemptID) (task.FocusedTestResult, error) {
	return taskworkflow.RunFocusedTest(meta, state, store, attemptID)
}
func (testWorkflowTaskAdapter) RepairMetadata(args []string) error {
	return taskworkflow.RepairMetadata(args)
}

func runWorkflowCommand(args []string) error {
	return RunWorkflowCommand(testWorkflowAdapter(), args)
}

func printWorkflowHelp() { PrintWorkflowHelp() }

func runWorkflowSupervisor(args []string) error {
	return RunWorkflowSupervisor(testWorkflowAdapter(), args)
}

func validateWorkflowArgs(op string, args []string) error {
	return ValidateWorkflowArgs(testWorkflowAdapter(), op, args)
}

func parseWorkflowRunArgs(args []string) (bool, bool, error) {
	return ParseWorkflowRunArgs(args)
}

func ensureAutomatedWorkspace(id string, meta task.TaskMeta, primary bool) (task.TaskMeta, error) {
	return EnsureAutomatedWorkspace(testWorkflowAdapter(), id, meta, primary)
}

func runWorkflow(id string, execute bool, supervised ...bool) error {
	return RunWorkflow(testWorkflowAdapter(), id, execute, supervised...)
}

func runWorkflowWithOverrides(id string, execute, supervised, primary bool, provider, model string) error {
	return RunWorkflowWithOverrides(testWorkflowAdapter(), id, execute, supervised, primary, provider, model)
}

func advanceWorkflow(id string, meta task.TaskMeta, store *workflow.Store, supervised bool, provider, model string) (workflow.RuntimeState, error) {
	return AdvanceWorkflow(testWorkflowAdapter(), id, meta, store, supervised, provider, model)
}

func recommendRoutingProfiles(workspace string) (map[string]string, error) {
	return RecommendRoutingProfiles(workspace)
}

func repairWorkflowState(id string) error {
	return RepairWorkflowState(testWorkflowAdapter(), id)
}
