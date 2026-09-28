package execution

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/ai"
	"aiw/internal/session"
	"aiw/internal/taskpath"
	"aiw/internal/workflow"
)

const defaultCodexStageTimeout = 30 * time.Minute

// CodexStageHost is the real, supervised Coder adapter. The surrounding
// JournaledVerificationHost supplies E03 validation; this adapter supplies
// Codex process identity, durable JSONL, Session recovery, and Coder receipts.
type CodexStageHost struct {
	ID       string
	Store    *workflow.Store
	Sessions *session.Store
	Timeout  time.Duration
}

type codexStageObservation struct {
	RequestID     string `json:"request_id"`
	RequestDigest string `json:"request_digest"`
	State         string `json:"state"`
	PID           int    `json:"pid,omitempty"`
	ProcessToken  string `json:"process_token,omitempty"`
	TerminalEvent string `json:"terminal_event,omitempty"`
	StderrDigest  string `json:"stderr_digest,omitempty"`
}

func (h *CodexStageHost) Identity() string { return h.ID }

func (h *CodexStageHost) Check(state workflow.RuntimeState, request workflow.StageRequest, _ *workflow.TestManifest) error {
	if h == nil || h.Store == nil || h.Sessions == nil || h.ID == "" { return errors.New("host-unavailable: managed Codex host is incomplete") }
	if request.Phase == workflow.PhaseCompile { return h.checkCompile(state, request) }
	if request.Phase != workflow.PhaseCoder || request.Model == nil || !strings.EqualFold(request.Model.Provider, "codex") { return errors.New("host-unavailable: pilot supports only a Codex Coder stage") }
	if state.Task.ID != request.TaskID || state.WriteLease == nil || state.WriteLease.RequestID != request.ID || state.WriteLease.Generation != request.LeaseGeneration { return errors.New("stale: Codex request does not own the current writer lease") }
	var input workflow.ExecutionInput
	if err := h.Store.ReadExecutionArtifact(request.TaskID, request.Input, &input); err != nil { return err }
	if input.Actor != workflow.ActorCoder || input.RequestID != request.ID || input.TaskID != request.TaskID || input.WorkItemID != request.WorkItemID || input.AttemptID != request.AttemptID || input.SessionID != request.SessionID || input.Turn != request.Turn || input.Workspace != request.Workspace || input.WorkspaceInputs == nil || input.WorkspaceInputs.Digest() != request.InputDigest || !equalCodexPaths(input.AllowedPaths, request.AllowedPaths) {
		return errors.New("stale: Codex Session input does not match the frozen Coder request")
	}
	// Source validation must precede ClaimStageDispatch. A rejected frozen
	// source must never leave an otherwise unstarted request as unknown.
	if err := h.Store.ValidateFrozenTaskSources(input); err != nil { return err }
	status, err := h.Sessions.Load(request.SessionID)
	if err != nil { return fmt.Errorf("load bound Codex Session: %w", err) }
	if status.Task == nil || status.Task.TaskID != string(request.TaskID) || status.Task.WorkItemID != string(request.WorkItemID) || status.Task.AttemptID != string(request.AttemptID) || status.Session.ID != request.SessionID || status.Session.LastTurn+1 != request.Turn || status.Workspace.Path != request.Workspace {
		return errors.New("stale: Codex Session does not match the frozen Task, Attempt, workspace, or turn")
	}
	if status.Session.State != session.StateActive && status.Session.State != session.StateCreated { return errors.New("Codex Session is not ready for a new frozen turn") }
	return nil
}

func (h *CodexStageHost) VerifyReceipt(request workflow.StageRequest, receipt workflow.VerificationReceipt) error {
	if request.Phase == workflow.PhaseCompile {
		saved, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID)
		if err != nil { return err }
		if !equalCodexJSON(saved, receipt) || receipt.Result.Executor != h.ID || len(receipt.Result.Evidence) != 1 { return errors.New("compile receipt differs from its immutable host record") }
		var proof codexPilotCompileEvidence
		if err := h.Store.ReadExecutionArtifact(request.TaskID, receipt.Result.Evidence[0], &proof); err != nil { return err }
		if proof.RequestDigest != codexRequestDigest(request) || proof.Plan != request.Plan || proof.ResultStatus != receipt.Result.Status { return errors.New("compile evidence does not match its frozen request") }
		return nil
	}
	saved, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID)
	if err != nil { return err }
	if !equalCodexJSON(saved, receipt) || receipt.Request.ID != request.ID || receipt.Result.Executor != h.ID { return errors.New("Codex receipt differs from its immutable Task journal") }
	journal, err := h.invocationJournal(request)
	if err != nil { return err }
	proof, err := journal.Reconcile()
	if err != nil { return err }
	if !proof.Terminal || proof.Receipt.RequestDigest == "" { return errors.New("Codex receipt lacks a stopped terminal process proof") }
	return nil
}

