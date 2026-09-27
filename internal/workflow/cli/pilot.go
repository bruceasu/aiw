package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	"aiw/internal/ai"
	"aiw/internal/session"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow"
	"aiw/internal/workflow/execution"
)

const codexPilotHostID = "codex-cli-pilot-v1"

type codexPilotEvidence struct {
	Version int `json:"version"`
	TaskID workflow.TaskID `json:"task_id"`
	SourceSchema int `json:"source_schema"`
	SourceEventSequence uint64 `json:"source_event_sequence"`
	WorkItemID workflow.WorkItemID `json:"work_item_id"`
	Workspace string `json:"workspace"`
	Selection workflow.AISelection `json:"selection"`
	RoutingPolicy workflow.GenerationRoutingPolicy `json:"routing_policy"`
	RoutingPlan workflow.ActorReference `json:"routing_plan"`
	Budget workflow.TaskUsageBudget `json:"usage_budget"`
	PreparedAt string `json:"prepared_at"`
}

type codexPilotAssessment struct {
	State workflow.RuntimeState
	Item workflow.WorkItem
	Plan workflow.RoutingPlan
	Selection workflow.AISelection
	Policy workflow.GenerationRoutingPolicy
	Budget workflow.TaskUsageBudget
	Evidence codexPilotEvidence
	Blockers []string
}

func runCodexPilot(args []string) error {
	if len(args) != 2 { return errors.New("usage: wf pilot <task-id> <status|activate|run>") }
	id, operation := args[0], args[1]
	if !taskAdapter.SafeID(id) { return fmt.Errorf("invalid task id: %s", id) }
	store := workflow.NewStore("")
	switch operation {
	case "status":
		return showCodexPilotStatus(id, store)
	case "activate":
		return activateCodexPilot(id, store)
	case "run":
		return runCodexPilotTask(id, store)
	default:
		return fmt.Errorf("unknown pilot operation %q; use status, activate, or run", operation)
	}
}

