package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aiw/internal/ai"
	"aiw/internal/gitx"
	"aiw/internal/session"
	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/ui"
	"aiw/internal/workflow"
	"aiw/internal/workflow/execution"
)

var taskAdapter TaskAdapter

// RunWorkflowCommand executes Workflow CLI with the application adapter
// supplied by the standalone program.
func RunWorkflowCommand(adapter TaskAdapter, args []string) error {
	taskAdapter = adapter
	return runWorkflowCommand(args)
}

// PrintWorkflowHelp is the adapter consumed by the public Workflow facade.
func PrintWorkflowHelp() { printWorkflowHelp() }

// The following narrow helpers keep the old Task package's focused regression
// seam intact while production dispatch lives in this package.
func ValidateWorkflowArgs(adapter TaskAdapter, op string, args []string) error {
	taskAdapter = adapter
	return validateWorkflowArgs(op, args)
}

func ParseWorkflowRunArgs(args []string) (bool, bool, error) {
	return parseWorkflowRunArgs(args)
}

func EnsureAutomatedWorkspace(adapter TaskAdapter, id string, meta task.TaskMeta, primary bool) (task.TaskMeta, error) {
	taskAdapter = adapter
	return ensureAutomatedWorkspace(id, meta, primary)
}

func RunWorkflow(adapter TaskAdapter, id string, execute bool, supervised ...bool) error {
	taskAdapter = adapter
	return runWorkflow(id, execute, supervised...)
}

func RunWorkflowWithOverrides(adapter TaskAdapter, id string, execute, supervised, primary bool, provider, model string) error {
	taskAdapter = adapter
	return runWorkflowWithOverrides(id, execute, supervised, primary, provider, model)
}

func SyncWorkflowChecklist(id string, store *workflow.Store) (workflow.RuntimeState, error) {
	return syncWorkflowChecklist(id, store)
}

func ProjectWorkflowState(adapter TaskAdapter, id string, state workflow.RuntimeState) error {
	taskAdapter = adapter
	return projectWorkflowState(id, state)
}

func AdvanceWorkflow(adapter TaskAdapter, id string, meta task.TaskMeta, store *workflow.Store, supervised bool, provider, model string) (workflow.RuntimeState, error) {
	taskAdapter = adapter
	return advanceWorkflow(id, meta, store, supervised, provider, model)
}

func CompatibleWorkflow(id string) (task.TaskMeta, *workflow.Store, error) {
	return compatibleWorkflow(id)
}

func RecommendRoutingProfiles(workspace string) (map[string]string, error) {
	return recommendRoutingProfiles(workspace)
}

func RepairWorkflowState(adapter TaskAdapter, id string) error {
	taskAdapter = adapter
	return repairWorkflowState(id)
}

