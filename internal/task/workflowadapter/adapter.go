package workflowadapter

import (
	"fmt"

	"aiw/internal/gitx"
	"aiw/internal/task"
	"aiw/internal/workflow"
)

// Operations are the Task-owned operations required by Workflow. The
// implementation currently remains in the Task command package while the
// migration moves it behind this named Task-owned seam.
type Operations interface {
	SafeID(string) bool
	ResolveWorkspaceKind(task.TaskMeta) string
	VerifiedTaskWorktree(task.TaskMeta) bool
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

// Adapter connects the standalone Workflow CLI to Task-owned side effects.
type Adapter struct{ operations Operations }

func New(operations Operations) Adapter { return Adapter{operations: operations} }

// DefaultOperations is the concrete Task-owned adapter used by the standalone
// Workflow program. It deliberately has no dependency on the Task CLI package.
type DefaultOperations struct{}

func (DefaultOperations) SafeID(id string) bool { return SafeID(id) }
func (DefaultOperations) ResolveWorkspaceKind(meta task.TaskMeta) string { return ResolveWorkspaceKind(meta) }
func (DefaultOperations) VerifiedTaskWorktree(meta task.TaskMeta) bool { return VerifiedTaskWorktree(meta) }
func (DefaultOperations) PreflightSupervisedGitWorkspace(meta task.TaskMeta) ([]string, error) {
	return task.PreflightSupervisedGitWorkspace(meta)
}
func (DefaultOperations) SupervisedWorkItemInstruction(id string, workItemID workflow.WorkItemID, environment []string) (string, error) {
	return SupervisedWorkItemInstruction(id, workItemID, environment)
}
func (DefaultOperations) RunTaskAgentWithEnvironment(args, environment []string) error {
	return RunTaskAgentWithEnvironment(args, environment)
}
func (DefaultOperations) AddTaskWorktree(id string) error { return addTaskWorktree(id) }
func (DefaultOperations) CreateTaskSession(id, worktree string) error { return createTaskSession(id, worktree) }
func (DefaultOperations) LocalMergeDelivery(id string, meta task.TaskMeta, store *workflow.Store, message string) error {
	return LocalMergeDelivery(id, meta, store, message)
}
func (DefaultOperations) PreflightForceCloseDelivery(meta task.TaskMeta, delivery workflow.DeliveryState) error {
	return PreflightForceCloseDelivery(meta, delivery)
}
func (DefaultOperations) RunFocusedTest(meta task.TaskMeta, state workflow.RuntimeState, store *workflow.Store, attemptID workflow.AttemptID) (task.FocusedTestResult, error) {
	return RunFocusedTest(meta, state, store, attemptID)
}
func (DefaultOperations) RepairMetadata(args []string) error { return RepairMetadata(args) }

// ProjectWorkflowState persists the Task metadata projection after a Core
// transition. Workflow CLI and Task-owned side effects share this seam.
func ProjectWorkflowState(id string, state workflow.RuntimeState) error {
	if _, err := WriteWorkflowSummary(task.ResolveTaskMetaPath(id), state); err != nil {
		return fmt.Errorf("sync task metadata snapshot after Workflow Core update: %w", err)
	}
	return nil
}

// LocalMergeDelivery performs the recoverable local delivery protocol.
func LocalMergeDelivery(id string, meta task.TaskMeta, store *workflow.Store, message string) error {
	return localMergeDelivery(id, meta, store, message)
}

// PreflightForceCloseDelivery validates a force-close delivery choice before
// any cleanup or metadata mutation occurs.
func PreflightForceCloseDelivery(meta task.TaskMeta, delivery workflow.DeliveryState) error {
	return preflightForceCloseDelivery(meta, delivery)
}

// RunFocusedTest executes the selected immutable verification check through
// the Task-owned controlled runner.
func RunFocusedTest(meta task.TaskMeta, state workflow.RuntimeState, store *workflow.Store, attemptID workflow.AttemptID) (task.FocusedTestResult, error) {
	run, err := ResolveFocusedTestRun(meta, state, attemptID)
	if err != nil {
		return task.FocusedTestResult{}, err
	}
	process := FocusedTestProcessRunner(noNetworkFocusedTestProcessRunner{})
	if focusedTestNetworkEnforcementWaived(state, run) {
		process = waivedFocusedTestProcessRunner{}
	}
	execution, err := ExecuteFocusedTest(store, run, installedFocusedTestNetworkEnforcer{}, process)
	return task.FocusedTestResult{RuntimeState: execution.RuntimeState}, err
}

// RepairMetadata applies the guarded historical metadata repair operation.
func RepairMetadata(args []string) error {
	options, err := parseMetadataRepairOptions(args)
	if err != nil {
		return err
	}
	return repairHistoricalMetadata(options)
}

func (a Adapter) SafeID(id string) bool { return a.operations.SafeID(id) }

func (a Adapter) ResolveWorkspaceKind(meta task.TaskMeta) string {
	return a.operations.ResolveWorkspaceKind(meta)
}

func (a Adapter) VerifiedTaskWorktree(meta task.TaskMeta) bool {
	return a.operations.VerifiedTaskWorktree(meta)
}

func (a Adapter) PrepareAutomatedWorkspace(_ *workflow.Store, id string, meta task.TaskMeta) error {
	kind := a.ResolveWorkspaceKind(meta)
	if kind == "isolated" {
		if !a.VerifiedTaskWorktree(meta) {
			return fmt.Errorf("Task %s has an invalid isolated worktree binding", id)
		}
		return nil
	}
	if kind != "primary" {
		return fmt.Errorf("Task %s workspace is %s; repair or explicitly bind it before automated execution", id, kind)
	}
	isPrimary, primaryPath, err := gitx.IsPrimaryWorktree()
	if err != nil {
		return err
	}
	if !isPrimary {
		return fmt.Errorf("automatic isolation must start from the primary worktree: %s", primaryPath)
	}
	if err := a.operations.AddTaskWorktree(id); err != nil {
		return fmt.Errorf("create automatic worktree: %w", err)
	}
	return nil
}

func (a Adapter) PreflightSupervisedGitWorkspace(meta task.TaskMeta) ([]string, error) {
	return a.operations.PreflightSupervisedGitWorkspace(meta)
}

func (a Adapter) SupervisedWorkItemInstruction(id string, workItemID workflow.WorkItemID, environment []string) (string, error) {
	return a.operations.SupervisedWorkItemInstruction(id, workItemID, environment)
}

func (a Adapter) RunTaskAgentWithEnvironment(args, environment []string) error {
	return a.operations.RunTaskAgentWithEnvironment(args, environment)
}

func (a Adapter) AddTaskWorktree(id string) error { return a.operations.AddTaskWorktree(id) }

func (a Adapter) CreateTaskSession(id, worktree string) error {
	return a.operations.CreateTaskSession(id, worktree)
}

func (a Adapter) LocalMergeDelivery(id string, meta task.TaskMeta, store *workflow.Store, message string) error {
	return a.operations.LocalMergeDelivery(id, meta, store, message)
}

func (a Adapter) PreflightForceCloseDelivery(meta task.TaskMeta, delivery workflow.DeliveryState) error {
	return a.operations.PreflightForceCloseDelivery(meta, delivery)
}

func (a Adapter) RunFocusedTest(meta task.TaskMeta, state workflow.RuntimeState, store *workflow.Store, attemptID workflow.AttemptID) (task.FocusedTestResult, error) {
	return a.operations.RunFocusedTest(meta, state, store, attemptID)
}

func (a Adapter) RepairMetadata(args []string) error {
	return a.operations.RepairMetadata(args)
}