func assessCodexPilot(id string, store *workflow.Store, state workflow.RuntimeState) codexPilotAssessment {
	a := codexPilotAssessment{State: state}
	if state.SchemaVersion == workflow.DurableSchemaVersion {
		a.Blockers = append(a.Blockers, "Task is already Schema 10; use activation reconciliation/status, not fresh activation")
		return a
	}
	if state.SchemaVersion != workflow.SchemaVersion { a.Blockers = append(a.Blockers, fmt.Sprintf("expected Schema 9, found Schema %d", state.SchemaVersion)) }
	if len(state.WorkItems) != 1 {
		a.Blockers = append(a.Blockers, fmt.Sprintf("requires exactly one mapped WorkItem; found %d", len(state.WorkItems)))
	} else {
		a.Item = state.WorkItems[0]
		if a.Item.State != workflow.WorkItemReady { a.Blockers = append(a.Blockers, fmt.Sprintf("WorkItem %s must be ready; found %s", a.Item.ID, a.Item.State)) }
	}
	if len(state.Attempts) != 0 { a.Blockers = append(a.Blockers, "Task already has Attempt history") }
	if state.WriteLease != nil { a.Blockers = append(a.Blockers, "Task has an active workspace write lease") }
	if state.PendingEvent != nil { a.Blockers = append(a.Blockers, "Task has an unconfirmed state event") }
	if state.Automation.PreparedRequest != nil { a.Blockers = append(a.Blockers, "Task has a prepared legacy Agent request") }
	if state.Automation.Supervisor.LeaseID != "" { a.Blockers = append(a.Blockers, "Task has an active Supervisor lease") }
	if state.Automation.Supervisor.StoppedAt != "" { a.Blockers = append(a.Blockers, "Task has an explicit legacy Stop") }
	for _, gate := range state.Gates { if gate.State == workflow.GateOpen { a.Blockers = append(a.Blockers, "Task has open Gate "+string(gate.ID)) } }
	if strings.TrimSpace(state.Task.Workspace) == "" { a.Blockers = append(a.Blockers, "Task workspace is not bound") }
	if runtime.GOOS != "windows" { a.Blockers = append(a.Blockers, "durable Schema 10 Task locking is unavailable on this platform") }
	if locked, err := store.TaskLocked(state.Task.ID); err != nil { a.Blockers = append(a.Blockers, "cannot verify Task lock: "+err.Error())
	} else if locked { a.Blockers = append(a.Blockers, "another writer holds the Task lock") }

	plan, err := store.LoadRoutingPlan(state.Task.ID)
	if err != nil {
		a.Blockers = append(a.Blockers, "frozen routing/compile plan unavailable: "+err.Error())
	} else {
		a.Plan = plan
		if !plan.Compile.Available { a.Blockers = append(a.Blockers, "compile plan unavailable: "+plan.Compile.Reason) }
		if a.Item.ID != "" {
			selection, selectionErr := resolveSupervisedAISelection(store, state.Task.ID, state, a.Item.ID, "", "")
			if selectionErr != nil {
				a.Blockers = append(a.Blockers, "Codex selection unavailable: "+selectionErr.Error())
			} else if !strings.EqualFold(selection.Provider, "codex") {
				a.Blockers = append(a.Blockers, "selected Coder provider is not Codex CLI")
			} else {
				a.Selection = *selection
				policy, policyErr := codexPilotGenerationPolicy(*selection)
				if policyErr != nil {
					a.Blockers = append(a.Blockers, "Codex profile policy unavailable: "+policyErr.Error())
				} else {
					a.Policy = policy
				}
			}
		}
	}
	defaults, budgetErr := ai.LoadUsageBudgetDefaults()
	if budgetErr != nil {
		a.Blockers = append(a.Blockers, "positive input/output Token limits are required in aiw.toml: "+budgetErr.Error())
	} else {
		a.Budget = workflow.TaskUsageBudget{InputTokenLimit: defaults.InputTokens, OutputTokenLimit: defaults.OutputTokens}
	}
	a.Evidence = codexPilotEvidence{Version: 1, TaskID: state.Task.ID, SourceSchema: state.SchemaVersion,
		SourceEventSequence: state.LastEventSequence, WorkItemID: a.Item.ID, Workspace: pilotExecutionWorkspace(state),
		Selection: a.Selection, RoutingPolicy: a.Policy, RoutingPlan: a.Plan.Reference(), Budget: a.Budget,
		PreparedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	return a
}

// codexPilotGenerationPolicy snapshots the selected profile and only higher
// configured Codex levels. The default is first, followed by deterministic
// level/name order; profiles from other providers never enter the pilot.
func codexPilotGenerationPolicy(selected workflow.AISelection) (workflow.GenerationRoutingPolicy, error) {
	profiles, err := ai.LoadProfiles()
	if err != nil { return workflow.GenerationRoutingPolicy{}, err }
	ordered := make([]ai.Profile, 0, len(profiles))
	for _, profile := range profiles {
		if profile.Level >= selected.Level && strings.EqualFold(profile.Provider, "codex") { ordered = append(ordered, profile) }
	}
	if configured, ok := profiles[selected.Profile]; !ok || !strings.EqualFold(configured.Provider, "codex") {
		return workflow.GenerationRoutingPolicy{}, fmt.Errorf("selected profile %q is not a configured Codex profile", selected.Profile)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Level != ordered[j].Level { return ordered[i].Level < ordered[j].Level }
		return ordered[i].Name < ordered[j].Name
	})
	order := make([]string, 0, len(ordered))
	for _, profile := range ordered { order = append(order, profile.Name) }
	policy, err := execution.ConfiguredGenerationPolicy(selected.Profile, order, "", "")
	if err != nil { return policy, err }
	for index, profile := range ordered {
		resolved, config, err := ai.ResolveProfile(profile.Name)
		if err != nil { return workflow.GenerationRoutingPolicy{}, err }
		digestSource := strings.Join([]string{resolved.Name, config.Name, config.Model, fmt.Sprint(resolved.Level), resolved.ReasoningIntensity, config.BaseURL, config.Command, config.CodexCommand, config.CopilotCommand}, "\n")
		digest := sha256.Sum256([]byte(digestSource))
		selection := workflow.AISelection{Profile: resolved.Name, Provider: config.Name, Model: config.Model,
			Digest: hex.EncodeToString(digest[:]), Level: resolved.Level,
			ReasoningIntensity: resolved.ReasoningIntensity}
		if profile.Name == selected.Profile { selection = selected }
		policy.Profiles[index] = selection
	}
	return policy, nil
}

