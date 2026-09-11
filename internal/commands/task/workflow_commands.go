package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/taskx"
	"aiw/internal/ui"
	"aiw/internal/workflow"
)

func runWorkflowCommand(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printWorkflowHelp()
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: task workflow <operation> <task-id> [arguments]")
	}
	op, id := args[0], args[1]
	if op == "diagnose" {
		if len(args) != 2 {
			return fmt.Errorf("usage: task workflow diagnose <task-id>")
		}
		diagnostics, err := workflow.NewStore("").Diagnose(workflow.TaskID(id))
		if err != nil {
			return err
		}
		for _, diagnostic := range diagnostics {
			fmt.Printf("%s: %s\nrepair: %s\n", diagnostic.Code, diagnostic.Message, diagnostic.Repair)
		}
		meta, metaErr := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id))
		if metaErr == nil && resolvedWorkspaceKind(meta) == "isolated" && !verifiedTaskWorktree(meta) {
			fmt.Printf("workspace-binding-invalid: Task %s isolated worktree is not registered or does not exist\nrepair: aiw wt repair\n", id)
		}
		return nil
	}
	if op == "recover" {
		if len(args) != 2 {
			return fmt.Errorf("usage: task workflow recover <task-id>")
		}
		state, err := workflow.NewStore("").RecoverPendingEvent(workflow.TaskID(id))
		if err != nil {
			return err
		}
		return projectWorkflowState(id, state)
	}
	if op == "repair" {
		if len(args) != 2 {
			return fmt.Errorf("usage: task workflow repair <task-id>")
		}
		return repairWorkflowState(id)
	}
	if op == "run" {
		execute, primary, provider, model, err := parseWorkflowRunArgsWithOverrides(args)
		if err != nil {
			return err
		}
		if !safeID(id) {
			return fmt.Errorf("invalid task id: %s", id)
		}
		return runWorkflowWithOverrides(id, execute, false, primary, provider, model)
	}
	if op == "supervise" {
		return runWorkflowSupervisor(args)
	}
	if op == "delivery" {
		if len(args) != 3 || (args[2] != string(workflow.DeliveryMerged) && args[2] != string(workflow.DeliveryDiscarded)) {
			return fmt.Errorf("usage: task workflow delivery <task-id> <merged|discarded>")
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
	if op == "delivery-failed" {
		if len(args) < 4 {
			return fmt.Errorf("usage: task workflow delivery-failed <task-id> <stage> <detail>")
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
		state, err = advanceWorkflow(id, meta, store)
	case "attempt":
		if len(args) < 3 {
			return fmt.Errorf("usage: task workflow attempt <task-id> <start|checkpoint> ...")
		}
		switch args[2] {
		case "start":
			if len(args) != 5 {
				return fmt.Errorf("usage: task workflow attempt <task-id> start <work-item-id> <attempt-id>")
			}
			state, err = store.StartAttempt(workflow.TaskID(id), workflow.Attempt{ID: workflow.AttemptID(args[4]), WorkItemID: workflow.WorkItemID(args[3]), Workspace: meta.Worktree})
		case "checkpoint":
			if len(args) != 5 {
				return fmt.Errorf("usage: task workflow attempt <task-id> checkpoint <attempt-id> <running|paused>")
			}
			state, err = store.CheckpointAttempt(workflow.TaskID(id), workflow.AttemptID(args[3]), workflow.AttemptState(args[4]))
		default:
			return fmt.Errorf("unknown attempt operation: %s", args[2])
		}
	case "evidence":
		if len(args) < 6 || len(args) > 7 {
			return fmt.Errorf("usage: task workflow evidence <task-id> <evidence-id> <work-item-id> <kind> <state> [reference]")
		}
		reference := ""
		if len(args) == 7 {
			reference = args[6]
		}
		state, err = store.RecordEvidence(workflow.TaskID(id), workflow.Evidence{ID: workflow.EvidenceID(args[2]), WorkItemID: workflow.WorkItemID(args[3]), Kind: workflow.EvidenceKind(args[4]), State: workflow.EvidenceState(args[5]), Reference: reference})
	case "gate":
		if len(args) != 4 {
			return fmt.Errorf("usage: task workflow gate <task-id> <gate-id> <resolved|waived>")
		}
		state, err = store.ResolveGate(workflow.TaskID(id), workflow.GateID(args[2]), workflow.GateState(args[3]))
	case "complete":
		if len(args) != 3 {
			return fmt.Errorf("usage: task workflow complete <task-id> <work-item-id>")
		}
		state, err = store.CompleteWorkItem(workflow.TaskID(id), workflow.WorkItemID(args[2]))
	case "retry-policy":
		limit, parseErr := strconv.Atoi(args[3])
		if parseErr != nil {
			return fmt.Errorf("retry limit must be a whole number")
		}
		state, err = store.SetRetryPolicy(workflow.TaskID(id), workflow.WorkItemID(args[2]), workflow.RetryPolicy{MaxAttempts: limit})
	case "reopen":
		state, err = store.ReopenWorkItem(workflow.TaskID(id), workflow.WorkItemID(args[2]), strings.Join(args[3:], " "))
	case "force-close":
		if err := preflightForceCloseDelivery(meta, workflow.DeliveryState(args[2])); err != nil {
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
		execution, executeErr := runFocusedTestCommand(meta, state, store, workflow.AttemptID(args[2]), nil, osFocusedTestProcessRunner{})
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

// runFocusedTestCommand is the command adapter for the controlled
// focused-test path. Keeping the resolver, network enforcer, and process
// runner explicit makes the no-process-before-validation boundary testable
// without allowing callers to supply a command.
func runFocusedTestCommand(meta taskx.TaskMeta, state workflow.RuntimeState, store *workflow.Store, attemptID workflow.AttemptID, enforcer FocusedTestNetworkEnforcer, process FocusedTestProcessRunner) (FocusedTestExecution, error) {
	run, err := ResolveFocusedTestRun(meta, state, attemptID)
	if err != nil {
		return FocusedTestExecution{}, err
	}
	return ExecuteFocusedTest(store, run, enforcer, process)
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
				return false, false, "", "", fmt.Errorf("usage: task workflow run <task-id> [--execute] [--primary] [--provider NAME] [--model MODEL]")
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
	}
	state, err := syncWorkflowChecklist(id, store)
	if err != nil {
		return err
	}
	outcome := workflow.NextRunnerOutcome(state)
	if outcome.Kind == "" {
		state, err = advanceWorkflow(id, meta, store)
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
	if err := ensureRunnerHandoff(id, outcome.Request); err != nil {
		return err
	}
	return runBoundedTaskTurn(id, supervised, provider, model)
}

// runBoundedTaskTurn is the canonical handoff from automatic workflow
// execution to the one-turn Task Agent command. Keeping this boundary here
// ensures Runner and Supervisor never start an interactive Chat or a legacy
// agent command surface.
func runBoundedTaskTurn(id string, supervised bool, overrides ...string) error {
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
	return runTaskAgent(args)
}

func ensureAutomatedWorkspace(id string, meta taskx.TaskMeta, primary bool) (taskx.TaskMeta, error) {
	kind := resolvedWorkspaceKind(meta)
	if primary {
		if kind != "primary" {
			return taskx.TaskMeta{}, fmt.Errorf("--primary requires Task %s to be bound to the primary workspace, found %s", id, kind)
		}
		isPrimary, primaryPath, err := gitx.IsPrimaryWorktree()
		if err != nil {
			return taskx.TaskMeta{}, err
		}
		if !isPrimary {
			return taskx.TaskMeta{}, fmt.Errorf("--primary requires the current workspace to be the primary worktree: %s", primaryPath)
		}
		fmt.Printf("workflow workspace: primary task=%s\n", id)
		return meta, nil
	}
	if kind == "isolated" {
		if !verifiedTaskWorktree(meta) {
			return taskx.TaskMeta{}, fmt.Errorf("Task %s has an invalid isolated worktree binding", id)
		}
		fmt.Printf("workflow workspace: isolated task=%s worktree=%s\n", id, meta.Worktree)
		return meta, nil
	}
	if kind != "primary" {
		return taskx.TaskMeta{}, fmt.Errorf("Task %s workspace is %s; repair or explicitly bind it before automated execution", id, kind)
	}
	isPrimary, primaryPath, err := gitx.IsPrimaryWorktree()
	if err != nil {
		return taskx.TaskMeta{}, err
	}
	if !isPrimary {
		return taskx.TaskMeta{}, fmt.Errorf("automatic isolation must start from the primary worktree: %s", primaryPath)
	}
	if err := addTaskWorktree(id); err != nil {
		return taskx.TaskMeta{}, fmt.Errorf("create automatic worktree: %w", err)
	}
	meta, err = taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id))
	if err != nil {
		return taskx.TaskMeta{}, err
	}
	if resolvedWorkspaceKind(meta) != "isolated" || !verifiedTaskWorktree(meta) {
		return taskx.TaskMeta{}, fmt.Errorf("automatic worktree for Task %s was not verified", id)
	}
	fmt.Printf("workflow workspace: isolated task=%s worktree=%s\n", id, meta.Worktree)
	return meta, nil
}

func verifiedTaskWorktree(meta taskx.TaskMeta) bool {
	worktree := strings.TrimSpace(meta.Worktree)
	if worktree == "" || worktree == "." || !gitx.WorktreeRegistered(worktree) {
		return false
	}
	if !filepath.IsAbs(worktree) {
		root, err := gitx.PrimaryWorktree()
		if err != nil {
			return false
		}
		worktree = filepath.Join(root, filepath.FromSlash(worktree))
	}
	return fsx.Exists(worktree)
}

func syncWorkflowChecklist(id string, store *workflow.Store) (workflow.RuntimeState, error) {
	path, err := workflowChecklistPath(id)
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	items, err := taskx.ReadWorkflowChecklist(path)
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	candidates := make([]workflow.ChecklistCandidate, len(items))
	for index, item := range items {
		candidates[index] = workflow.ChecklistCandidate{Item: item.Number, Title: item.Title, Completed: item.Completed}
	}
	return store.SyncChecklist(workflow.TaskID(id), candidates, taskx.ChecklistFingerprint(items))
}

// workflowChecklistPath selects the Task's bound workspace as the source of
// truth while automated execution is isolated. Workflow state remains stored
// in the primary workspace, but an Agent's checklist updates live in its
// isolated worktree until delivery merges that branch.
func workflowChecklistPath(id string) (string, error) {
	meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id))
	if err != nil {
		return "", err
	}
	if resolvedWorkspaceKind(meta) != "isolated" {
		return filepath.Join(taskx.TaskDir(id), "tasks.md"), nil
	}
	worktree := strings.TrimSpace(meta.Worktree)
	if worktree == "" {
		return "", fmt.Errorf("Task %s has no isolated worktree", id)
	}
	if !filepath.IsAbs(worktree) {
		root, err := gitx.PrimaryWorktree()
		if err != nil {
			return "", err
		}
		worktree = filepath.Join(root, filepath.FromSlash(worktree))
	}
	return filepath.Join(worktree, taskx.TaskDir(id), "tasks.md"), nil
}

func advanceWorkflow(id string, meta taskx.TaskMeta, store *workflow.Store) (workflow.RuntimeState, error) {
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
	state, err = store.StartAttempt(workflow.TaskID(id), workflow.Attempt{ID: attemptID, WorkItemID: item.ID, SessionID: meta.Session, Workspace: meta.Worktree})
	if err != nil {
		return workflow.RuntimeState{}, err
	}
	request := &workflow.PreparedAgentRequest{TaskID: workflow.TaskID(id), WorkItemID: item.ID, AttemptID: attemptID, SessionID: meta.Session, Workspace: meta.Worktree}
	return store.RecordAutomation(workflow.TaskID(id), state.Automation.PlanFingerprint, workflow.AutomationCursor{Result: "agent-request-prepared", Detail: string(item.ID)}, request)
}

func validateWorkflowArgs(op string, args []string) error {
	if !safeID(args[1]) {
		return fmt.Errorf("invalid task id: %s", args[1])
	}
	switch op {
	case "plan", "sync", "advance":
		if len(args) != 2 {
			return fmt.Errorf("usage: task workflow plan <task-id>")
		}
	case "attempt":
		if len(args) != 5 || (args[2] != "start" && args[2] != "checkpoint") {
			return fmt.Errorf("usage: task workflow attempt <task-id> <start|checkpoint> <id> <id-or-state>")
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
			return fmt.Errorf("usage: task workflow evidence <task-id> <evidence-id> <work-item-id> <kind> <state> [reference]")
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
			return fmt.Errorf("usage: task workflow gate <task-id> <gate-id> <resolved|waived>")
		}
		if strings.TrimSpace(args[2]) == "" || (args[3] != string(workflow.GateResolved) && args[3] != string(workflow.GateWaived)) {
			return fmt.Errorf("invalid gate resolution")
		}
	case "complete":
		if len(args) != 3 {
			return fmt.Errorf("usage: task workflow complete <task-id> <work-item-id>")
		}
		if _, err := workflow.ParseWorkItemID(args[2]); err != nil {
			return err
		}
	case "retry-policy":
		if len(args) != 4 {
			return fmt.Errorf("usage: task workflow retry-policy <task-id> <work-item-id> <1-5>")
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
			return fmt.Errorf("usage: task workflow reopen <task-id> <work-item-id> <reason>")
		}
		if _, err := workflow.ParseWorkItemID(args[2]); err != nil {
			return err
		}
		if strings.TrimSpace(strings.Join(args[3:], " ")) == "" {
			return fmt.Errorf("reopen reason is required")
		}
	case "force-close":
		if len(args) < 4 || (args[2] != string(workflow.DeliveryMerged) && args[2] != string(workflow.DeliveryDiscarded)) {
			return fmt.Errorf("usage: task workflow force-close <task-id> <merged|discarded> <reason>")
		}
		if strings.TrimSpace(strings.Join(args[3:], " ")) == "" {
			return fmt.Errorf("force-close reason is required")
		}
	case "focused-test":
		if len(args) != 3 || strings.TrimSpace(args[2]) == "" {
			return fmt.Errorf("usage: task workflow focused-test <task-id> <attempt-id>")
		}
	default:
		return fmt.Errorf("unknown workflow operation: %s", op)
	}
	return nil
}

func knownEvidenceKind(value string) bool {
	return value == string(workflow.EvidenceStaticReview) || value == string(workflow.EvidenceCommand) || value == string(workflow.EvidenceApproval) || value == string(workflow.EvidenceManual)
}

func knownEvidenceState(value string) bool {
	return value == string(workflow.EvidencePending) || value == string(workflow.EvidencePassed) || value == string(workflow.EvidenceFailed) || value == string(workflow.EvidenceWaived)
}

func compatibleWorkflow(id string) (taskx.TaskMeta, *workflow.Store, error) {
	meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id))
	if err != nil {
		return taskx.TaskMeta{}, nil, err
	}
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(taskx.WorkflowRuntimeFromMeta(meta)); err != nil {
		return taskx.TaskMeta{}, nil, err
	}
	return meta, store, nil
}