func (h *CodexStageHost) VerifyObservation(_ workflow.RuntimeState, request workflow.StageRequest, result workflow.StageResult) error {
	if request.Phase == workflow.PhaseCompile { return errors.New("compile stage has no terminal receipt; preserve the original request as unknown") }
	if len(result.Evidence) != 1 { return errors.New("Codex observation requires one exact host artifact") }
	var evidence codexStageObservation
	if err := h.Store.ReadExecutionArtifact(request.TaskID, result.Evidence[0], &evidence); err != nil { return err }
	if evidence.RequestID != request.ID || evidence.RequestDigest != codexRequestDigest(request) { return errors.New("Codex observation does not identify the original request") }
	if evidence.State == "preflight-rejected" {
		if result.Status != "not-dispatched" || evidence.PID != 0 || evidence.ProcessToken != "" || evidence.TerminalEvent != "" { return errors.New("preflight recovery cannot claim a started process") }
		return h.proveMissingSourceNeverDispatched(request)
	}
	if evidence.State == "cli-argument-rejected" {
		if result.Status != "not-dispatched" { return errors.New("CLI argument rejection requires a not-dispatched observation") }
		proof, stderrDigest, err := h.proveCLIArgumentRejected(request)
		if err != nil { return err }
		if evidence.PID != proof.Receipt.PID || evidence.ProcessToken != proof.Receipt.ProcessToken || evidence.StderrDigest != stderrDigest || evidence.TerminalEvent != "" { return errors.New("CLI argument rejection proof changed") }
		return nil
	}
	if evidence.PID == 0 && (evidence.ProcessToken != "" || evidence.State != "prepared") || evidence.PID > 0 && evidence.ProcessToken == "" { return errors.New("Codex observation has an incomplete process identity") }
	journal, err := h.invocationJournal(request)
	if err != nil { return err }
	proof, err := journal.Reconcile()
	if err != nil { return err }
	if proof.Receipt.RequestDigest != evidence.RequestDigest || proof.Receipt.PID != evidence.PID || proof.Receipt.ProcessToken != evidence.ProcessToken || proof.Receipt.State != evidence.State { return errors.New("Codex observation conflicts with the durable invocation journal") }
	return nil
}

// ProveMissingSourceNotDispatched is an explicit recovery for the legacy pilot
// input that omitted its Task source. It cannot recover an arbitrary unknown
// invocation and never starts a Session or Codex process.
func (h *CodexStageHost) ProveMissingSourceNotDispatched(request workflow.StageRequest) (workflow.DispatchObservation, error) {
	if err := h.proveMissingSourceNeverDispatched(request); err != nil { return workflow.DispatchObservation{}, err }
	evidence := codexStageObservation{RequestID: request.ID, RequestDigest: codexRequestDigest(request), State: "preflight-rejected"}
	ref, err := h.Store.PersistVerificationArtifact(request.TaskID, "verification-codex-invocation-observation", evidence)
	if err != nil { return workflow.DispatchObservation{}, err }
	return workflow.DispatchObservation{RequestID: request.ID, Executor: h.ID, Generation: request.LeaseGeneration, Status: "not-dispatched", Evidence: ref}, nil
}

