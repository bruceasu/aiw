package task

import (
	"encoding/json"
	"fmt"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// managedCleanupHost is the AIW-owned implementation used by the durable
// delivery adapter. Its service verifies real paths, branch ancestry, exact
// cleanup grants and complete sealed-source evidence before each operation.
// The old automatic cleanup entry point is never a fallback for this host.
type managedCleanupHost struct { store *workflow.Store }

func (h managedCleanupHost) check(plan workflow.ManagedDeliveryPlan, action workflow.DeliveryAction, kind string) error {
	if h.store == nil || h.store.DeliveryServices == nil || h.store.DeliveryServices.Check == nil { return fmt.Errorf("managed cleanup authorization service is unavailable") }
	state, err := h.store.Load(plan.TaskID)
	if err != nil { return err }
	if state.SchemaVersion != workflow.DurableSchemaVersion || state.Protocol == nil || state.Protocol.Stop != nil || state.Protocol.Delivery == nil || state.Delivery != workflow.DeliveryMerged || plan.Sources == nil || action.Kind != kind { return fmt.Errorf("managed cleanup requires recorded merge, active intent and sealed sources") }
	saved, _ := json.Marshal(state.Protocol.Delivery.Plan)
	current, _ := json.Marshal(plan)
	if string(saved) != string(current) { return fmt.Errorf("cleanup plan differs from the persisted plan") }
	claimed := false
	for i, record := range state.Protocol.Delivery.Actions { if record.ID == action.ID && record.State == "unknown" && state.Protocol.Delivery.Plan.Actions[i] == action { claimed = true } }
	if !claimed { return fmt.Errorf("cleanup action has no durable intent") }
	if err := h.store.VerifyAuxiliarySourceSeal(state, *plan.Sources); err != nil { return err }
	return h.store.DeliveryServices.Check(state, plan, action)
}

func (h managedCleanupHost) RemoveWorktree(plan workflow.ManagedDeliveryPlan, action workflow.DeliveryAction) error {
	if err := h.check(plan, action, "remove-worktree"); err != nil { return err }
	return gitRunAt(plan.Binding.ParentPath, "worktree", "remove", "--", plan.Binding.WorktreePath)
}

func (h managedCleanupHost) UnassignWorkspace(plan workflow.ManagedDeliveryPlan, action workflow.DeliveryAction) error {
	if err := h.check(plan, action, "unbind-workspace"); err != nil { return err }
	_, err := taskx.UnassignWorkspace(taskx.ResolveTaskMetaPath(string(plan.TaskID)), string(plan.TaskID))
	return err
}

func (h managedCleanupHost) RemoveBranch(plan workflow.ManagedDeliveryPlan, action workflow.DeliveryAction) error {
	if err := h.check(plan, action, "remove-branch"); err != nil { return err }
	return gitRunAt(plan.Binding.ParentPath, "branch", "-d", "--", plan.Binding.TaskBranch)
}