func runWorkflowCommand(args []string) error {
	if len(args) > 0 && args[0] == "knowledge" { return RunKnowledgeCommand(args[1:]) }
	if len(args) > 0 && args[0] == "auxiliary" { return execution.RunAuxiliaryMaintenance(args[1:]) }
	if len(args) > 0 && (args[0] == "continue" || args[0] == "resume") { return runRemediationResponse(args) }
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printWorkflowHelp()
		return nil
	}
	if args[0] == "repair-metadata" {
		return taskAdapter.RepairMetadata(args[1:])
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: wf <operation> <task-id> [arguments]")
	}
	op, id := args[0], args[1]
	if op == "status" {
		if len(args) != 2 {
			return fmt.Errorf("usage: wf status <task-id>")
		}
		if !taskAdapter.SafeID(id) {
			return fmt.Errorf("invalid task id: %s", id)
		}
		state, err := workflow.NewStore("").Load(workflow.TaskID(id))
		if err != nil {
			return fmt.Errorf("load Workflow status: %w", err)
		}
		printUsageStatus(id, state)
		return nil
	}
	if op == "budget" {
		if !taskAdapter.SafeID(id) {
			return fmt.Errorf("invalid task id: %s", id)
		}
		return runWorkflowBudgetCommand(id, args[2:])
	}
	if op == "recommend-routing" {
		if len(args) != 2 || !taskAdapter.SafeID(id) { return fmt.Errorf("usage: wf recommend-routing <task-id>") }
		meta, store, err := compatibleWorkflow(id)
		if err != nil { return err }
		workspace := meta.Worktree
		if strings.TrimSpace(workspace) == "" { return fmt.Errorf("Task %s workspace is unassigned", id) }
		plan := workflow.NewRoutingPlan(workflow.TaskID(id), "defaults", workflow.DefaultRoutingProfiles(), workflow.CompilePlanForWorkspace(workspace))
		if profiles, recommendErr := recommendRoutingProfiles(workspace); recommendErr == nil {
			plan = workflow.NewRoutingPlan(workflow.TaskID(id), "llm", profiles, plan.Compile)
		}
		state, reference, err := store.PersistRoutingPlan(workflow.TaskID(id), plan)
		if err != nil { return err }
		fmt.Printf("routing plan: %s (source=%s; review recommended, not required)\n", reference, plan.Source)
		return projectWorkflowState(id, state)
	}
	if op == "report" {
		if len(args) != 2 {
			return fmt.Errorf("usage: wf report <task-id>")
		}
		if !taskAdapter.SafeID(id) {
			return fmt.Errorf("invalid task id: %s", id)
		}
		content, err := workflow.NewStore("").ReadLatestFailureReport(workflow.TaskID(id))
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("no unresolved failure report for task=%s\n", id)
			return nil
		}
		if err != nil {
			return fmt.Errorf("read latest failure report: %w", err)
		}
		fmt.Print(string(content))
		return nil
	}
	if op == "usage" {
		if !taskAdapter.SafeID(id) {
			return fmt.Errorf("invalid task id: %s", id)
		}
		query, format, err := parseUsageReportArgs(args[2:])
		if err != nil {
			return err
		}
		report, err := workflow.NewStore("").GetUsageReport(context.Background(), workflow.TaskID(id), query)
		if err != nil {
			return fmt.Errorf("get Task usage report: %w", err)
		}
		if format == "json" {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(report)
		}
		costs, _ := json.Marshal(report.Totals.KnownCostByCurrency)
		outcomes, _ := json.Marshal(report.Outcomes)
		fmt.Printf("Task %s usage\n", report.TaskID)
		fmt.Printf("CALLS\t%d\nUSAGE_UNKNOWN\t%d\nKNOWN INPUT TOKENS\t%d\nKNOWN OUTPUT TOKENS\t%d\nKNOWN TOTAL TOKENS\t%d\nKNOWN COST BY CURRENCY\t%s\nOUTCOMES\t%s\nBUDGET APPROVALS\t%d\nBUDGET TERMINATIONS\t%d\nDIFFICULTY CHANGES\t%d\n",
			report.Totals.Calls, report.Totals.UsageUnknown,
			report.Totals.KnownInputTokens, report.Totals.KnownOutputTokens,
			report.Totals.KnownTotalTokens, costs, outcomes,
			len(report.BudgetApprovals), len(report.BudgetTerminations), len(report.DifficultyChanges))
		for _, approval := range report.BudgetApprovals {
			fmt.Printf("BUDGET APPROVAL\t%s\t%s\t%s\t%d -> %d\n", approval.At, approval.Actor, approval.Reason, approval.Previous.TokenLimit, approval.New.TokenLimit)
		}
		for _, change := range report.DifficultyChanges {
			fmt.Printf("DIFFICULTY CHANGE\t%s\t%s/%s/%d -> %s/%s/%d\t%s\n", change.RecordedAt.Format(time.RFC3339), change.PreviousProfile, change.PreviousModel, change.PreviousLevel, change.SelectedProfile, change.SelectedModel, change.SelectedLevel, change.Reason)
		}
		fmt.Printf("RECORDED AT\tWORK ITEM\tATTEMPT\tPROVIDER\tPROFILE\tMODEL\tOUTCOME\n")
		for _, call := range report.Calls {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n", call.RecordedAt.Format(time.RFC3339), call.WorkItemID, call.AttemptID, call.Provider, call.Profile, call.Model, call.Outcome)
		}
		return nil
	}
	if op == "diagnose" {
		if len(args) != 2 {
			return fmt.Errorf("usage: wf diagnose <task-id>")
		}
		diagnostics, err := workflow.NewStore("").Diagnose(workflow.TaskID(id))
		if err != nil {
			return err
		}
		for _, diagnostic := range diagnostics {
			fmt.Printf("%s: %s\nrepair: %s\n", diagnostic.Code, diagnostic.Message, diagnostic.Repair)
		}
		meta, metaErr := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
		if metaErr == nil && taskAdapter.ResolveWorkspaceKind(meta) == "isolated" && !taskAdapter.VerifiedTaskWorktree(meta) {
			fmt.Printf("workspace-binding-invalid: Task %s isolated worktree is not registered or does not exist\nrepair: aiw wt repair\n", id)
		}
		return nil
	}
	if op == "recover" {
		if len(args) != 2 {
			return fmt.Errorf("usage: wf recover <task-id>")
		}
		state, err := workflow.NewStore("").RecoverPendingEvent(workflow.TaskID(id))
		if err != nil {
			return err
		}
		return projectWorkflowState(id, state)
	}
	if op == "repair" {
		if len(args) != 2 {
			return fmt.Errorf("usage: wf repair <task-id>")
		}
		return repairWorkflowState(id)
	}
	if op == "run" {
		execute, primary, provider, model, err := parseWorkflowRunArgsWithOverrides(args)
		if err != nil {
			return err
		}
		if !taskAdapter.SafeID(id) {
			return fmt.Errorf("invalid task id: %s", id)
		}
		return runWorkflowWithOverrides(id, execute, false, primary, provider, model)
	}
	if op == "supervise" {
		return runWorkflowSupervisor(args)
	}
	if op == "delivery" {
		if len(args) != 3 || (args[2] != string(workflow.DeliveryMerged) && args[2] != string(workflow.DeliveryDiscarded)) {
			return fmt.Errorf("usage: wf delivery <task-id> <merged|discarded>")
		}
		_, store, err := compatibleWorkflow(id)
		if err != nil {
			return err
		}
		state, err := store.SetDelivery(workflow.TaskID(id), workflow.DeliveryState(args[2]))
		if err != nil {
			return err
		}
		return projectWorkflowState(id, state)
	}
	if op == "local-merge" {
		if len(args) < 3 {
			return fmt.Errorf("usage: wf local-merge <task-id> <commit-message>")
		}
		meta, store, err := compatibleWorkflow(id)
		if err != nil {
			return err
		}
		return taskAdapter.LocalMergeDelivery(id, meta, store, strings.Join(args[2:], " "))
	}
	if op == "delivery-failed" {
		if len(args) < 4 {
			return fmt.Errorf("usage: wf delivery-failed <task-id> <stage> <detail>")
		}
		_, store, err := compatibleWorkflow(id)
		if err != nil {
			return err
		}
		state, err := store.RecordDeliveryFailure(workflow.TaskID(id), args[2], strings.Join(args[3:], " "))
		if err != nil {
			return err
		}
		return projectWorkflowState(id, state)
	}
	if err := validateWorkflowArgs(op, args); err != nil {
		return err
	}
	meta, store, err := compatibleWorkflow(id)
	if err != nil {
		return err
	}
	var state workflow.RuntimeState
	switch op {
	case "plan", "sync":
		state, err = syncWorkflowChecklist(id, store)
	case "advance":
		state, err = advanceWorkflow(id, meta, store, false, "", "")
	case "attempt":
		if len(args) < 3 {
			return fmt.Errorf("usage: wf attempt <task-id> <start|checkpoint> ...")
		}
		switch args[2] {
		case "start":
			if len(args) != 5 {
				return fmt.Errorf("usage: wf attempt <task-id> start <work-item-id> <attempt-id>")
			}
			state, err = store.StartAttempt(workflow.TaskID(id), workflow.Attempt{ID: workflow.AttemptID(args[4]), WorkItemID: workflow.WorkItemID(args[3]), Workspace: meta.Worktree})
		case "checkpoint":
			if len(args) != 5 {
				return fmt.Errorf("usage: wf attempt <task-id> checkpoint <attempt-id> <running|paused>")
			}
			state, err = store.CheckpointAttempt(workflow.TaskID(id), workflow.AttemptID(args[3]), workflow.AttemptState(args[4]))
		default:
			return fmt.Errorf("unknown attempt operation: %s", args[2])
		}
	case "evidence":
		if len(args) < 6 || len(args) > 7 {
			return fmt.Errorf("usage: wf evidence <task-id> <evidence-id> <work-item-id> <kind> <state> [reference]")
		}
		reference := ""
		if len(args) == 7 {
			reference = args[6]
		}
		state, err = store.RecordEvidence(workflow.TaskID(id), workflow.Evidence{ID: workflow.EvidenceID(args[2]), WorkItemID: workflow.WorkItemID(args[3]), Kind: workflow.EvidenceKind(args[4]), State: workflow.EvidenceState(args[5]), Reference: reference})
	case "gate":
		if len(args) != 4 {
			return fmt.Errorf("usage: wf gate <task-id> <gate-id> <resolved|waived>")
		}
		state, err = store.ResolveGate(workflow.TaskID(id), workflow.GateID(args[2]), workflow.GateState(args[3]))
	case "skip-focused-test":
		var workItemID workflow.WorkItemID
		state, workItemID, err = store.SkipFocusedTest(workflow.TaskID(id), strings.Join(args[2:], " "))
		if err == nil {
			err = projectAcceptedWorkItem(id, store, state, workItemID)
		}
	case "complete":
		if len(args) != 3 {
			return fmt.Errorf("usage: wf complete <task-id> <work-item-id>")
		}
		state, err = store.CompleteWorkItem(workflow.TaskID(id), workflow.WorkItemID(args[2]))
		if err == nil {
			err = projectAcceptedWorkItem(id, store, state, workflow.WorkItemID(args[2]))
		}
	case "retry-policy":
		limit, parseErr := strconv.Atoi(args[3])
		if parseErr != nil {
			return fmt.Errorf("retry limit must be a whole number")
		}
		state, err = store.SetRetryPolicy(workflow.TaskID(id), workflow.WorkItemID(args[2]), workflow.RetryPolicy{MaxAttempts: limit})
	case "reopen":
		state, err = store.ReopenWorkItem(workflow.TaskID(id), workflow.WorkItemID(args[2]), strings.Join(args[3:], " "))
	case "force-close":
		if err := taskAdapter.PreflightForceCloseDelivery(meta, workflow.DeliveryState(args[2])); err != nil {
			if _, recordErr := store.RecordDeliveryFailure(workflow.TaskID(id), "preflight", err.Error()); recordErr != nil {
				return fmt.Errorf("%w; record recovery evidence: %v", err, recordErr)
			}
			return err
		}
		state, err = store.ForceClose(workflow.TaskID(id), strings.Join(args[3:], " "), workflow.DeliveryState(args[2]))
	case "focused-test":
		state, err = store.Load(workflow.TaskID(id))
		if err != nil {
			break
		}
		execution, executeErr := taskAdapter.RunFocusedTest(meta, state, store, workflow.AttemptID(args[2]))
		if execution.RuntimeState.Task.ID != "" {
			if projectErr := projectWorkflowState(id, execution.RuntimeState); projectErr != nil {
				return projectErr
			}
		}
		return executeErr
	default:
		return fmt.Errorf("unknown workflow operation: %s", op)
	}
	if err != nil {
		return err
	}
	return projectWorkflowState(id, state)
}