func showCodexPilotStatus(id string, store *workflow.Store) error {
	state, err := store.Load(workflow.TaskID(id))
	if err != nil { return fmt.Errorf("load pilot Task: %w", err) }
	if state.SchemaVersion == workflow.DurableSchemaVersion {
		fmt.Printf("Codex pilot task=%s schema=10\n", id)
		if state.Protocol != nil && state.Protocol.Usage != nil && state.Protocol.Usage.Budget != nil { fmt.Println("usage budget: configured")
		} else { fmt.Println("usage budget: missing; activation reconciliation required") }
		if state.Protocol != nil && state.Protocol.Stop != nil { fmt.Printf("Stop: %s\n", state.Protocol.Stop.Reason) }
		fmt.Printf("execution: %s\n", pilotExecutionStatus(state))
		return nil
	}
	a := assessCodexPilot(id, store, state)
	fmt.Printf("Codex pilot task=%s schema=%d\n", id, state.SchemaVersion)
	fmt.Printf("work items=%d attempts=%d workspace=%s\n", len(state.WorkItems), len(state.Attempts), state.Task.Workspace)
	if a.Selection.Provider != "" { fmt.Printf("coder=%s/%s profile=%s\n", a.Selection.Provider, a.Selection.Model, a.Selection.Profile) }
	if a.Plan.TaskID != "" { fmt.Printf("compile plan available=%t targets=%d\n", a.Plan.Compile.Available, len(a.Plan.Compile.Targets)) }
	if a.Budget.InputTokenLimit > 0 { fmt.Printf("token limits input=%d output=%d\n", a.Budget.InputTokenLimit, a.Budget.OutputTokenLimit) }
	if len(a.Blockers) == 0 { fmt.Println("preflight: ready for explicit activation")
	} else { fmt.Printf("preflight: blocked (%d)\n", len(a.Blockers)); for _, blocker := range a.Blockers { fmt.Printf("- %s\n", blocker) } }
	fmt.Printf("execution: %s\n", pilotExecutionStatus(state))
	return nil
}

func pilotExecutionStatus(state workflow.RuntimeState) string {
	if state.Protocol == nil { return "activation reconciliation required" }
	if state.Protocol.Stop != nil { return "stopped: " + state.Protocol.Stop.Reason }
	if state.Protocol.Usage == nil || state.Protocol.Usage.Budget == nil { return "blocked: Task Token budget is not configured" }
	if len(state.Protocol.Items) == 0 { return "ready: run the current Coder stage" }
	item := state.Protocol.Items[0]
	switch item.Phase {
	case workflow.PhaseCoder:
		if pilotCurrentRequestConsumed(state, item.CurrentRequest) { return "Coder terminal result is preserved; the pilot will not auto-retry it. Inspect the recorded result before deciding what to do next." }
		return "Coder pending or awaiting reconciliation; run the same request with `aiw wf pilot <task-id> run`"
	case workflow.PhaseReport: return "validate the terminal Coder report with `aiw wf pilot <task-id> run`"
	case workflow.PhaseCompile:
		if pilotCurrentRequestConsumed(state, item.CurrentRequest) { return "compile result is preserved; the pilot will not repeat it. Inspect the compile evidence before taking further action." }
		return "compile pending or awaiting reconciliation; run the same request with `aiw wf pilot <task-id> run`"
	case workflow.PhaseTester: return "stopped before Tester; WorkItem is not accepted"
	default: return "pilot does not support phase " + string(item.Phase)
	}
}

func pilotCurrentRequestConsumed(state workflow.RuntimeState, requestID string) bool {
	if state.Protocol == nil || requestID == "" { return false }
	for _, record := range state.Protocol.Requests { if record.Request.ID == requestID { return record.Consumed && record.Result != nil } }
	return false
}