// ProveCLIArgumentRejected handles only the known pre-dispatch Codex CLI
// parser failure. It does not start a process or infer safety from PID absence.
func (h *CodexStageHost) ProveCLIArgumentRejected(request workflow.StageRequest) (workflow.DispatchObservation, error) {
	proof, stderrDigest, err := h.proveCLIArgumentRejected(request)
	if err != nil { return workflow.DispatchObservation{}, err }
	evidence := codexStageObservation{RequestID: request.ID, RequestDigest: codexRequestDigest(request), State: "cli-argument-rejected", PID: proof.Receipt.PID, ProcessToken: proof.Receipt.ProcessToken, StderrDigest: stderrDigest}
	ref, err := h.Store.PersistVerificationArtifact(request.TaskID, "verification-codex-invocation-observation", evidence)
	if err != nil { return workflow.DispatchObservation{}, err }
	return workflow.DispatchObservation{RequestID: request.ID, Executor: h.ID, Generation: request.LeaseGeneration, Status: "not-dispatched", Evidence: ref}, nil
}

func (h *CodexStageHost) proveCLIArgumentRejected(request workflow.StageRequest) (CodexInvocationProof, string, error) {
	reject := func(reason string) (CodexInvocationProof, string, error) { return CodexInvocationProof{}, "", errors.New(reason) }
	if h == nil || h.Store == nil || h.Sessions == nil || h.ID == "" || request.Phase != workflow.PhaseCoder || request.Turn != 1 || request.Model == nil || !strings.EqualFold(request.Model.Provider, "codex") { return reject("CLI rejection recovery requires the original Codex Coder turn") }
	if _, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID); err == nil { return reject("Codex request already has a terminal receipt") } else if !errors.Is(err, os.ErrNotExist) { return CodexInvocationProof{}, "", err }
	journal, err := h.invocationJournal(request)
	if err != nil { return CodexInvocationProof{}, "", err }
	proof, err := journal.Reconcile()
	if err != nil { return CodexInvocationProof{}, "", err }
	if !codexCLIRejectionProof(proof) { return reject("Codex process or output cannot prove the known CLI parser rejection") }
	status, err := h.Sessions.Load(request.SessionID)
	if err != nil { return CodexInvocationProof{}, "", err }
	if status.Task == nil || status.Task.TaskID != string(request.TaskID) || status.Task.WorkItemID != string(request.WorkItemID) || status.Task.AttemptID != string(request.AttemptID) || status.Session.ID != request.SessionID || status.Session.State != session.StateFailed || status.Session.LastTurn != request.Turn || status.Workspace.Path != request.Workspace || status.Backend.Name != "codex" || status.Backend.Model != request.Model.Model || status.Result.Status != "failed" || status.Execution.LastExitCode == nil || *status.Execution.LastExitCode != 2 { return reject("bound Codex Session does not prove the CLI parser failure") }
	for _, name := range []string{"outputs/0001-events.jsonl", "outputs/0001-live.jsonl", "outputs/0001-final.txt"} {
		content, err := h.Sessions.ReadText(request.SessionID, name)
		if err != nil || content != "" { return reject("Codex Session has missing or nonempty output; keep request unknown") }
	}
	stderr, err := h.Sessions.ReadText(request.SessionID, "outputs/0001-stderr.log")
	if err != nil { return CodexInvocationProof{}, "", err }
	if !codexCLIParserRejected(stderr) { return reject("Codex stderr does not prove the known unsupported-argument rejection") }
	return proof, digestHex([]byte(stderr)), nil
}

func codexCLIRejectionProof(proof CodexInvocationProof) bool {
	r := proof.Receipt
	return proof.Stopped && !proof.Terminal && !proof.Running && r.PID > 0 && r.ProcessToken != "" && r.OutputBytes == 0 && len(proof.Events) == 0 && r.TerminalEvent == "" && (r.State == "running" || r.State == "unknown") && (r.ExitCode == nil || *r.ExitCode == 2)
}

func codexCLIParserRejected(stderr string) bool {
	return strings.HasPrefix(strings.TrimSpace(stderr), "error: unexpected argument '--ask-for-approval' found") && strings.Contains(stderr, "Usage: codex exec")
}