func parseWorkflowRunArgs(args []string) (bool, bool, error) {
	execute, primary, _, _, err := parseWorkflowRunArgsWithOverrides(args)
	return execute, primary, err
}

func parseWorkflowRunArgsWithOverrides(args []string) (bool, bool, string, string, error) {
	execute, primary := false, false
	provider, model := "", ""
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--execute":
			execute = true
		case "--primary":
			primary = true
		case "--provider", "--model":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return false, false, "", "", fmt.Errorf("%s requires a value", args[i])
			}
			if args[i] == "--provider" {
				provider = args[i+1]
			} else {
				model = args[i+1]
			}
			i++
		default:
			if strings.HasPrefix(args[i], "--provider=") {
				provider = strings.TrimPrefix(args[i], "--provider=")
			} else if strings.HasPrefix(args[i], "--model=") {
				model = strings.TrimPrefix(args[i], "--model=")
			} else {
				return false, false, "", "", fmt.Errorf("usage: wf run <task-id> [--execute] [--primary] [--provider NAME] [--model MODEL]")
			}
		}
	}
	if primary && !execute {
		return false, false, "", "", fmt.Errorf("--primary requires --execute")
	}
	return execute, primary, provider, model, nil
}

func runWorkflow(id string, execute bool, supervised ...bool) error {
	return runWorkflowWithOptions(id, execute, len(supervised) > 0 && supervised[0], false)
}

func runWorkflowWithOptions(id string, execute, supervised, primary bool) error {
	return runWorkflowWithOverrides(id, execute, supervised, primary, "", "")
}

func runWorkflowWithOverrides(id string, execute, supervised, primary bool, provider, model string) error {
	meta, store, err := compatibleWorkflow(id)
	if err != nil {
		return err
	}
	if execute {
		meta, err = ensureAutomatedWorkspace(id, meta, primary)
		if err != nil {
			return err
		}
		if !primary {
			if err := taskAdapter.PrepareAutomatedWorkspace(store, id, meta); err != nil {
				return fmt.Errorf("coordinate automated workspace: %w", err)
			}
		}
	}
	var sessionEnvironment []string
	if execute && supervised {
		sessionEnvironment, err = taskAdapter.PreflightSupervisedGitWorkspace(meta)
		if err != nil {
			return recordSupervisedGitPreflightFailure(store, id, err)
		}
	}
	state, err := syncWorkflowChecklist(id, store)
	if err != nil {
		return err
	}
	if execute && supervised {
		state, skippedWorkItem, skipErr := skipUnconfiguredFocusedVerification(id, store, state)
		if skipErr != nil {
			return skipErr
		}
		if skippedWorkItem != "" {
			return projectAcceptedWorkItem(id, store, state, skippedWorkItem)
		}
	}
	outcome := workflow.NextRunnerOutcome(state)
	if outcome.Kind == "" {
		state, err = advanceWorkflow(id, meta, store, supervised, provider, model)
		if err != nil {
			return err
		}
		outcome = workflow.NextRunnerOutcome(state)
	}
	if outcome.Kind != workflow.RunnerPrepared {
		state, err = store.RecordAutomation(workflow.TaskID(id), state.Automation.PlanFingerprint, workflow.AutomationCursor{Result: string(outcome.Kind), Detail: outcome.Detail}, state.Automation.PreparedRequest)
		if err != nil {
			return err
		}
		return projectWorkflowState(id, state)
	}
	if !execute {
		return projectWorkflowState(id, state)
	}
	if supervised && outcome.Request.AISelection != nil {
		provider, model = outcome.Request.AISelection.Provider, outcome.Request.AISelection.Model
	}
	if state.Policy == nil {
		snapshot, resolveErr := workflow.ResolvePolicy(workflow.PolicyLayers{
			{Ordinary: map[string]string{
				"network":                workflow.NetworkPolicyDeny,
				"local-unit-test-runner": "aiw-no-network-runner",
			}, Capabilities: map[string]workflow.CapabilityAuthorization{
				"local-unit-test": {State: workflow.CapabilityAllowed},
			}},
			{Ordinary: map[string]string{"provider": provider, "model": model}},
		})
		if resolveErr != nil {
			return resolveErr
		}
		state, err = store.SnapshotPolicy(workflow.TaskID(id), snapshot)
		if err != nil {
			return err
		}
	}
	if err := taskworkflow.EnsureRunnerHandoff(id, outcome.Request, supervised); err != nil {
		return err
	}
	// A request can be prepared by a non-executing workflow pass. Only the
	// supervisor that is about to invoke the managed adapter may mark it
	// dispatched; a later recovery must inspect its bound Session result rather
	// than silently create another turn.
	if supervised {
		if outcome.Request.DispatchedAt != "" {
			return nil
		}
		if _, err := taskAdapter.SupervisedWorkItemInstruction(id, outcome.Request.WorkItemID, sessionEnvironment); err != nil {
			return recordSupervisedGitPreflightFailure(store, id, err)
		}
		if err := execution.EnsureSupervisedCompilePlan(store, outcome.Request); err != nil {
			return err
		}
		if state.SchemaVersion < 10 {
			if _, err := store.DispatchPreparedAgentRequest(workflow.TaskID(id), outcome.Request.AttemptID); err != nil { return err }
		}
		if outcome.Request.Handoff != "" {
			args := []string{"turn", id, "--supervised", "--handoff", outcome.Request.Handoff}
			if provider != "" { args = append(args, "--provider", provider) }
			if model != "" { args = append(args, "--model", model) }
			return taskAdapter.RunTaskAgentWithEnvironment(args, sessionEnvironment)
		}
	}
	return runBoundedTaskTurn(id, supervised, sessionEnvironment, provider, model)
}