func runCodexPilotTask(id string, store *workflow.Store) error {
	state, err := store.Load(workflow.TaskID(id))
	if err != nil { return fmt.Errorf("load pilot Task: %w", err) }
	if state.SchemaVersion != workflow.DurableSchemaVersion { return errors.New("Schema 10 activation is required; run `aiw wf pilot <task-id> activate` first") }
	evidence, ref, err := loadCodexPilotEvidence(id, store, state)
	if err != nil { return err }
	host := &execution.CodexStageHost{ID: codexPilotHostID, Store: store, Sessions: session.NewStore("")}
	verify := func(current workflow.RuntimeState, refs []workflow.ActorReference) error {
		if current.Protocol == nil || len(refs) != 1 || refs[0] != ref { return errors.New("pilot activation identity differs") }
		var saved codexPilotEvidence
		if err := store.ReadExecutionArtifact(workflow.TaskID(id), ref, &saved); err != nil { return err }
		if !reflect.DeepEqual(saved, evidence) { return errors.New("pilot activation evidence changed") }
		return nil
	}
	if err := execution.ConnectCodexPilot(store, host, evidence.RoutingPolicy, verify); err != nil { return err }
	if state.Protocol.Stop != nil { return fmt.Errorf("Codex pilot stopped: %s", state.Protocol.Stop.Reason) }
	if state.Protocol.Usage == nil || state.Protocol.Usage.Budget == nil { return errors.New("Codex pilot Token budget is missing; reconcile activation before running") }
	if len(state.Protocol.Items) == 0 {
		if len(state.WorkItems) != 1 { return errors.New("Codex pilot requires exactly one WorkItem") }
		attemptID := workflow.AttemptID(fmt.Sprintf("pilot-%d", time.Now().UTC().UnixNano()))
		state, err = store.BeginExecution(workflow.TaskID(id), state.StateRevision, workflow.Attempt{ID: attemptID, WorkItemID: evidence.WorkItemID, SessionID: string(attemptID), Workspace: pilotExecutionWorkspace(state)})
		if err != nil { return fmt.Errorf("begin Schema 10 Coder Attempt: %w", err) }
	}
	if len(state.Protocol.Items) != 1 || state.Protocol.Items[0].WorkItemID != evidence.WorkItemID { return errors.New("pilot Task has an unexpected execution cursor") }
	item := state.Protocol.Items[0]
	if item.Phase == workflow.PhaseTester { fmt.Printf("Codex pilot stopped before Tester; WorkItem %s remains unaccepted\n", item.WorkItemID); return nil }
	if item.Phase == workflow.PhaseCoder {
		requestID := item.CurrentRequest
		if requestID == "" {
			requestID, err = preparePilotCoderStage(id, store, state, item, evidence)
			if err != nil { return err }
		}
		if err := execution.RunVerificationStage(context.Background(), store, workflow.TaskID(id), requestID, host); err != nil { return fmt.Errorf("Coder stage %s: %w", requestID, err) }
		state, err = store.Load(workflow.TaskID(id))
		if err != nil { return err }
		item = state.Protocol.Items[0]
	}
	if item.Phase == workflow.PhaseReport {
		requestID := item.CurrentRequest
		result, err := store.ReadStageResult(workflow.TaskID(id), requestID)
		if err != nil { return fmt.Errorf("read terminal Coder result: %w", err) }
		if result.Status != "passed" || len(result.Evidence) < 2 { return errors.New("Coder result has no validated terminal report reference") }
		state, err = store.ValidateStageReport(workflow.TaskID(id), state.StateRevision, item.WorkItemID, result.Evidence[1])
		if err != nil { return fmt.Errorf("validate implementation report: %w", err) }
		item = state.Protocol.Items[0]
	}
	if item.Phase == workflow.PhaseCompile {
		requestID := item.CurrentRequest
		if requestID == "" || !pilotStageRequestIsCompile(state, requestID) {
			requestID, err = preparePilotCompileStage(id, store, state, item, evidence)
			if err != nil { return err }
		}
		if err := execution.RunVerificationStage(context.Background(), store, workflow.TaskID(id), requestID, host); err != nil { return fmt.Errorf("compile stage %s: %w", requestID, err) }
		state, err = store.Load(workflow.TaskID(id))
		if err != nil { return err }
		if len(state.Protocol.Items) == 1 && state.Protocol.Items[0].Phase == workflow.PhaseTester {
			fmt.Printf("Codex pilot compile completed; stopped before Tester. WorkItem %s is not accepted.\n", item.WorkItemID)
			return projectWorkflowState(id, state)
		}
		result, resultErr := store.ReadStageResult(workflow.TaskID(id), requestID)
		if resultErr != nil { return fmt.Errorf("compile did not advance the Task; preserved request %s: %w", requestID, resultErr) }
		return fmt.Errorf("frozen compile did not pass (status=%s); evidence is preserved and the pilot will not dispatch another model call", result.Status)
	}
	return fmt.Errorf("Codex pilot stopped at %s; inspect `aiw wf %s status`", item.Phase, id)
}

