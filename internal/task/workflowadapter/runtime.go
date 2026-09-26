package workflowadapter

import (
	"aiw/internal/task"
	"aiw/internal/workflow"
)

// CompatibleWorkflow loads Task metadata and ensures that the Workflow
// runtime has the durable compatibility projection required by Task commands.
func CompatibleWorkflow(id string) (task.TaskMeta, *workflow.Store, error) {
	meta, err := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	if err != nil {
		return task.TaskMeta{}, nil, err
	}
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(WorkflowRuntimeFromMeta(meta)); err != nil {
		return task.TaskMeta{}, nil, err
	}
	return meta, store, nil
}
