package execution

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"aiw/internal/workflow"
)

// AIWCleanupHost implements only the managed ownership/ancestry/source-checked
// operations. No cleanup shell command is exposed to an Agent.
type AIWCleanupHost interface {
	RemoveWorktree(workflow.ManagedDeliveryPlan, workflow.DeliveryAction) error
	UnassignWorkspace(workflow.ManagedDeliveryPlan, workflow.DeliveryAction) error
	RemoveBranch(workflow.ManagedDeliveryPlan, workflow.DeliveryAction) error
}

// LocalDeliveryExecutor contains the local Git allowlist. The host inspector
// supplies exact commit/index/path facts and durable observation evidence.
// E03 supplies Store.DeliveryServices; E05 supplies sealed-source inspection.
// Missing host capabilities fail closed and never fall back to localMerge.
type LocalDeliveryExecutor struct {
	Store *workflow.Store
	ExecutorID string
	Cleanup AIWCleanupHost
	Inspect func(workflow.ManagedDeliveryPlan, workflow.DeliveryAction, string) (workflow.DeliveryObservation, error)
	HostCheck func(workflow.RuntimeState, workflow.ManagedDeliveryPlan, workflow.DeliveryAction) error
	// Git is an optional controlled host runner, useful for bounded tests. The
	// default passes argv directly to Git; it never invokes a shell.
	Git func(string, ...string) error
}

func (e LocalDeliveryExecutor) Identity() string { return e.ExecutorID }

func (e LocalDeliveryExecutor) Check(state workflow.RuntimeState, plan workflow.ManagedDeliveryPlan, action workflow.DeliveryAction) error {
	if e.Store == nil || e.ExecutorID == "" || e.Inspect == nil || e.HostCheck == nil || e.Store.DeliveryServices == nil || e.Store.DeliveryServices.Check == nil { return errors.New("managed local delivery host is incomplete") }
	if state.Protocol == nil || state.Protocol.Stop != nil || state.Protocol.Delivery == nil { return errors.New("managed delivery is disabled or stopped") }
	saved, _ := json.Marshal(state.Protocol.Delivery.Plan)
	current, _ := json.Marshal(plan)
	if string(saved) != string(current) { return errors.New("delivery plan differs from the frozen Core plan") }
	matched := false
	for _, planned := range plan.Actions { if planned == action { matched = true } }
	if !matched { return errors.New("delivery action differs from its frozen plan") }
	if err := e.Store.DeliveryServices.Check(state, plan, action); err != nil { return err }
	return e.HostCheck(state, plan, action)
}

func (e LocalDeliveryExecutor) Reconcile(plan workflow.ManagedDeliveryPlan, action workflow.DeliveryAction) (workflow.DeliveryObservation, error) {
	if e.Inspect == nil { return workflow.DeliveryObservation{}, errors.New("delivery inspector is unavailable") }
	return e.Inspect(plan, action, e.ExecutorID)
}

func (e LocalDeliveryExecutor) Execute(plan workflow.ManagedDeliveryPlan, action workflow.DeliveryAction) (workflow.DeliveryObservation, error) {
	if e.Store == nil { return workflow.DeliveryObservation{}, errors.New("managed Store is unavailable") }
	state, err := e.Store.Load(plan.TaskID)
	if err != nil { return workflow.DeliveryObservation{}, err }
	if err := e.Check(state, plan, action); err != nil { return workflow.DeliveryObservation{}, err }
	claimed := false
	for _, record := range state.Protocol.Delivery.Actions { if record.ID == action.ID && record.State == "unknown" && record.Executor == e.ExecutorID { claimed = true } }
	if !claimed { return workflow.DeliveryObservation{}, errors.New("delivery action has no owned durable intent") }
	run := e.Git
	if run == nil {
		run = func(dir string, args ...string) error {
			command := exec.Command("git", args...)
			command.Dir = dir
			output, err := command.CombinedOutput()
			if err != nil { return fmt.Errorf("local Git action: %w: %s", err, output) }
			return nil
		}
	}
	switch action.Kind {
	case "create-branch":
		err = run(plan.Binding.ParentPath, "branch", "--", plan.Binding.TaskBranch, plan.TaskCommit)
	case "commit":
		if plan.Message == "" { return workflow.DeliveryObservation{}, errors.New("commit message is required") }
		paths := make([]string, 0, len(plan.Paths))
		for _, path := range plan.Paths { paths = append(paths, path.Path) }
		if len(paths) == 0 { return workflow.DeliveryObservation{}, errors.New("commit paths are empty") }
		err = run(plan.Binding.WorktreePath, append([]string{"--literal-pathspecs", "add", "--"}, paths...)...)
		if err == nil {
			// Index mutation is part of this frozen commit action. Recheck the
			// allowed staged state, current Stop and grant before the commit.
			current, loadErr := e.Store.Load(plan.TaskID)
			if loadErr != nil { err = loadErr } else { err = e.Check(current, plan, action) }
			if err == nil { err = run(plan.Binding.WorktreePath, append([]string{"--literal-pathspecs", "commit", "--only", "-m", plan.Message, "--"}, paths...)...) }
		}
	case "merge":
		if plan.Message == "" { return workflow.DeliveryObservation{}, errors.New("merge message is required") }
		err = run(plan.Binding.ParentPath, "merge", "--no-ff", "-m", plan.Message, "--", plan.Binding.TaskBranch)
	case "remove-worktree", "unbind-workspace", "remove-branch":
		if e.Cleanup == nil { return workflow.DeliveryObservation{}, errors.New("AIW cleanup host is unavailable") }
		switch action.Kind {
		case "remove-worktree": err = e.Cleanup.RemoveWorktree(plan, action)
		case "unbind-workspace": err = e.Cleanup.UnassignWorkspace(plan, action)
		case "remove-branch": err = e.Cleanup.RemoveBranch(plan, action)
		}
	default:
		return workflow.DeliveryObservation{}, errors.New("action is outside the local delivery allowlist")
	}
	// A conflict or crash is observed in place. No reset, abort, force-delete,
	// push, PR, archive or implicit cleanup is attempted here.
	observation, observeErr := e.Reconcile(plan, action)
	return observation, errors.Join(err, observeErr)
}