func preparePilotCoderStage(id string, store *workflow.Store, state workflow.RuntimeState, item workflow.ItemExecution, evidence codexPilotEvidence) (string, error) {
	var workItem *workflow.WorkItem
	for index := range state.WorkItems { if state.WorkItems[index].ID == item.WorkItemID { workItem = &state.WorkItems[index]; break } }
	if workItem == nil { return "", errors.New("pilot WorkItem is missing from Core state") }
	line, checklistPath, err := pilotChecklistLine(id, workItem.Checklist.Item)
	if err != nil { return "", err }
	contextValue := struct { WorkItem workflow.WorkItem `json:"work_item"`; ChecklistPath string `json:"checklist_path"`; ChecklistLine string `json:"checklist_line"` }{*workItem, checklistPath, line}
	contextRef, err := store.PersistVerificationArtifact(workflow.TaskID(id), "verification-pilot-work-item", contextValue)
	if err != nil { return "", err }
	policy := workflow.TestExecutionPolicy{SchemaVersion: 1, Version: "trusted-local-codex-pilot-v1", TaskID: workflow.TaskID(id),
		WorkspaceDigest: workflow.WorkspaceBindingDigest(state), InputScope: []string{"."}, TestPaths: []string{}, FixturePaths: []string{}, Checks: []workflow.VerificationCheck{}, DescriptiveDocuments: []workflow.ActorReference{}}
	policyRef, err := store.PersistVerificationArtifact(workflow.TaskID(id), "verification-pilot-policy", policy)
	if err != nil { return "", err }
	profiles := evidence.RoutingPolicy
	route, err := workflow.RouteGeneration(state, workflow.GenerationRoutingContext{TaskID: workflow.TaskID(id), WorkItemID: item.WorkItemID, Actor: workflow.ActorCoder, Input: contextRef, Requirements: []string{line}}, profiles, nil)
	if err != nil { return "", fmt.Errorf("route frozen Codex profile: %w", err) }
	if route.Index < 0 || route.Index >= len(route.Models) { return "", errors.New("Codex routing selected an invalid profile tier") }
	selection := route.Models[route.Index]
	attemptID := item.AttemptID
	sessionID := string(attemptID)
	status, err := ensurePilotSession(sessionID, id, item.WorkItemID, attemptID, pilotExecutionWorkspace(state), selection)
	if err != nil { return "", err }
	prepared := workflow.PreparedAgentRequest{TaskID: workflow.TaskID(id), WorkItemID: item.WorkItemID, AttemptID: attemptID,
		SessionID: sessionID, ExpectedSessionTurn: status.Session.LastTurn+1, Workspace: pilotExecutionWorkspace(state), AISelection: &selection}
	requestID := workflow.ExecutionRequestID(prepared)
	root, err := filepath.Abs(prepared.Workspace)
	if err != nil { return "", err }
	toolchain, err := workflow.ValidationToolchainIdentity()
	if err != nil { return "", err }
	bindings := []workflow.ActorReference{evidence.RoutingPlan, policyRef}
	inputs, err := workflow.CaptureValidationInputs(root, policy.InputScope, bindings, toolchain)
	if err != nil { return "", fmt.Errorf("freeze Coder workspace inputs: %w", err) }
	prompt := "Implement only this mapped WorkItem. Treat the quoted checklist line as the requirement. Do not run tests, deliver, accept, or clean up. Report changed files, behavior, and compile risks.\n\nWorkItem: " + workItem.Title + "\nChecklist: " + line
	input := workflow.ExecutionInput{SchemaVersion: 1, RequestID: requestID, TaskID: workflow.TaskID(id), WorkItemID: item.WorkItemID,
		AttemptID: attemptID, SessionID: sessionID, Turn: prepared.ExpectedSessionTurn, Actor: workflow.ActorCoder, Workspace: prepared.Workspace,
		AISelection: &selection, AllowedPaths: []string{"."}, Sources: []workflow.InputSource{{Kind: "work-item", Path: checklistPath + "#" + workItem.Checklist.Item,
		Required: true, Status: "loaded", Content: line, SHA256: pilotDigest([]byte(line))}}, WorkspaceInputs: &inputs, Prompt: prompt}
	inputRef, err := store.PersistExecutionInput(prepared, input)
	if err != nil { return "", err }
	request := workflow.StageRequest{ActorRequest: workflow.ActorRequest{SchemaVersion: workflow.ActorContractSchemaVersion, ID: requestID, Actor: workflow.ActorCoder,
		TaskID: workflow.TaskID(id), WorkItemID: item.WorkItemID, AttemptID: attemptID, Workspace: prepared.Workspace, PreparedAt: time.Now().UTC().Format(time.RFC3339Nano)},
		Phase: workflow.PhaseCoder, SessionID: sessionID, Turn: prepared.ExpectedSessionTurn, Model: &selection, Route: &route, AllowedPaths: []string{"."},
		Input: inputRef, InputDigest: inputs.Digest(), Plan: evidence.RoutingPlan, Policy: policyRef}
	updated, err := store.PrepareStage(workflow.TaskID(id), state.StateRevision, request)
	if err != nil { return "", fmt.Errorf("prepare frozen Coder stage: %w", err) }
	return updated.Protocol.Items[0].CurrentRequest, nil
}