func (h *CodexStageHost) proveMissingSourceNeverDispatched(request workflow.StageRequest) error {
	if h == nil || h.Store == nil || h.Sessions == nil || h.ID == "" || request.Phase != workflow.PhaseCoder || request.Turn != 1 || request.Model == nil || !strings.EqualFold(request.Model.Provider, "codex") { return errors.New("preflight recovery requires the original Codex Coder turn") }
	var input workflow.ExecutionInput
	if err := h.Store.ReadExecutionArtifact(request.TaskID, request.Input, &input); err != nil { return err }
	if input.RequestID != request.ID || input.TaskID != request.TaskID || input.WorkItemID != request.WorkItemID || input.AttemptID != request.AttemptID || input.SessionID != request.SessionID || input.Turn != request.Turn || input.Actor != workflow.ActorCoder || input.Workspace != request.Workspace || !equalCodexJSON(input.AISelection, request.Model) || input.WorkspaceInputs == nil || input.WorkspaceInputs.Digest() != request.InputDigest || len(input.Sources) != 1 || input.Sources[0].Kind != "work-item" || !input.Sources[0].Required || input.Sources[0].Status != "loaded" || input.Sources[0].SHA256 != digestHex([]byte(input.Sources[0].Content)) { return errors.New("unknown request is not the legacy pilot source-preflight failure") }
	status, err := h.Sessions.Load(request.SessionID)
	if err != nil { return err }
	if status.Task == nil || status.Task.TaskID != string(request.TaskID) || status.Task.WorkItemID != string(request.WorkItemID) || status.Task.AttemptID != string(request.AttemptID) || status.Session.ID != request.SessionID || status.Session.State != session.StateCreated || status.Session.LastTurn != 0 || status.Backend.Name != "codex" || status.Backend.Model != request.Model.Model || status.Result.Status != "not_started" || status.Execution.Phase != "" || status.Execution.LastStartedAt != "" || status.Execution.LastCompletedAt != "" || status.Execution.LastExitCode != nil || status.Workspace.Path != request.Workspace { return errors.New("original Codex Session has execution evidence; keep request unknown") }
	if _, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID); err == nil { return errors.New("original Codex request already has a receipt") } else if !errors.Is(err, os.ErrNotExist) { return err }
	journal, err := h.invocationJournal(request)
	if err != nil { return err }
	for _, path := range []string{journal.recordPath, journal.outputPath} {
		if _, err := os.Lstat(path); err == nil { return errors.New("Codex invocation journal exists; keep request unknown") } else if !errors.Is(err, os.ErrNotExist) { return err }
	}
	return nil
}

func (h *CodexStageHost) Start(ctx context.Context, request workflow.StageRequest, input *workflow.ExecutionInput, _ *workflow.TestManifest) (*workflow.DispatchObservation, *workflow.VerificationReceipt, error) {
	if request.Phase == workflow.PhaseCompile { return h.startCompile(ctx, request) }
	if input == nil { return nil, nil, errors.New("frozen Coder input is missing") }
	if _, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID); err == nil { return h.Observe(request) } else if !errors.Is(err, os.ErrNotExist) { return nil, nil, err }
	journal, err := h.invocationJournal(request)
	if err != nil { return nil, nil, err }
	if err := journal.Prepare(); err != nil { return nil, nil, fmt.Errorf("reserve original Codex invocation: %w", err) }
	limit := h.Timeout
	if limit <= 0 { limit = defaultCodexStageTimeout }
	runCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	var result ai.Response
	var runErr error
	state, err := h.Store.Load(request.TaskID)
	if err != nil { return nil, nil, err }
	if state.Protocol == nil || state.Protocol.Stop != nil { return nil, nil, errors.New("Codex execution is stopped before dispatch") }
	prompt := input.Prompt
	if input.OutputSchema != nil { prompt += codexReportIdentityInstruction(request) }
	frozen := session.FrozenTurn{Prompt: prompt, OutputSchema: input.OutputSchema, ExpectedTurn: request.Turn, ReasoningIntensity: request.Model.ReasoningIntensity, InvocationObserver: journal}
	result, runErr = session.ExecuteFrozenTurn(runCtx, h.Sessions, request.SessionID, "artifact-generation", frozen, "codex", request.Model.Model, nil)
	proof, err := journal.Reconcile()
	if err != nil { return nil, nil, errors.Join(runErr, err) }
	if !proof.Terminal { return nil, nil, errors.Join(runErr, errors.New("Codex process exited without a complete terminal receipt")) }
	if result.CompletedAt.IsZero() { result = codexResponse(proof, request, time.Now().UTC()) }
	if err := session.RecoverFrozenTurn(h.Sessions, request.SessionID, request.Turn, result); err != nil { return nil, nil, errors.Join(runErr, err) }
	receipt, err := h.terminalReceipt(request, *input, result, proof)
	if err != nil { return nil, nil, errors.Join(runErr, err) }
	if err := h.Store.PersistVerificationReceipt(request.TaskID, receipt); err != nil { return nil, nil, errors.Join(runErr, err) }
	return nil, &receipt, runErr
}

