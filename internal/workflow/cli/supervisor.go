package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/ui"
	"aiw/internal/workflow"
	"aiw/internal/workflow/execution"
)

// runWorkflowSupervisor routes the supervise CLI action and prints recovery
// guidance for any failed start, status, or stop operation.
func runWorkflowSupervisor(args []string) (runErr error) {
	if len(args) < 3 || !taskAdapter.SafeID(args[1]) {
		return fmt.Errorf("usage: wf supervise <task-id> <start|status|stop> [--provider NAME] [--model MODEL]")
	}
	id, action := args[1], args[2]
	defer func() {
		if runErr != nil {
			printSupervisorFailureGuidance(id, runErr)
		}
	}()
	provider, model, err := parseProviderModelOverrides(args[3:])
	if err != nil {
		return err
	}
	if action != "start" && (provider != "" || model != "") {
		return fmt.Errorf("provider/model overrides are only valid for supervisor start")
	}
	if action != "start" && action != "status" && action != "stop" {
		return fmt.Errorf("usage: wf supervise <task-id> <start|status|stop>")
	}
	store := workflow.NewStore("")
	state, err := store.Load(workflow.TaskID(id))
	if action == "start" && errors.Is(err, os.ErrNotExist) {
		// Only durable metadata can initialize a missing projection. A change
		// directory or migration marker cannot restore execution history.
		meta, metaErr := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
		if metaErr != nil {
			return fmt.Errorf("cannot initialize workflow Task %s: restore valid task metadata before starting; existing artifacts were preserved: %w", id, metaErr)
		}
		if meta.ID != id {
			return fmt.Errorf("task metadata id mismatch: expected %s, got %s", id, meta.ID)
		}
		state, err = store.EnsureCompatible(taskworkflow.WorkflowRuntimeFromMeta(meta))
	}
	if err != nil {
		return err
	}
	switch action {
	case "status":
		printSupervisorStatus(state)
		printSupervisorRuntimeStatus(state)
		if state.Automation.Cursor.Result == "awaiting-human" {
			if _, report, readErr := store.ReadPendingRemediation(workflow.TaskID(id)); readErr == nil {
				printRemediationChoices(report)
			} else {
				fmt.Printf("human response unavailable: %v; next: aiw wf diagnose %s\n", readErr, id)
			}
		}
		if p := state.Protocol; p != nil {
			fmt.Printf("Durable execution: revision=%d budget-known=%t stopped=%t\n", state.StateRevision, p.BudgetKnown, p.Stop != nil)
			if p.Stop != nil { fmt.Printf("Stop reason: %s\n", p.Stop.Reason) }
			for _, item := range p.Items { fmt.Printf("%s: phase=%s request=%s\n", item.WorkItemID, item.Phase, item.CurrentRequest) }
			for _, request := range p.Requests { if !request.Consumed && request.Dispatch != "not-dispatched" { fmt.Printf("In flight: %s observation=%s executor=%s\n", request.Request.ID, request.Dispatch, request.Executor) } }
			for _, recovery := range p.Recoveries { fmt.Printf("Recovery %s/%s: remaining=%d\n", recovery.WorkItemID, recovery.Phase, 2-len(recovery.Requests)) }
			if a := p.Auxiliary; a != nil {
				if a.HostGap != "" { fmt.Printf("Auxiliary host gap: %s\n", a.HostGap) }
				fmt.Printf("Auxiliary sources: consumed=%d total=%d\n", a.Cursor, len(a.Sources))
				for _, job := range a.Jobs { fmt.Printf("Auxiliary %s/%s: state=%s recovery-used=%t reason=%s\n", job.Kind, job.Key, job.State, job.RecoveryUsed, job.Reason) }
			}
		}
		return nil
	case "start":
		return startWorkflowSupervisor(id, provider, model, store, state)
	case "stop":
		if state.SchemaVersion == workflow.DurableSchemaVersion {
			_, err := store.RequestExecutionStop(workflow.TaskID(id), state.StateRevision, "Explicit supervisor Stop", "wf supervise stop")
			return err
		}
		if state.Automation.Supervisor.LeaseID == "" {
			return nil
		}
		_, err := store.StopSupervisor(workflow.TaskID(id), state.Automation.Supervisor.LeaseID)
		return err
	default:
		return fmt.Errorf("usage: wf supervise <task-id> <start|status|stop>")
	}
}