func preparePilotCompileStage(id string, store *workflow.Store, state workflow.RuntimeState, item workflow.ItemExecution, evidence codexPilotEvidence) (string, error) {
	var coder *workflow.StageRecord
	for index := range state.Protocol.Requests { record := &state.Protocol.Requests[index]; if record.Request.WorkItemID == item.WorkItemID && record.Request.Phase == workflow.PhaseCoder { coder = record; break } }
	if coder == nil || coder.Result == nil { return "", errors.New("compile requires the original terminal Coder stage") }
	input := workflow.ExecutionInput{}
	if err := store.ReadExecutionArtifact(workflow.TaskID(id), coder.Request.Input, &input); err != nil { return "", err }
	if input.WorkspaceInputs == nil { return "", errors.New("Coder input has no validation baseline") }
	root, err := filepath.Abs(pilotExecutionWorkspace(state))
	if err != nil { return "", err }
	toolchain, err := workflow.ValidationToolchainIdentity()
	if err != nil { return "", err }
	current, err := workflow.CaptureValidationInputs(root, input.WorkspaceInputs.Scope, input.WorkspaceInputs.Bindings, toolchain)
	if err != nil { return "", err }
	inputRef, err := store.PersistVerificationArtifact(workflow.TaskID(id), "verification-pilot-compile-input", current)
	if err != nil { return "", err }
	requestID := fmt.Sprintf("pilot-compile-%s-%d", item.AttemptID, time.Now().UTC().UnixNano())
	request := workflow.StageRequest{ActorRequest: workflow.ActorRequest{SchemaVersion: workflow.ActorContractSchemaVersion, ID: requestID, Actor: workflow.ActorCompiler,
		TaskID: workflow.TaskID(id), WorkItemID: item.WorkItemID, AttemptID: item.AttemptID, Workspace: pilotExecutionWorkspace(state), PreparedAt: time.Now().UTC().Format(time.RFC3339Nano)},
		Phase: workflow.PhaseCompile, AllowedPaths: []string{"."}, Input: inputRef, InputDigest: current.Digest(), Plan: evidence.RoutingPlan, Policy: coder.Request.Policy}
	updated, err := store.PrepareStage(workflow.TaskID(id), state.StateRevision, request)
	if err != nil { return "", fmt.Errorf("prepare frozen compile stage: %w", err) }
	return updated.Protocol.Items[0].CurrentRequest, nil
}

func pilotStageRequestIsCompile(state workflow.RuntimeState, requestID string) bool {
	if state.Protocol == nil { return false }
	for _, record := range state.Protocol.Requests { if record.Request.ID == requestID { return record.Request.Phase == workflow.PhaseCompile } }
	return false
}

func ensurePilotSession(sessionID, taskID string, workItemID workflow.WorkItemID, attemptID workflow.AttemptID, workspace string, selection workflow.AISelection) (session.Status, error) {
	store := session.NewStore("")
	status, err := store.Load(sessionID)
	if errors.Is(err, session.ErrSessionNotFound) {
		status, err = store.Create(sessionID, "Schema 10 Codex pilot "+taskID, workspace, selection.Provider, selection.Model, "Implement only the frozen Workflow Coder request. Do not test, accept, deliver, or clean up.")
	}
	if err != nil { return session.Status{}, fmt.Errorf("load/create pilot Session: %w", err) }
	if status.Workspace.Path != workspace { return session.Status{}, errors.New("pilot Session workspace differs from the frozen Task") }
	if status.Task != nil && (status.Task.TaskID != taskID || status.Task.WorkItemID != string(workItemID) || status.Task.AttemptID != string(attemptID)) { return session.Status{}, errors.New("pilot Session is bound to a different Task or Attempt") }
	if status.Task == nil {
		status, err = store.Update(sessionID, func(current *session.Status) error {
			current.Task = &session.ManagedExecutionRef{SchemaVersion: workflow.DurableSchemaVersion, TaskID: taskID, WorkItemID: string(workItemID), AttemptID: string(attemptID)}
			return nil
		})
		if err != nil { return session.Status{}, err }
	}
	return status, nil
}