func codexReportIdentityInstruction(request workflow.StageRequest) string {
	return fmt.Sprintf("\n\nRequired report identity (copy these values exactly): schema_version=1, request_id=%s, task_id=%s, work_item_id=%s, attempt_id=%s, session_id=%s, turn=%d, actor=coder, input_sha256=%s. Each report section requires state, items, and reason. The final answer must be the JSON object only, without Markdown fences.\n", request.ID, request.TaskID, request.WorkItemID, request.AttemptID, request.SessionID, request.Turn, request.Input.SHA256)
}

func (h *CodexStageHost) Observe(request workflow.StageRequest) (*workflow.DispatchObservation, *workflow.VerificationReceipt, error) {
	if request.Phase == workflow.PhaseCompile {
		receipt, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID)
		if err != nil { return nil, nil, fmt.Errorf("compile result is unknown; never repeat the original compile request: %w", err) }
		return nil, &receipt, nil
	}
	if receipt, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID); err == nil { return nil, &receipt, nil } else if !errors.Is(err, os.ErrNotExist) { return nil, nil, err }
	journal, err := h.invocationJournal(request)
	if err != nil { return nil, nil, err }
	proof, err := journal.Reconcile()
	if err != nil { return nil, nil, err }
	if proof.Terminal {
		var input workflow.ExecutionInput
		if err := h.Store.ReadExecutionArtifact(request.TaskID, request.Input, &input); err != nil { return nil, nil, err }
		response := codexResponse(proof, request, time.Now().UTC())
		if err := session.RecoverFrozenTurn(h.Sessions, request.SessionID, request.Turn, response); err != nil { return nil, nil, err }
		receipt, err := h.terminalReceipt(request, input, response, proof)
		if err != nil { return nil, nil, err }
		if err := h.Store.PersistVerificationReceipt(request.TaskID, receipt); err != nil { return nil, nil, err }
		return nil, &receipt, nil
	}
	evidence := codexStageObservation{RequestID: request.ID, RequestDigest: proof.Receipt.RequestDigest, State: proof.Receipt.State, PID: proof.Receipt.PID, ProcessToken: proof.Receipt.ProcessToken, TerminalEvent: proof.Receipt.TerminalEvent}
	ref, err := h.Store.PersistVerificationArtifact(request.TaskID, "verification-codex-invocation-observation", evidence)
	if err != nil { return nil, nil, err }
	return &workflow.DispatchObservation{RequestID: request.ID, Executor: h.ID, Generation: request.LeaseGeneration, Status: "unknown", Evidence: ref}, nil, nil
}

type codexPilotCompileEvidence struct {
	RequestDigest string `json:"request_digest"`
	Plan workflow.ActorReference `json:"plan"`
	ResultStatus string `json:"result_status"`
	StartedAt string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Targets []codexPilotCompileTarget `json:"targets"`
}

type codexPilotCompileTarget struct {
	Name string `json:"name"`
	ExitCode int `json:"exit_code"`
	Output string `json:"output"`
	Error string `json:"error,omitempty"`
}

func pilotExecutionItem(state workflow.RuntimeState, id workflow.WorkItemID) (*workflow.ItemExecution, error) {
	if state.Protocol == nil { return nil, errors.New("Schema 10 execution protocol is missing") }
	for index := range state.Protocol.Items {
		if state.Protocol.Items[index].WorkItemID == id { return &state.Protocol.Items[index], nil }
	}
	return nil, fmt.Errorf("no execution cursor for WorkItem %s", id)
}