// RunWorkflowSupervisor is the narrow compatibility seam used by Task-owned
// tests and adapters. Production CLI dispatch calls the private implementation
// directly from this package.
func RunWorkflowSupervisor(adapter TaskAdapter, args []string) error {
	taskAdapter = adapter
	return runWorkflowSupervisor(args)
}

// startWorkflowSupervisor renders terminal Tasks or delegates execution to the
// foreground Supervisor with CLI reporting callbacks.
func startWorkflowSupervisor(id, provider, model string, store *workflow.Store,
	state workflow.RuntimeState) error {
	if state.Automation.Cursor.Result == "awaiting-human" {
		_, report, err := store.ReadPendingRemediation(workflow.TaskID(id))
		if err != nil { return err }
		printRemediationChoices(report)
		return nil
	}
	store.ResumeAuxiliaryHost(workflow.TaskID(id))
	if state.Delivery == workflow.DeliveryMerged || state.Delivery == workflow.DeliveryDiscarded {
		printSupervisorStatus(state)
		return nil
	}
	return newWorkflowSupervisor(store, state.Task.Workspace, provider, model).Start(id, provider, model, state)
}

// newWorkflowSupervisor wires execution to CLI reporting and the existing
// single-step Runner and Git Delivery adapters.
func newWorkflowSupervisor(store *workflow.Store, workspace, provider, model string) execution.Supervisor {
	return execution.Supervisor{
		Store: store,
		Analyzer: execution.LLMRemediationAnalyzer{Workspace: workspace, Provider: provider, Model: model},
		RunStep: func(id string, execute bool, provider, model string) error {
			return runWorkflowWithOverrides(id, execute, true, false, provider, model)
		},
		Merge:  taskAdapter.LocalMergeDelivery,
		Report: projectWorkflowState,
		Delivered: func(id, parentBranch string) {
			fmt.Printf("Task %s locally merged into %s; worktree and task branch removed. Review the parent branch before pushing.\n", id, parentBranch)
		},
	}
}

// parseProviderModelOverrides accepts only the optional provider and model
// overrides supported by supervise start.
func parseProviderModelOverrides(args []string) (string, string, error) {
	provider, model := "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "--model":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return "", "", fmt.Errorf("%s requires a value", args[i])
			}
			if args[i] == "--provider" {
				provider = args[i+1]
			} else {
				model = args[i+1]
			}
			i++
		default:
			return "", "", fmt.Errorf("unknown supervisor option: %s", args[i])
		}
	}
	return provider, model, nil
}