// recordSupervisedGitPreflightFailure keeps every failure before Agent dispatch
// in the Core-owned workspace-access diagnostic path. In particular, it must
// not turn missing trust evidence into an empty Agent instruction or broaden
// the command-local safe.directory scope.
func recordSupervisedGitPreflightFailure(store *workflow.Store, id string, preflightErr error) error {
	if _, gateErr := store.OpenWorkspaceAccessGate(workflow.TaskID(id), preflightErr.Error()); gateErr != nil {
		return fmt.Errorf("%w; record workspace-access Gate: %v", preflightErr, gateErr)
	}
	return preflightErr
}

func skipUnconfiguredFocusedVerification(id string, store *workflow.Store, state workflow.RuntimeState) (workflow.RuntimeState, workflow.WorkItemID, error) {
	if _, err := os.Stat(task.VerificationPlanPath(id)); err == nil {
		return state, "", nil
	} else if !os.IsNotExist(err) {
		return state, "", fmt.Errorf("inspect verification plan: %w", err)
	}
	updated, workItemID, err := store.SkipFocusedTest(workflow.TaskID(id), "no Verification Plan supplied; optional focused verification skipped by supervisor policy")
	if err != nil && !strings.Contains(err.Error(), "has no authorized focused verification Work Item") {
		return state, "", err
	}
	if workItemID == "" {
		return state, "", nil
	}
	return updated, workItemID, nil
}

// runBoundedTaskTurn is the canonical handoff from automatic workflow
// execution to the one-turn Task Agent command. Keeping this boundary here
// ensures Runner and Supervisor never start an interactive Chat or a legacy
// agent command surface.
func runBoundedTaskTurn(id string, supervised bool, environment []string, overrides ...string) error {
	args := []string{"turn", id}
	if supervised {
		args = append(args, "--supervised")
	}
	if len(overrides) > 0 && strings.TrimSpace(overrides[0]) != "" {
		args = append(args, "--provider", overrides[0])
	}
	if len(overrides) > 1 && strings.TrimSpace(overrides[1]) != "" {
		args = append(args, "--model", overrides[1])
	}
	return taskAdapter.RunTaskAgentWithEnvironment(args, environment)
}

func ensureAutomatedWorkspace(id string, meta task.TaskMeta, primary bool) (task.TaskMeta, error) {
	kind := taskAdapter.ResolveWorkspaceKind(meta)
	if primary {
		if kind != "primary" {
			return task.TaskMeta{}, fmt.Errorf("--primary requires Task %s to be bound to the primary workspace, found %s", id, kind)
		}
		isPrimary, primaryPath, err := gitx.IsPrimaryWorktree()
		if err != nil {
			return task.TaskMeta{}, err
		}
		if !isPrimary {
			return task.TaskMeta{}, fmt.Errorf("--primary requires the current workspace to be the primary worktree: %s", primaryPath)
		}
		fmt.Printf("workflow workspace: primary task=%s\n", id)
		return meta, nil
	}
	if kind == "isolated" {
		if !taskAdapter.VerifiedTaskWorktree(meta) {
			return task.TaskMeta{}, fmt.Errorf("Task %s has an invalid isolated worktree binding", id)
		}
		fmt.Printf("workflow workspace: isolated task=%s worktree=%s\n", id, meta.Worktree)
		return meta, nil
	}
	if kind != "primary" {
		return task.TaskMeta{}, fmt.Errorf("Task %s workspace is %s; repair or explicitly bind it before automated execution", id, kind)
	}
	isPrimary, primaryPath, err := gitx.IsPrimaryWorktree()
	if err != nil {
		return task.TaskMeta{}, err
	}
	if !isPrimary {
		return task.TaskMeta{}, fmt.Errorf("automatic isolation must start from the primary worktree: %s", primaryPath)
	}
	if err := taskAdapter.AddTaskWorktree(id); err != nil {
		return task.TaskMeta{}, fmt.Errorf("create automatic worktree: %w", err)
	}
	meta, err = task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	if err != nil {
		return task.TaskMeta{}, err
	}
	if taskAdapter.ResolveWorkspaceKind(meta) != "isolated" || !taskAdapter.VerifiedTaskWorktree(meta) {
		return task.TaskMeta{}, fmt.Errorf("automatic worktree for Task %s was not verified", id)
	}
	fmt.Printf("workflow workspace: isolated task=%s worktree=%s\n", id, meta.Worktree)
	return meta, nil
}

func syncWorkflowChecklist(id string, store *workflow.Store) (workflow.RuntimeState, error) {
	return taskworkflow.SyncWorkflowChecklist(id, store)
}

// projectAcceptedWorkItem is the only reverse projection from Workflow Core to
// OpenSpec. workflowChecklistPath resolves the isolated Task worktree, so this
// path never writes the parent checkout during supervised execution.
func projectAcceptedWorkItem(id string, store *workflow.Store, state workflow.RuntimeState, workItemID workflow.WorkItemID) error {
	return taskworkflow.ProjectAcceptedWorkItem(id, store, state, workItemID)
}

// workflowChecklistPath selects the Task's bound workspace as the source of
// truth while automated execution is isolated. Workflow state remains stored
// in the primary workspace, but an Agent's checklist updates live in its
// isolated worktree until delivery merges that branch.
func workflowChecklistPath(id string) (string, error) {
	return taskworkflow.WorkflowChecklistPath(id)
}

