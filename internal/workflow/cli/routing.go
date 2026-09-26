package cli

import (
	"os"
	"path/filepath"

	"aiw/internal/task"
	"aiw/internal/workflow"
)

// ensureLocalRoutingPlan keeps supervision startable when advisory routing
// has not been requested explicitly. It only discovers local compile inputs;
// it never invokes a model.
func ensureLocalRoutingPlan(store *workflow.Store, id workflow.TaskID, workspace string) (workflow.RoutingPlan, error) {
	plan, err := store.LoadRoutingPlan(id)
	if err == nil || !os.IsNotExist(err) {
		return plan, err
	}
	if !filepath.IsAbs(workspace) {
		workspace = filepath.Join(task.RuntimeRoot(), filepath.FromSlash(workspace))
	}
	plan = workflow.NewRoutingPlan(id, "defaults", workflow.DefaultRoutingProfiles(), workflow.CompilePlanForWorkspace(workspace))
	_, _, err = store.PersistRoutingPlan(id, plan)
	return plan, err
}