// printSupervisorFailureGuidance reports the recorded diagnostics and the next
// recovery action without mutating Task state.
func printSupervisorFailureGuidance(id string, cause error) {
	fmt.Fprintf(os.Stderr, "supervisor: stopped with an error for task=%s\n", id)
	fmt.Fprintf(os.Stderr, "next: aiw wf diagnose %s\n", id)
	store := workflow.NewStore("")
	diagnostics, err := store.Diagnose(workflow.TaskID(id))
	if err != nil {
		fmt.Fprintf(os.Stderr, "diagnosis unavailable: %v\n", err)
	} else {
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "runtime-missing" {
				meta, metaErr := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
				if metaErr != nil || meta.ID != id {
					diagnostic.Repair = "restore valid Task metadata from backup before starting; a change directory or migration marker cannot restore execution history"
				} else {
					diagnostic.Repair = "supervise start can initialize an inactive projection from the existing Task metadata; missing Attempt history will not be restored"
				}
			}
			fmt.Fprintf(os.Stderr, "detected %s: %s\nnext: %s\n", diagnostic.Code, diagnostic.Message, diagnostic.Repair)
		}
	}
	if meta, metaErr := task.ReadTaskMeta(task.ResolveTaskMetaPath(id)); metaErr == nil && taskAdapter.ResolveWorkspaceKind(meta) == "isolated" && !taskAdapter.VerifiedTaskWorktree(meta) {
		fmt.Fprintf(os.Stderr, "detected workspace-binding-invalid: the isolated worktree is missing or unregistered\nnext: aiw wt repair\n")
	}
	message := strings.ToLower(cause.Error())
	switch {
	case strings.Contains(message, "rename") || strings.Contains(message, "access is denied"):
		fmt.Fprintf(os.Stderr, "file access failure: wait for another process to release the state file; if diagnose reports event-pending, run: aiw wf recover %s\n", id)
	case strings.Contains(message, "invalid isolated worktree") || strings.Contains(message, "worktree"):
		fmt.Fprintln(os.Stderr, "workspace failure: inspect the worktree binding; repair it before retrying the supervisor")
	case strings.Contains(message, "session"):
		fmt.Fprintln(os.Stderr, "session failure: inspect the recorded Session output before retrying; preserve the Attempt so its result remains auditable")
	case strings.Contains(message, "gate"):
		fmt.Fprintln(os.Stderr, "gate failure: resolve or waive only the reported Gate after the required human decision")
	}
	fmt.Fprintf(os.Stderr, "then inspect: aiw wf supervise %s status\n", id)
	fmt.Fprintln(os.Stderr, "do not start another supervisor until diagnosis or recovery completes")
}

// printSupervisorStatus renders the durable lease and retry snapshot.
func printSupervisorStatus(state workflow.RuntimeState) {
	terminal := ui.NewTerminal(os.Stdout)
	supervisor := state.Automation.Supervisor
	status := "stopped"
	if supervisor.LeaseID != "" {
		status = "lease-valid"
		if supervisor.LeaseExpiresAt != "" {
			expires, err := time.Parse(time.RFC3339, supervisor.LeaseExpiresAt)
			if err == nil && !time.Now().UTC().Before(expires) {
				status = "lease-expired"
			}
		}
	}
	terminal.Section("Supervisor")
	terminal.State("snapshot", status)
	terminal.Field("lease", supervisor.LeaseID)
	terminal.Field("expires", supervisor.LeaseExpiresAt)
	terminal.Field("observed event", fmt.Sprintf("%d", supervisor.ObservedEvent))
	if state.Policy != nil {
		terminal.Field("policy digest", state.Policy.Digest)
	}
	if supervisor.Result != "" {
		terminal.Section("Last result")
		terminal.Field("result", supervisor.Result)
		terminal.Field("detail", supervisor.Detail)
	}
	if supervisor.RetryAfter != "" {
		retryStatus := "eligible"
		if retryAt, err := time.Parse(time.RFC3339, supervisor.RetryAfter); err == nil && time.Now().UTC().Before(retryAt) {
			retryStatus = "not-yet-eligible"
		}
		terminal.Section("Retry")
		terminal.State("eligibility", retryStatus)
		terminal.Field("eligible at", supervisor.RetryAfter)
		terminal.Field("scheduling", "not confirmed by this timestamp")
	}
	compileState, compileNext, diagnostics := supervisorCompileGuidance(state)
	if compileState != "" {
		terminal.Section("Compilation")
		terminal.State("state", compileState)
		if diagnostics != "" { terminal.Field("diagnostics", diagnostics) }
	}
	if state.Automation.Cursor.Result == "awaiting-human" {
		terminal.Field("next", fmt.Sprintf("fill response file, then aiw wf continue %s", state.Task.ID))
	} else if supervisor.Result == "budget-awaiting-approval" {
		terminal.Field("next", fmt.Sprintf("review budget decision: aiw wf budget %s approve --reason <reason>", state.Task.ID))
	} else if compileNext != "" {
		terminal.Field("next", compileNext)
	} else if supervisor.RetryAfter != "" {
		terminal.Field("next", fmt.Sprintf("after eligibility, run aiw wf supervise %s start if no external supervisor is managing this Task", state.Task.ID))
	}
}

