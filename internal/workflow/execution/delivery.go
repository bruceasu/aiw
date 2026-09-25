package execution

import (
	"fmt"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// DeliverCompleted invokes Git Delivery only for an eligible completed isolated Task.
// A false result leaves the Task available for diagnosis or later execution.
func (s Supervisor) DeliverCompleted(id string) (bool, error) {
	store := s.Store
	state, err := store.Load(workflow.TaskID(id))
	if err != nil {
		return false, err
	}
	if state.SchemaVersion == workflow.DurableSchemaVersion { return false, fmt.Errorf("durable delivery requires the grant-bound managed action runner") }
	summary := workflow.DeriveSummary(state)
	if summary.Execution != workflow.ExecutionCompleted ||
		(summary.Validation != workflow.ValidationPassed && summary.Validation != workflow.ValidationNotRequired && summary.Validation != workflow.ValidationWaived) ||
		state.Delivery == workflow.DeliveryMerged || state.Delivery == workflow.DeliveryDiscarded ||
		state.Automation.PreparedRequest != nil ||
		workflow.NextRunnerOutcome(state).Kind != workflow.RunnerNoWork {
		return false, nil
	}
	meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id))
	if err != nil {
		return false, err
	}
	if taskx.ResolvedWorkspaceKind(meta) != "isolated" {
		return false, nil
	}
	if err := s.Merge(id, meta, store, "Complete Task "+id); err != nil {
		return false, err
	}
	s.Delivered(id, meta.ParentBranch)
	return true, nil
}