func advanceWorkflow(id string, meta task.TaskMeta, store *workflow.Store, supervised bool, providerOverride, modelOverride string) (workflow.RuntimeState, error) {
	state, err := syncWorkflowChecklist(id, store)
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	for _, repair := range state.Automation.ProjectionRepairs {
		if repair.ResolvedAt == "" {
			return store.RecordAutomation(workflow.TaskID(id), state.Automation.PlanFingerprint, workflow.AutomationCursor{Result: "repair-required", Detail: repair.Target}, nil)
		}
	}
	if state.Automation.PreparedRequest != nil && state.WriteLease != nil && state.WriteLease.AttemptID == state.Automation.PreparedRequest.AttemptID {
		return state, nil
	}
	if state.WriteLease != nil {
		return store.RecordAutomation(workflow.TaskID(id), state.Automation.PlanFingerprint, workflow.AutomationCursor{Result: "blocked", Detail: "workspace write lease is active"}, nil)
	}
	item, err := workflow.SelectReadyMappedWorkItem(state)
	if err != nil {
		return store.RecordAutomation(workflow.TaskID(id), state.Automation.PlanFingerprint, workflow.AutomationCursor{Result: "no-executable-work", Detail: err.Error()}, nil)
	}
	attemptID := workflow.AttemptID(fmt.Sprintf("attempt-%d", time.Now().UTC().UnixNano()))
	var selection *workflow.AISelection
	if supervised {
		if _, err := ensureLocalRoutingPlan(store, workflow.TaskID(id), meta.Worktree); err != nil {
			return workflow.RuntimeState{}, err
		}
		selection, err = resolveSupervisedAISelection(store, workflow.TaskID(id), state, item.ID, providerOverride, modelOverride)
		if err != nil {
			return workflow.RuntimeState{}, err
		}
	}
	sessionStatus, err := session.NewStore("").Load(meta.Session)
	if errors.Is(err, session.ErrSessionNotFound) && meta.Session == id {
		if err = os.MkdirAll(filepath.Join(task.RuntimeTaskDir(id), "artifacts"), 0o755); err == nil {
			err = taskAdapter.CreateTaskSession(id, meta.Worktree)
		}
		if err == nil {
			sessionStatus, err = session.NewStore("").Load(meta.Session)
		}
	}
	if err != nil {
		return workflow.RuntimeState{}, fmt.Errorf("load prepared request Session: %w", err)
	}
	request := &workflow.PreparedAgentRequest{TaskID: workflow.TaskID(id), WorkItemID: item.ID, AttemptID: attemptID, SessionID: meta.Session, ExpectedSessionTurn: sessionStatus.Session.LastTurn + 1, Workspace: meta.Worktree, AISelection: selection}
	if supervised {
		request.Compile = &workflow.SupervisedCompileState{}
		plan, loadErr := store.LoadRoutingPlan(workflow.TaskID(id))
		if loadErr == nil { request.Compile.Plan, request.Compile.PlanReference = &plan.Compile, plan.Reference() } else if !os.IsNotExist(loadErr) { return state, loadErr }
	}
	state, err = store.StartAttempt(workflow.TaskID(id), workflow.Attempt{ID: attemptID, WorkItemID: item.ID, SessionID: meta.Session, Workspace: meta.Worktree})
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	return store.RecordAutomation(workflow.TaskID(id), state.Automation.PlanFingerprint, workflow.AutomationCursor{Result: "agent-request-prepared", Detail: string(item.ID)}, request)
}

func resolveSupervisedAISelection(store *workflow.Store, id workflow.TaskID, state workflow.RuntimeState, workItemID workflow.WorkItemID, providerOverride, modelOverride string) (*workflow.AISelection, error) {
	profileName := ai.DefaultProfileForActor("coder")
	if plan, err := store.LoadRoutingPlan(id); err == nil && strings.TrimSpace(plan.Profiles["coder"]) != "" {
		profileName = plan.Profiles["coder"]
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load routing plan: %w", err)
	}
	profile, config, err := ai.ResolveProfile(profileName)
	if err != nil {
		return nil, err
	}
	profiles, err := ai.LoadProfiles()
	if err != nil {
		return nil, err
	}
	requestedLevel, adjustmentReason := 0, ""
	previousProfileName, previousProvider, previousModel, previousIntensity := "", "", "", ""
	previousLevel := 0
	configured, ok := profiles[profileName]
	if !ok && len(profiles) > 0 {
		configured, ok = lowestProfile(profiles)
	}
	if ok {
		profile = configured
		requestedLevel = configured.Level + unresolvedAgentRounds(state, workItemID)
		previousProfile := configured
		if previous, found := latestUsageSelection(state.Protocol, workItemID); found {
			previousProfile, _, err = ai.ResolveProfile(previous.Profile)
			if err != nil { return nil, err }
			previousProvider, previousModel = previous.Provider, previous.Model
			previousIntensity = previous.ReasoningIntensity
			previousLevel = previous.DifficultyLevel
		} else {
			previousProvider, previousModel = previousProfile.Provider, previousProfile.Model
			previousIntensity, previousLevel = previousProfile.ReasoningIntensity, previousProfile.Level
		}
		selected, found := profileAtOrAbove(profiles, requestedLevel)
		if found {
			profile, config, err = ai.ResolveProfile(selected.Name)
			if err != nil { return nil, err }
		} else if requestedLevel > configured.Level {
			profile, config, err = ai.ResolveProfile(previousProfile.Name)
			if err != nil { return nil, err }
		}
		if requestedLevel > configured.Level {
			previousProfileName = previousProfile.Name
			if found {
				adjustmentReason = "unresolved_agent_round"
			} else {
				adjustmentReason = "no_higher_profile_available"
			}
		}
	}
	if strings.TrimSpace(providerOverride) != "" || strings.TrimSpace(modelOverride) != "" {
		provider, model := config.Name, config.Model
		if strings.TrimSpace(providerOverride) != "" { provider = providerOverride }
		if strings.TrimSpace(modelOverride) != "" { model = modelOverride }
		config, err = ai.ConfigFor(provider, model)
		if err != nil { return nil, err }
	}
	digestSource := strings.Join([]string{profile.Name, config.Name, config.Model, fmt.Sprint(profile.Level), profile.ReasoningIntensity, config.BaseURL, config.Command, config.CodexCommand, config.CopilotCommand}, "\n")
	digest := sha256.Sum256([]byte(digestSource))
	return &workflow.AISelection{
		Profile: profile.Name, Provider: config.Name, Model: config.Model, Digest: hex.EncodeToString(digest[:]),
		Level: profile.Level, ReasoningIntensity: profile.ReasoningIntensity,
		RequestedLevel: requestedLevel, AdjustmentReason: adjustmentReason,
		PreviousProfile: previousProfileName, PreviousProvider: previousProvider,
		PreviousModel: previousModel, PreviousLevel: previousLevel,
		PreviousReasoningIntensity: previousIntensity,
	}, nil
}

func unresolvedAgentRounds(state workflow.RuntimeState, workItemID workflow.WorkItemID) int {
	for _, item := range state.WorkItems {
		if item.ID == workItemID && (item.State == workflow.WorkItemCompleted || item.AcceptedReference != nil) { return 0 }
	}
	count := 0
	for _, attempt := range state.Attempts {
		if attempt.WorkItemID == workItemID && attempt.SessionID != "" && attempt.Outcome != nil { count++ }
	}
	return count
}

func latestUsageSelection(protocol *workflow.ExecutionProtocol, workItemID workflow.WorkItemID) (workflow.UsageEvent, bool) {
	if protocol == nil || protocol.Usage == nil { return workflow.UsageEvent{}, false }
	for i := len(protocol.Usage.Records)-1; i >= 0; i-- {
		var event workflow.UsageEvent
		if json.Unmarshal(protocol.Usage.Records[i], &event) == nil && event.WorkItemID == workItemID {
			return event, true
		}
	}
	return workflow.UsageEvent{}, false
}

func profileAtOrAbove(profiles map[string]ai.Profile, requestedLevel int) (ai.Profile, bool) {
	var selected ai.Profile
	found := false
	for _, candidate := range profiles {
		if candidate.Level < requestedLevel { continue }
		if !found || candidate.Level < selected.Level || (candidate.Level == selected.Level && candidate.Name < selected.Name) {
			selected, found = candidate, true
		}
	}
	return selected, found
}

