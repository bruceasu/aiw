package task

import (
	"os"
	"path/filepath"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// Missing advisory routing must not leave supervision without a compile plan.
// Discover local scripts only; this preparation never invokes a model.
func ensureLocalRoutingPlan(store *workflow.Store, id workflow.TaskID, workspace string) (workflow.RoutingPlan, error) {
	plan, err := store.LoadRoutingPlan(id)
	if err == nil || !os.IsNotExist(err) {
		return plan, err
	}
	if !filepath.IsAbs(workspace) {
		workspace = filepath.Join(taskx.RuntimeRoot(), filepath.FromSlash(workspace))
	}
	plan = workflow.NewRoutingPlan(id, "defaults", workflow.DefaultRoutingProfiles(), workflow.CompilePlanForWorkspace(workspace))
	_, _, err = store.PersistRoutingPlan(id, plan)
	return plan, err
}