func (h *CodexStageHost) checkCompile(state workflow.RuntimeState, request workflow.StageRequest) error {
	if state.Protocol == nil || state.Protocol.Stop != nil || request.Model != nil || request.Route != nil || request.Actor != workflow.ActorCompiler { return errors.New("compile stage is not available or is incorrectly routed") }
	item, err := pilotExecutionItem(state, request.WorkItemID)
	if err != nil || item.AttemptID != request.AttemptID || item.Phase != workflow.PhaseCompile || item.CurrentRequest != request.ID || item.ValidatedReport == nil { return errors.New("compile does not match the current validated Coder report") }
	plan, err := h.Store.LoadRoutingPlan(request.TaskID)
	if err != nil || plan.Reference() != request.Plan || !plan.Compile.Available { return errors.New("frozen compile plan is unavailable or changed") }
	var inputs workflow.ValidationInputs
	if err := h.Store.ReadExecutionArtifact(request.TaskID, request.Input, &inputs); err != nil { return err }
	if inputs.Digest() != request.InputDigest || len(inputs.Scope) == 0 { return errors.New("compile input manifest differs from the frozen request") }
	return nil
}

func (h *CodexStageHost) startCompile(ctx context.Context, request workflow.StageRequest) (*workflow.DispatchObservation, *workflow.VerificationReceipt, error) {
	if receipt, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID); err == nil { return nil, &receipt, nil
	} else if !errors.Is(err, os.ErrNotExist) { return nil, nil, err }
	state, err := h.Store.Load(request.TaskID)
	if err != nil { return nil, nil, err }
	if err := h.checkCompile(state, request); err != nil { return nil, nil, err }
	var inputs workflow.ValidationInputs
	if err := h.Store.ReadExecutionArtifact(request.TaskID, request.Input, &inputs); err != nil { return nil, nil, err }
	plan, err := h.Store.LoadRoutingPlan(request.TaskID)
	if err != nil { return nil, nil, err }
	root, err := filepath.Abs(request.Workspace)
	if err != nil { return nil, nil, err }
	started := time.Now().UTC()
	limit := h.Timeout
	if limit <= 0 { limit = defaultCodexStageTimeout }
	runCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	proof := codexPilotCompileEvidence{RequestDigest: codexRequestDigest(request), Plan: request.Plan, ResultStatus: "passed", StartedAt: started.Format(time.RFC3339Nano)}
	var failures []string
	for _, target := range workflow.SelectCompileTargets(plan.Compile, nil) {
		current, loadErr := h.Store.Load(request.TaskID)
		if loadErr != nil { return nil, nil, loadErr }
		if current.Protocol == nil || current.Protocol.Stop != nil { return nil, nil, errors.New("Stop prevents the next frozen compile target") }
		adapter, resolveErr := workflow.ResolveCompileTarget(root, target)
		entry := codexPilotCompileTarget{Name: target.Name, ExitCode: -1}
		if resolveErr == nil {
			entry.Output, entry.ExitCode, resolveErr = (workflow.ExecCompileCommandRunner{}).Run(runCtx, root, adapter)
		}
		if resolveErr != nil { entry.Error = resolveErr.Error(); failures = append(failures, target.Name+": "+resolveErr.Error()) }
		proof.Targets = append(proof.Targets, entry)
	}
	if len(proof.Targets) == 0 { return nil, nil, errors.New("frozen compile plan selected no targets") }
	proof.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if len(failures) != 0 { proof.ResultStatus = "failed" }
	ref, err := h.Store.PersistVerificationArtifact(request.TaskID, "verification-compile-result", proof)
	if err != nil { return nil, nil, err }
	toolchain, err := workflow.ValidationToolchainIdentity()
	if err != nil { return nil, nil, err }
	after, err := workflow.CaptureValidationInputs(root, inputs.Scope, inputs.Bindings, toolchain)
	if err != nil { return nil, nil, err }
	result := workflow.StageResult{RequestID: request.ID, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID,
		LeaseGeneration: request.LeaseGeneration, InputDigest: request.InputDigest, Executor: h.ID, Terminal: true,
		Status: proof.ResultStatus, Evidence: []workflow.ActorReference{ref}}
	if proof.ResultStatus == "failed" { result.FailureClass = "unattributed" }
	if proof.ResultStatus == "passed" { result.Evidence = []workflow.ActorReference{ref} }
	receipt := workflow.VerificationReceipt{SchemaVersion: 1, Request: request, Result: result, Inputs: inputs, After: after, Checks: []workflow.TestCheckReceipt{}}
	if err := h.Store.PersistVerificationReceipt(request.TaskID, receipt); err != nil { return nil, nil, err }
	return nil, &receipt, nil
}