func lowestProfile(profiles map[string]ai.Profile) (ai.Profile, bool) {
	var selected ai.Profile
	found := false
	for _, candidate := range profiles {
		if !found || candidate.Level < selected.Level || (candidate.Level == selected.Level && candidate.Name < selected.Name) {
			selected, found = candidate, true
		}
	}
	return selected, found
}

func validateWorkflowArgs(op string, args []string) error {
	if !taskAdapter.SafeID(args[1]) {
		return fmt.Errorf("invalid task id: %s", args[1])
	}
	switch op {
	case "plan", "sync", "advance", "recommend-routing":
		if len(args) != 2 {
			return fmt.Errorf("usage: wf plan <task-id>")
		}
	case "budget":
		if len(args) < 3 {
			return fmt.Errorf("usage: wf budget <task-id> <configure|approve|terminate> [options]")
		}
	case "attempt":
		if len(args) != 5 || (args[2] != "start" && args[2] != "checkpoint") {
			return fmt.Errorf("usage: wf attempt <task-id> <start|checkpoint> <id> <id-or-state>")
		}
		if args[2] == "start" {
			if _, err := workflow.ParseWorkItemID(args[3]); err != nil {
				return err
			}
			if strings.TrimSpace(args[4]) == "" {
				return fmt.Errorf("attempt id is required")
			}
		} else if strings.TrimSpace(args[3]) == "" {
			return fmt.Errorf("attempt id is required")
		} else if args[4] != string(workflow.AttemptRunning) && args[4] != string(workflow.AttemptPaused) {
			return fmt.Errorf("unsupported attempt checkpoint: %s", args[4])
		}
	case "evidence":
		if len(args) < 6 || len(args) > 7 {
			return fmt.Errorf("usage: wf evidence <task-id> <evidence-id> <work-item-id> <kind> <state> [reference]")
		}
		if strings.TrimSpace(args[2]) == "" {
			return fmt.Errorf("evidence id is required")
		}
		if _, err := workflow.ParseWorkItemID(args[3]); err != nil {
			return err
		}
		if !knownEvidenceKind(args[4]) || !knownEvidenceState(args[5]) {
			return fmt.Errorf("unsupported evidence kind or state")
		}
	case "gate":
		if len(args) != 4 {
			return fmt.Errorf("usage: wf gate <task-id> <gate-id> <resolved|waived>")
		}
		if strings.TrimSpace(args[2]) == "" || (args[3] != string(workflow.GateResolved) && args[3] != string(workflow.GateWaived)) {
			return fmt.Errorf("invalid gate resolution")
		}
	case "skip-focused-test":
		if len(args) < 3 || strings.TrimSpace(strings.Join(args[2:], " ")) == "" {
			return fmt.Errorf("usage: wf skip-focused-test <task-id> <reason>")
		}
	case "complete":
		if len(args) != 3 {
			return fmt.Errorf("usage: wf complete <task-id> <work-item-id>")
		}
		if _, err := workflow.ParseWorkItemID(args[2]); err != nil {
			return err
		}
	case "retry-policy":
		if len(args) != 4 {
			return fmt.Errorf("usage: wf retry-policy <task-id> <work-item-id> <1-5>")
		}
		if _, err := workflow.ParseWorkItemID(args[2]); err != nil {
			return err
		}
		limit, err := strconv.Atoi(args[3])
		if err != nil || limit < workflow.MinRetryLimit || limit > workflow.MaxRetryLimit {
			return fmt.Errorf("retry limit must be between %d and %d", workflow.MinRetryLimit, workflow.MaxRetryLimit)
		}
	case "reopen":
		if len(args) < 4 {
			return fmt.Errorf("usage: wf reopen <task-id> <work-item-id> <reason>")
		}
		if _, err := workflow.ParseWorkItemID(args[2]); err != nil {
			return err
		}
		if strings.TrimSpace(strings.Join(args[3:], " ")) == "" {
			return fmt.Errorf("reopen reason is required")
		}
	case "force-close":
		if len(args) < 4 || (args[2] != string(workflow.DeliveryMerged) && args[2] != string(workflow.DeliveryDiscarded)) {
			return fmt.Errorf("usage: wf force-close <task-id> <merged|discarded> <reason>")
		}
		if strings.TrimSpace(strings.Join(args[3:], " ")) == "" {
			return fmt.Errorf("force-close reason is required")
		}
	case "focused-test":
		if len(args) != 3 || strings.TrimSpace(args[2]) == "" {
			return fmt.Errorf("usage: wf focused-test <task-id> <attempt-id>")
		}
	default:
		return fmt.Errorf("unknown workflow operation: %s", op)
	}
	return nil
}

type routingRecommendation struct { Profiles map[string]string `json:"profiles"` }

func recommendRoutingProfiles(workspace string) (map[string]string, error) {
	config, err := ai.LoadConfig()
	if err != nil || strings.TrimSpace(config.Name) == "" { return nil, fmt.Errorf("no configured LLM recommendation provider") }
	profiles, err := ai.LoadProfiles()
	if err != nil { return nil, err }
	allowed := []string{"fast", "balanced", "reasoning"}
	for name := range profiles { allowed = append(allowed, name) }
	prompt := "Recommend one profile for each actor: analysis, coder, tester, verifier. Use only these profile names: " + strings.Join(allowed, ", ") + ". Return JSON with only profiles. Do not include commands. Workspace: " + filepath.ToSlash(workspace)
	output, err := ai.RunLLMWithSystemPrompt(prompt, ai.LLMConfig{Provider: config.Name, Model: config.Model, APIBaseURL: config.BaseURL, APIKey: config.APIKey, CodexCommand: config.CodexCommand, CopilotCommand: config.CopilotCommand, Workspace: workspace, ReadOnly: true}, map[string]any{"type": "object"}, "Return JSON only. Recommend safe AI routing. Never recommend commands.")
	if err != nil { return nil, err }
	var recommendation routingRecommendation
	if err := json.Unmarshal([]byte(output), &recommendation); err != nil { return nil, fmt.Errorf("decode routing recommendation: %w", err) }
	permitted := make(map[string]bool, len(allowed))
	for _, name := range allowed { permitted[name] = true }
	for actor := range workflow.DefaultRoutingProfiles() {
		if !permitted[recommendation.Profiles[actor]] { return nil, fmt.Errorf("routing recommendation has invalid %s profile", actor) }
	}
	return recommendation.Profiles, nil
}

func knownEvidenceKind(value string) bool {
	return value == string(workflow.EvidenceStaticReview) || value == string(workflow.EvidenceCommand) || value == string(workflow.EvidenceApproval) || value == string(workflow.EvidenceManual)
}

func knownEvidenceState(value string) bool {
	return value == string(workflow.EvidencePending) || value == string(workflow.EvidencePassed) || value == string(workflow.EvidenceFailed) || value == string(workflow.EvidenceWaived)
}