func projectWorkflowState(id string, state workflow.RuntimeState) error {
	meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id))
	if err != nil {
		return err
	}
	return reportWorkflowState(meta, state)
}

// reportWorkflowState renders Core-owned state without projecting it into the
// Task metadata or OpenSpec checklist. Runtime storage is the sole durable
// owner of execution summaries, evidence, leases, retries, and diagnostics.
func reportWorkflowState(meta taskx.TaskMeta, state workflow.RuntimeState) error {
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

func reportAuthoredChecklist(meta taskx.TaskMeta, state workflow.RuntimeState, terminal *ui.Terminal) error {
	path, err := workflowChecklistPath(meta.ID)
	if err != nil {
		return fmt.Errorf("locate authored checklist: %w", err)
	}
	items, err := taskx.ReadWorkflowChecklist(path)
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

func printDeliveryGuidance(meta taskx.TaskMeta, state workflow.RuntimeState) {
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
	terminal.Line(fmt.Sprintf("  2. If you edited files manually, commit them with `aiw wt commit %s \"message\"`.", meta.ID))
	terminal.Line(fmt.Sprintf("  3. Merge the task branch with `aiw wt pull %s`.", meta.ID))
	terminal.Line("  4. Remove the merged worktree with `aiw wt rm " + meta.ID + " --delete-branch`.")
}

func repairWorkflowState(id string) error {
	meta, store, err := compatibleWorkflow(id)
	if err != nil {
		return err
	}
	path, err := workflowChecklistPath(id)
	if err != nil {
		return err
	}
	items, err := taskx.ReadWorkflowChecklist(path)
	if err != nil {
		return err
	}
	candidates := make([]workflow.ChecklistCandidate, len(items))
	for index, item := range items {
		candidates[index] = workflow.ChecklistCandidate{Item: item.Number, Title: item.Title, Completed: item.Completed}
	}
	state, err := store.RepairChecklist(workflow.TaskID(id), candidates)
	if err != nil {
		return err
	}
	return reportWorkflowState(meta, state)
}

func printWorkflowHelp() {
	fmt.Print(`AIW workflow manages a Task's checklist, execution state, evidence, and recovery.

Usage:
  aiw workflow <command> <task-id> [options]
  aiw task workflow <command> <task-id> [options]

Typical flow:
  1. aiw workflow plan <task-id>       Create or reconcile Work Items from tasks.md.
  2. aiw workflow run <task-id>        Preview the next action without executing it.
  3. aiw workflow run <task-id> --execute
                                         Execute one prepared Work Item.

Plan and execute:
  plan <task-id>                       Create or reconcile Work Items from tasks.md.
  sync <task-id>                       Synchronize checklist completion into Workflow Core.
  advance <task-id>                    Prepare the next ready Work Item without executing it.
  run <task-id> [--execute] [--primary] [--provider NAME] [--model MODEL]
                                         Preview the next action, or execute one Work Item.
  supervise <task-id> <start|status|stop>
                                         Start, inspect, or stop managed execution.

Record progress:
  attempt <task-id> start <work-item-id> <attempt-id>
  attempt <task-id> checkpoint <attempt-id> <running|paused>
  evidence <task-id> <evidence-id> <work-item-id> <kind> <state> [reference]
  gate <task-id> <gate-id> <resolved|waived>
  complete <task-id> <work-item-id>
  delivery <task-id> <merged|discarded>
  retry-policy <task-id> <work-item-id> <1-5>
                                         Set the automatic Attempt limit (default: 3).
  reopen <task-id> <work-item-id> <reason>
                                         Reset an exhausted Work Item after recording a reason.
  force-close <task-id> <merged|discarded> <reason>
                                         Cancel managed execution and record terminal delivery.
  focused-test <task-id> <attempt-id>   Run the selected approved focused check.

Inspect and recover:
  diagnose <task-id>                   Show Workflow Core diagnostics and repair guidance.
  recover <task-id>                    Re-project a committed pending transition.
  repair <task-id>                     Re-project state and resolve recorded projection repairs.

Execution and workspace rules:
  - Without --execute, run only previews the next action.
  - --execute delegates one prepared Work Item to aiw turn.
  - Automated execution uses an isolated .wt/<task-id> worktree by default.
  - --primary is an explicit opt-out and only works for Tasks bound to the primary workspace.
  - Git delivery, merge, cleanup, and archive remain separate operations.
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

Examples:
  aiw workflow plan payment-retry
  aiw workflow run payment-retry
  aiw workflow run payment-retry --execute
  aiw workflow retry-policy payment-retry wi-0001 3
  aiw workflow reopen payment-retry wi-0001 "authorization granted"
  aiw workflow force-close payment-retry discarded "superseded by a new task"
  aiw workflow diagnose payment-retry
  aiw workflow focused-test payment-retry attempt-123
`)
}