func pilotChecklistLine(id, checklistItem string) (string, string, error) {
	path, err := taskworkflow.WorkflowChecklistPath(id)
	if err != nil { return "", "", err }
	content, err := os.ReadFile(path)
	if err != nil { return "", "", fmt.Errorf("read mapped WorkItem checklist: %w", err) }
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- [") { continue }
		close := strings.Index(trimmed, "]")
		if close < 0 { continue }
		fields := strings.Fields(strings.TrimSpace(trimmed[close+1:]))
		if len(fields) != 0 && strings.Trim(fields[0], "`*_") == checklistItem { return trimmed, path, nil }
	}
	return "", path, fmt.Errorf("mapped checklist item %q is absent from %s", checklistItem, path)
}

func pilotExecutionWorkspace(state workflow.RuntimeState) string {
	if state.Workspace != nil && state.Workspace.WorktreePath != "" { return state.Workspace.WorktreePath }
	return state.Task.Workspace
}

func pilotDigest(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func loadCodexPilotEvidence(id string, store *workflow.Store, state workflow.RuntimeState) (codexPilotEvidence, workflow.ActorReference, error) {
	if state.Protocol == nil || len(state.Protocol.Activation) != 1 { return codexPilotEvidence{}, workflow.ActorReference{}, errors.New("Schema 10 Task has no exact pilot activation evidence") }
	ref := state.Protocol.Activation[0]
	var evidence codexPilotEvidence
	if err := store.ReadExecutionArtifact(workflow.TaskID(id), ref, &evidence); err != nil { return evidence, workflow.ActorReference{}, err }
	if evidence.Version != 1 || evidence.TaskID != workflow.TaskID(id) || evidence.WorkItemID == "" || evidence.Workspace != pilotExecutionWorkspace(state) || evidence.SourceSchema != workflow.SchemaVersion || evidence.RoutingPlan.Kind != "routing-plan" || evidence.Budget.InputTokenLimit <= 0 || evidence.Budget.OutputTokenLimit <= 0 {
		return codexPilotEvidence{}, workflow.ActorReference{}, errors.New("stored Codex pilot activation evidence is incomplete or stale")
	}
	plan, err := store.LoadRoutingPlan(workflow.TaskID(id))
	if err != nil || plan.Reference() != evidence.RoutingPlan { return codexPilotEvidence{}, workflow.ActorReference{}, errors.New("frozen Codex pilot routing/compile plan changed") }
	return evidence, ref, nil
}

func activateCodexPilot(id string, store *workflow.Store) error {
	state, err := store.Load(workflow.TaskID(id))
	if err != nil { return fmt.Errorf("load pilot Task: %w", err) }
	if state.SchemaVersion == workflow.DurableSchemaVersion {
		return reconcileCodexPilotActivation(id, store, state)
	}
	assessment := assessCodexPilot(id, store, state)
	if len(assessment.Blockers) != 0 { return fmt.Errorf("Codex pilot activation blocked: %s", strings.Join(assessment.Blockers, "; ")) }
	if err := store.PrepareDurableTaskLock(workflow.TaskID(id), func() error {
		locked, err := store.Load(workflow.TaskID(id))
		if err != nil { return err }
		if locked.StateRevision != state.StateRevision || locked.LastEventSequence != state.LastEventSequence { return errors.New("Task changed during pilot maintenance preflight") }
		if locked.Automation.Supervisor.LeaseID != "" || locked.WriteLease != nil || locked.PendingEvent != nil || len(locked.Attempts) != 0 { return errors.New("Task writer or Attempt appeared during pilot maintenance preflight") }
		return nil
	}); err != nil { return fmt.Errorf("prepare exclusive Task migration lock: %w", err) }
	ref, err := store.PersistPilotActivationEvidence(workflow.TaskID(id), assessment.Evidence)
	if err != nil { return fmt.Errorf("persist pilot activation evidence: %w", err) }
	verify := func(current workflow.RuntimeState, refs []workflow.ActorReference) error {
		if current.SchemaVersion != workflow.SchemaVersion || current.Task.ID != workflow.TaskID(id) || current.LastEventSequence != assessment.Evidence.SourceEventSequence || len(refs) != 1 || refs[0] != ref { return errors.New("pilot activation state changed or evidence identity differs") }
		for _, gate := range current.Gates { if gate.State == workflow.GateOpen { return fmt.Errorf("Task Gate %s opened during activation", gate.ID) } }
		var recorded codexPilotEvidence
		if err := store.ReadExecutionArtifact(workflow.TaskID(id), ref, &recorded); err != nil { return err }
		if !reflect.DeepEqual(recorded, assessment.Evidence) { return errors.New("persisted pilot activation evidence differs from the preflight snapshot") }
		plan, err := store.LoadRoutingPlan(workflow.TaskID(id))
		if err != nil { return err }
		if plan.Reference() != recorded.RoutingPlan { return errors.New("routing or compile plan changed during activation") }
		return nil
	}
	host := &execution.CodexStageHost{ID: codexPilotHostID, Store: store, Sessions: session.NewStore("")}
	if err := execution.ConnectCodexPilot(store, host, assessment.Policy, verify); err != nil { return err }
	migrated, err := store.MigrateDurableExecution(workflow.TaskID(id), []workflow.ActorReference{ref})
	if err != nil { return fmt.Errorf("migrate Task to Schema 10: %w", err) }
	if migrated.Protocol == nil || migrated.Protocol.Usage == nil || migrated.Protocol.Usage.Budget == nil {
		migrated, err = store.ConfigureTaskUsageBudget(workflow.TaskID(id), assessment.Budget)
		if err != nil { return fmt.Errorf("configure initial Task Token budget: %w", err) }
	} else if !reflect.DeepEqual(*migrated.Protocol.Usage.Budget, assessment.Budget) {
		return errors.New("existing Schema 10 Token budget differs from activation evidence; refusing to replace it")
	}
	fmt.Printf("Codex pilot activated for Task %s; WorkItem %s remains unaccepted\n", id, assessment.Item.ID)
	return projectWorkflowState(id, migrated)
}

func reconcileCodexPilotActivation(id string, store *workflow.Store, state workflow.RuntimeState) error {
	if state.Protocol == nil || len(state.Protocol.Activation) != 1 { return errors.New("Schema 10 Task has no exact pilot activation evidence") }
	ref := state.Protocol.Activation[0]
	var evidence codexPilotEvidence
	if err := store.ReadExecutionArtifact(workflow.TaskID(id), ref, &evidence); err != nil { return err }
	if evidence.Version != 1 || evidence.TaskID != workflow.TaskID(id) || evidence.WorkItemID == "" || evidence.Workspace != pilotExecutionWorkspace(state) || evidence.SourceSchema != workflow.SchemaVersion || evidence.RoutingPlan.Kind != "routing-plan" || evidence.Budget.InputTokenLimit <= 0 || evidence.Budget.OutputTokenLimit <= 0 {
		return errors.New("stored pilot activation evidence is incomplete or stale")
	}
	verify := func(current workflow.RuntimeState, refs []workflow.ActorReference) error {
		if current.SchemaVersion != workflow.DurableSchemaVersion || len(refs) != 1 || refs[0] != ref { return errors.New("activation reconciliation identity differs") }
		var recorded codexPilotEvidence
		if err := store.ReadExecutionArtifact(workflow.TaskID(id), ref, &recorded); err != nil { return err }
		if !reflect.DeepEqual(recorded, evidence) { return errors.New("stored pilot activation evidence changed") }
		return nil
	}
	host := &execution.CodexStageHost{ID: codexPilotHostID, Store: store, Sessions: session.NewStore("")}
	if err := execution.ConnectCodexPilot(store, host, evidence.RoutingPolicy, verify); err != nil { return err }
	updated, err := store.MigrateDurableExecution(workflow.TaskID(id), []workflow.ActorReference{ref})
	if err != nil { return err }
	if updated.Protocol == nil || updated.Protocol.Usage == nil || updated.Protocol.Usage.Budget == nil {
		updated, err = store.ConfigureTaskUsageBudget(workflow.TaskID(id), evidence.Budget)
		if err != nil { return fmt.Errorf("reconcile initial Token budget: %w", err) }
	} else if !reflect.DeepEqual(*updated.Protocol.Usage.Budget, evidence.Budget) {
		return errors.New("configured Token budget differs from immutable activation evidence")
	}
	fmt.Printf("Codex pilot activation reconciled for Task %s; WorkItem %s remains unaccepted\n", id, evidence.WorkItemID)
	return projectWorkflowState(id, updated)
}