func compatibleWorkflow(id string) (task.TaskMeta, *workflow.Store, error) {
	meta, err := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	if err != nil {
		return task.TaskMeta{}, nil, err
	}
	store := workflow.NewStore("")
	runtime, err := store.EnsureCompatible(taskworkflow.WorkflowRuntimeFromMeta(meta))
	if err != nil {
		return task.TaskMeta{}, nil, err
	}
	wanted := taskworkflow.WorkflowRuntimeFromMeta(meta).Task
	if runtime.Task.Workspace != wanted.Workspace || runtime.Task.Kind != wanted.Kind {
		if _, err := store.ReconcileTaskReference(workflow.TaskID(id), wanted); err != nil {
			return task.TaskMeta{}, nil, fmt.Errorf("reconcile Workflow Task workspace mapping: %w", err)
		}
	}
	return meta, store, nil
}

func projectWorkflowState(id string, state workflow.RuntimeState) error {
	meta, err := taskworkflow.WriteWorkflowSummary(task.ResolveTaskMetaPath(id), state)
	if err != nil {
		return fmt.Errorf("sync task metadata snapshot after Workflow Core update: %w", err)
	}
	return reportWorkflowState(meta, state)
}

// reportWorkflowState renders Core-owned state after the caller has updated
// the coarse task.toml snapshot. Runtime storage remains the sole durable
// owner of execution details, evidence, leases, retries, and diagnostics.
func reportWorkflowState(meta task.TaskMeta, state workflow.RuntimeState) error {
	terminal := ui.NewTerminal(os.Stdout)
	summary := workflow.DeriveSummary(state)
	terminal.Section("Workflow")
	terminal.State("status", string(summary.Status))
	terminal.Field("execution", string(summary.Execution))
	terminal.Field("validation", string(summary.Validation))
	delivery := string(summary.Delivery)
	if summary.Delivery == workflow.DeliveryUnmanaged {
		delivery = "pending"
	}
	terminal.State("delivery", delivery)
	if err := reportAuthoredChecklist(meta, state, &terminal); err != nil {
		terminal.Section("Authored checklist")
		terminal.Field("status", "unavailable: "+err.Error())
	}
	if request := state.Automation.PreparedRequest; request != nil {
		terminal.Section("Prepared agent request")
		terminal.Field("work item", string(request.WorkItemID))
		terminal.Field("attempt", string(request.AttemptID))
		terminal.Field("session", request.SessionID)
	}
	reportFocusedTestStatus(state, &terminal)
	reportOpenGates(state, &terminal)
	printDeliveryGuidance(meta, state)
	return nil
}

func reportFocusedTestStatus(state workflow.RuntimeState, terminal *ui.Terminal) {
	if state.FocusedTestPlanDigest == "" && state.FocusedTestAuthorization == nil {
		return
	}
	terminal.Section("Focused test")
	terminal.Field("profile", workflow.FocusedTestProfile)
	if state.FocusedTestPlanDigest == "" {
		terminal.Field("plan digest", "unavailable")
		terminal.Field("authorization", "disabled")
		return
	}
	terminal.Field("plan digest", state.FocusedTestPlanDigest)
	terminal.Field("authorization", string(state.FocusedTestAuthorizationState(state.FocusedTestPlanDigest)))
	if authorization := state.FocusedTestAuthorization; authorization != nil {
		terminal.Field("approved by", authorization.Approver)
		terminal.Field("approved at", authorization.ApprovedAt)
	}
}

func reportOpenGates(state workflow.RuntimeState, terminal *ui.Terminal) {
	open := make([]workflow.Gate, 0, len(state.Gates))
	for _, gate := range state.Gates {
		if gate.State == workflow.GateOpen {
			open = append(open, gate)
		}
	}
	if len(open) == 0 {
		return
	}
	terminal.Section("Open gates")
	for _, gate := range open {
		reason := strings.TrimSpace(gate.Reason)
		if reason == "" {
			reason = "no reason was recorded; inspect the Workflow event history before taking action"
		}
		terminal.Field(string(gate.ID), fmt.Sprintf("kind=%s; reason=%s; next=review the reason before resolving or waiving this Gate", gate.Kind, reason))
	}
}

func reportAuthoredChecklist(meta task.TaskMeta, state workflow.RuntimeState, terminal *ui.Terminal) error {
	path, err := workflowChecklistPath(meta.ID)
	if err != nil {
		return fmt.Errorf("locate authored checklist: %w", err)
	}
	items, err := task.ReadWorkflowChecklist(path)
	if err != nil {
		return fmt.Errorf("read authored checklist: %w", err)
	}
	mappedItems := make(map[string]workflow.WorkItem, len(state.WorkItems))
	for _, item := range state.WorkItems {
		mappedItems[item.Checklist.Item] = item
	}
	terminal.Section("Authored checklist")
	for _, item := range items {
		authored := "open"
		if item.Completed {
			authored = "completed"
		}
		core := "unmapped"
		if mapped, ok := mappedItems[item.Number]; ok {
			core = string(mapped.State)
		}
		terminal.Field(item.Number, fmt.Sprintf("authored=%s; core=%s; %s", authored, core, item.Title))
	}
	return nil
}

func printDeliveryGuidance(meta task.TaskMeta, state workflow.RuntimeState) {
	terminal := ui.NewTerminal(os.Stdout)
	summary := workflow.DeriveSummary(state)
	if meta.WorkspaceKind != "isolated" || summary.Execution != workflow.ExecutionCompleted || summary.Delivery == workflow.DeliveryMerged || summary.Delivery == workflow.DeliveryDiscarded {
		return
	}
	terminal.Section("Delivery")
	terminal.State("status", "pending")
	terminal.Field("workspace", meta.Worktree)
	terminal.Field("branch", meta.Branch)
	terminal.Field("parent", meta.ParentBranch)
	terminal.Section("Next")
	terminal.Line("  1. Review the worktree changes.")
	terminal.Line("  2. Supervise automatically runs local-merge when acceptance and validation are complete and no Gate remains open.")
	terminal.Line(fmt.Sprintf("  3. For manual delivery, run `aiw wf local-merge %s \"Complete Task %s\"`.", meta.ID, meta.ID))
}

func repairWorkflowState(id string) error {
	store := workflow.NewStore("")
	current, err := store.Load(workflow.TaskID(id))
	if err != nil {
		return err
	}
	if request := current.Automation.PreparedRequest; request != nil && request.DispatchedAt == "" &&
		(request.Compile == nil || request.Compile.Plan == nil) {
		plan, err := ensureLocalRoutingPlan(store, workflow.TaskID(id), request.Workspace)
		if err != nil {
			return err
		}
		if _, err := store.RepairUndispatchedCompilePlan(workflow.TaskID(id), request.AttemptID, plan); err != nil {
			return err
		}
	}
	if current.WriteLease != nil && current.Automation.PreparedRequest == nil {
		for _, attempt := range current.Attempts {
			if attempt.ID != current.WriteLease.AttemptID || attempt.SessionID == "" {
				continue
			}
			_, sessionErr := session.NewStore("").Load(attempt.SessionID)
			if errors.Is(sessionErr, session.ErrSessionNotFound) {
				if _, err := store.RepairMissingSessionAttempt(workflow.TaskID(id), attempt.ID); err != nil {
					return err
				}
			} else if sessionErr != nil {
				return sessionErr
			}
		}
	}
	state, err := taskworkflow.RepairWorkflowChecklist(id)
	if err != nil {
		return err
	}
	return projectWorkflowState(id, state)
}