func (h *CodexStageHost) terminalReceipt(request workflow.StageRequest, input workflow.ExecutionInput, response ai.Response, proof CodexInvocationProof) (workflow.VerificationReceipt, error) {
	if input.WorkspaceInputs == nil || input.WorkspaceInputs.Digest() != request.InputDigest { return workflow.VerificationReceipt{}, errors.New("Codex Coder input snapshot is stale") }
	prepared := workflow.PreparedAgentRequest{TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, SessionID: request.SessionID, ExpectedSessionTurn: request.Turn, Workspace: request.Workspace, InputReference: &request.Input, AISelection: request.Model, PreparedAt: request.PreparedAt, DispatchedAt: proof.Receipt.StartedAt.UTC().Format(time.RFC3339)}
	source := filepath.ToSlash(filepath.Join(".ai", "sessions", request.SessionID, "outputs", fmt.Sprintf("%04d-final.txt", request.Turn)))
	reportRef, err := h.Store.PersistExecutionReport(prepared, source, response.FinalOutput)
	if err != nil { return workflow.VerificationReceipt{}, err }
	toolchain, err := workflow.ValidationToolchainIdentity()
	if err != nil { return workflow.VerificationReceipt{}, err }
	after, err := workflow.CaptureValidationInputs(request.Workspace, input.WorkspaceInputs.Scope, input.WorkspaceInputs.Bindings, toolchain)
	if err != nil { return workflow.VerificationReceipt{}, err }
	result := workflow.StageResult{RequestID: request.ID, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, SessionID: request.SessionID, Turn: request.Turn, LeaseGeneration: request.LeaseGeneration, InputDigest: request.InputDigest, Executor: h.ID, Terminal: true, Status: "passed", Evidence: []workflow.ActorReference{reportRef}}
	if response.ExitCode != 0 || proof.Receipt.TerminalEvent == "turn.failed" { result.Status, result.FailureClass = "blocked", "infrastructure" }
	usage, err := json.Marshal(ai.UsageWithoutRawEvidence(response.Usage))
	if err != nil { return workflow.VerificationReceipt{}, err }
	return workflow.VerificationReceipt{SchemaVersion: 1, Request: request, Result: result, Inputs: *input.WorkspaceInputs, After: after, Checks: []workflow.TestCheckReceipt{}, Usage: usage}, nil
}

func (h *CodexStageHost) invocationJournal(request workflow.StageRequest) (*CodexInvocationJournal, error) {
	root := filepath.Join(taskpath.ActiveTaskDir(h.Store.Root, string(request.TaskID)), "reports", "codex-invocations")
	return NewCodexInvocationJournal(root, request)
}

func codexResponse(proof CodexInvocationProof, request workflow.StageRequest, now time.Time) ai.Response {
	started := proof.Receipt.StartedAt
	completed := proof.Receipt.CompletedAt
	if completed.IsZero() { completed = now }
	exitCode := 1
	if proof.Receipt.ExitCode != nil { exitCode = *proof.Receipt.ExitCode } else if proof.Receipt.TerminalEvent == "turn.completed" { exitCode = 0 }
	output := ai.DecodeCodexFinalOutput(proof.Events)
	if strings.TrimSpace(output) == "" { output = ai.DecodeCodexError(proof.Events) }
	if strings.TrimSpace(output) == "" { output = "Codex turn ended without a final assistant message" }
	return ai.Response{ThreadID: ai.DecodeThreadID(proof.Events), FinalOutput: output, Events: proof.Events, ExitCode: exitCode,
		Metadata: map[string]string{"provider": "codex", "request_id": request.ID}, StartedAt: started, CompletedAt: completed,
		Usage: ai.UsageFromRaw("codex", request.Model.Model, started, completed, proof.Events)}
}

func codexRequestDigest(request workflow.StageRequest) string {
	encoded, _ := json.Marshal(request)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func equalCodexPaths(left, right []string) bool { return equalCodexJSON(left, right) }
func equalCodexJSON(left, right any) bool {
	a, err := json.Marshal(left); if err != nil { return false }
	b, err := json.Marshal(right); return err == nil && string(a) == string(b)
}