// supervisorCompileGuidance describes recorded compile state only. A retry
// timestamp is not proof that a process will restart or that a compile result
// is known, so this view never advances the request or resolves a Gate.
func supervisorCompileGuidance(state workflow.RuntimeState) (status, next, diagnostics string) {
	id := state.Task.ID
	request := state.Automation.PreparedRequest
	if request != nil && request.Compile != nil && request.Compile.Result != nil {
		diagnostics = request.Compile.Result.Diagnostics.Path
	}
	if diagnostics == "" && request != nil {
		for _, item := range state.WorkItems {
			if item.ID == request.WorkItemID { diagnostics = item.LastOutputReference; break }
		}
	}
	for _, gate := range state.Gates {
		if gate.State != workflow.GateOpen { continue }
		switch {
		case strings.HasPrefix(string(gate.ID), "compile-plan-missing"):
			return "plan-missing", fmt.Sprintf("inspect frozen compile plan and owning Attempt: aiw wf diagnose %s; do not replace an active request or resolve the Gate without evidence", id), diagnostics
		case strings.HasPrefix(string(gate.ID), "compile-target-unavailable"):
			return "target-unavailable", fmt.Sprintf("repair the compile target in the Task workspace, then inspect the Gate: aiw wf diagnose %s; do not resolve it before verifying the target", id), diagnostics
		case strings.HasPrefix(string(gate.ID), "compiler-repair-limit-"):
			return "repair-limit-reached", fmt.Sprintf("inspect diagnostics and repair manually; then inspect the Gate and Work Item: aiw wf diagnose %s; resolve/reopen only after the repair is verified", id), diagnostics
		}
	}
	if request != nil && request.Compile != nil && request.Compile.RepairPending {
		return "repair-prepared", fmt.Sprintf("a repair turn is prepared; if no supervisor is running, run aiw wf supervise %s start to continue the same Attempt", id), diagnostics
	}
	if state.Automation.Supervisor.Result == "compiler-paused" {
		return "result-unknown", fmt.Sprintf("inspect the compiler request and recorded result first: aiw wf diagnose %s; do not rerun an uncertain compile", id), diagnostics
	}
	return "", "", ""
}

// printSupervisorRuntimeStatus renders the active request, workspace lease,
// open Gates, and pending projection repairs.
func printSupervisorRuntimeStatus(state workflow.RuntimeState) {
	terminal := ui.NewTerminal(os.Stdout)
	if request := state.Automation.PreparedRequest; request != nil {
		terminal.Section("Agent request")
		requestState := "prepared"
		if request.DispatchedAt != "" {
			requestState = "dispatched"
		}
		terminal.State("state", requestState)
		terminal.Field("work item", string(request.WorkItemID))
		terminal.Field("attempt", string(request.AttemptID))
		terminal.Field("session", request.SessionID)
		terminal.Field("workspace", request.Workspace)
		if request.SessionID != "" {
			terminal.Field("session output", fmt.Sprintf(".ai/sessions/%s/outputs", request.SessionID))
		}
	}
	if state.WriteLease != nil {
		terminal.Section("Workspace")
		terminal.State("write lease", "held")
		terminal.Field("attempt", string(state.WriteLease.AttemptID))
		terminal.Field("acquired", state.WriteLease.AcquiredAt)
		terminal.Field("path", state.WriteLease.Workspace)
	}
	for _, gate := range state.Gates {
		if gate.State == workflow.GateOpen {
			terminal.Section("Open gate")
			terminal.Field("id", string(gate.ID))
			terminal.Field("kind", string(gate.Kind))
			terminal.Field("work item", string(gate.WorkItemID))
			terminal.Field("reason", gate.Reason)
		}
	}
	for _, repair := range state.Automation.ProjectionRepairs {
		if repair.ResolvedAt == "" {
			terminal.Section("Projection repair")
			terminal.Field("event", fmt.Sprintf("%d", repair.EventSequence))
			terminal.Field("target", repair.Target)
			terminal.Field("next", repair.Recommended)
		}
	}
}

// deliverCompletedSupervisorTask connects the delivery result to CLI output.
func deliverCompletedSupervisorTask(id string, store *workflow.Store) (bool, error) {
	return newWorkflowSupervisor(store, ".", "", "").DeliverCompleted(id)
}