func printWorkflowHelp() {
	fmt.Print(`AIW workflow manages a Task's checklist, execution state, evidence, and recovery.

Usage:
  aiw wf <command> <task-id> [options]

Typical flow:
  1. aiw wf plan <task-id>       Create or reconcile Work Items from tasks.md.
  2. aiw wf run <task-id>        Preview the next action without executing it.
  3. aiw wf run <task-id> --execute
                                         Execute one prepared Work Item.

Auxiliary maintenance (does not enable schema 10):
  auxiliary inventory                Print a bounded local inventory for review.
  knowledge show <task>               Show versioned entries, coverage and review todo.
  knowledge review <task> <root> <file>  Review an exact version at a human terminal.
  knowledge import <task> <root> <file>  Preserve selected human text as a candidate.
  auxiliary policy <file>            Install a reviewed versioned R3 policy.
  auxiliary initialize               Consume .ai/auxiliary-inventory.json once.
  auxiliary settle                   Refresh storage without resetting model usage.

Plan and execute:
  plan <task-id>                       Create or reconcile Work Items from tasks.md.
  sync <task-id>                       Synchronize checklist completion into Workflow Core.
  recommend-routing <task-id>          Persist advisory AI routing and Compile Plan.
  advance <task-id>                    Prepare the next ready Work Item without executing it.
  run <task-id> [--execute] [--primary] [--provider NAME] [--model MODEL]
                                         Preview the next action, or execute one Work Item.
  supervise <task-id> <start|status|stop> [--provider NAME] [--model MODEL]
                                         Start, inspect, or stop managed execution; overrides apply only to start, which locally merges accepted isolated Tasks.

Record progress:
  attempt <task-id> start <work-item-id> <attempt-id>       Record a Work Item Attempt.
  attempt <task-id> checkpoint <attempt-id> <running|paused> Record an Attempt state checkpoint.
  evidence <task-id> <evidence-id> <work-item-id> <kind> <state> [reference]
                                         Record evidence; kind and state must be supported values.
  gate <task-id> <gate-id> <resolved|waived>                Resolve or waive a Gate.
  skip-focused-test <task-id> <reason>
                                      Waive optional focused verification and complete its Work Item.
  complete <task-id> <work-item-id>                       Mark a Work Item complete.
  delivery <task-id> <merged|discarded>                   Record Task delivery state.
	local-merge <task-id> <commit-message>
	                                     Commit accepted Task changes and perform verified local delivery.
  delivery-failed <task-id> <stage> <detail>              Record a delivery failure and its stage/detail.
  retry-policy <task-id> <work-item-id> <1-5>
                                         Set the automatic Attempt limit (default: 3).
  reopen <task-id> <work-item-id> <reason>
                                         Reset an exhausted Work Item after recording a reason.
  force-close <task-id> <merged|discarded> <reason>
                                         Cancel managed execution and record terminal delivery.
  focused-test <task-id> <attempt-id>   Run the selected approved focused check.

Inspect and recover:
  status <task-id>                    Show budget state, pending authorization, BLOCKED reason, Profile, and partial-usage diagnostics.
  usage <task-id> [--work-item <id>] [--attempt <id>] [--provider <id>] [--profile <id>] [--from <RFC3339>] [--to <RFC3339>] [--format table|json]
	                                     Show a bounded, read-only usage call report.
  budget <task-id> configure --tokens <n> --cost <CURRENCY=AMOUNT> [--cost <CURRENCY=AMOUNT> ...]
                                         Set the explicit initial Task budget once.
  budget <task-id> approve [--by <actor>] --reason <reason> [--tokens <n> --cost <CURRENCY=AMOUNT> ...]
                                         Approve a pending increase; omitted limits use +30%.
  budget <task-id> terminate [--by <actor>] --reason <reason>
                                         Stop at a pending budget Gate and leave the Task BLOCKED.
  report <task-id>                     Show the latest unresolved failure report without changing runtime state.
  diagnose <task-id>                   Show Workflow Core diagnostics and repair guidance.
	continue <task-id>                   Read the human remediation response and continue safely.
	resume <task-id>                     Alias of continue for a paused remediation.
	  recover <task-id>                    Re-project a committed pending transition.
	  repair <task-id>                     Re-project state and resolve recorded projection repairs.
  repair-metadata [task-id] [--dry-run]                   Preview or repair Core-derived metadata for discovered active, legacy, and archived Tasks.

Execution and workspace rules:
  - Without --execute, run only previews the next action.
  - --execute delegates one prepared Work Item to the internal Task Agent adapter.
  - Automated execution uses an isolated .wt/<task-id> worktree by default.
  - --primary is an explicit opt-out and only works for Tasks bound to the primary workspace.
  - --provider NAME and --model MODEL override the configured AI selection for run execution or supervisor start; both also accept --provider=NAME and --model=MODEL.
  - --primary requires --execute. Supervisor provider/model overrides are accepted only with supervise start.
  - supervise start commits and locally merges accepted isolated Tasks, then cleans verified merged resources and clears the current worktree binding while retaining delivery history.
  - Manual run does not deliver; push and archive remain separate operations.
  - Each Work Item has a default automatic Attempt limit of 3; retry-policy accepts 1 through 5.
  - Exhaustion blocks only that Work Item, clears its prepared request, and releases its lease.
  - reopen requires a reason and is the only operation that resets an exhausted Work Item's count.
  - force-close records CANCELLED; it never marks incomplete work or pending validation as complete.
  - merged force-close requires a clean, committed Task branch and a conflict-free merge into its recorded parent.
  - A failed force-close preflight or merge preserves the Task resources and recovery evidence.
  - focused-test accepts only an Attempt ID. It reads the persisted test-agent selection and
    approved Verification Plan; it never accepts a command, argv, directory, environment, or network override.
  - focused-test requires an explicit, digest-bound human authorization. It stops and opens a Gate
    when authorization is missing or stale, or the runtime cannot enforce network: deny.
  - retry-policy accepts a limit from 1 through 5; reopen and force-close require a non-empty reason.
  - delivery-failed joins all detail arguments after <stage> into the failure detail.
  - repair-metadata accepts at most one Task ID; --dry-run previews changes without applying them.

Examples:
  aiw wf plan payment-retry
  aiw wf run payment-retry
  aiw wf run payment-retry --execute
  aiw wf retry-policy payment-retry wi-0001 3
  aiw wf reopen payment-retry wi-0001 "authorization granted"
  aiw wf force-close payment-retry discarded "superseded by a new task"
  aiw wf diagnose payment-retry
  aiw wf focused-test payment-retry attempt-123
`)
}
