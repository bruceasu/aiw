package execution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/session"
	"aiw/internal/workflow"
)

func TestMissingSourceRecoveryRequiresUnstartedSessionAndAbsentJournal(t *testing.T) {
	store := workflow.NewStore(t.TempDir())
	sessions := session.NewStore(t.TempDir())
	request := codexPilotTestRequest()
	inputs := workflow.ValidationInputs{SchemaVersion: 1, Scope: []string{"."}, Toolchain: "fixed"}
	request.InputDigest = inputs.Digest()
	input := workflow.ExecutionInput{SchemaVersion: 1, RequestID: request.ID, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, SessionID: request.SessionID, Turn: request.Turn, Actor: workflow.ActorCoder, Workspace: request.Workspace, WorkspaceInputs: &inputs,
		AISelection: request.Model, Sources: []workflow.InputSource{{Kind: "work-item", Path: "tasks.md#1.1", Required: true, Status: "loaded", Content: "fixed", SHA256: digestHex([]byte("fixed"))}}}
	data, err := json.MarshalIndent(input, "", "  ")
	if err != nil { t.Fatal(err) }
	data = append(data, '\n')
	path := filepath.Join(store.Root, "tasks", string(request.TaskID), "reports", "inputs", request.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, data, 0o644); err != nil { t.Fatal(err) }
	request.Input = workflow.ActorReference{Kind: "execution-input", Path: "reports/inputs/" + request.ID + ".json", SHA256: digestHex(data)}
	if _, err := sessions.Create(request.SessionID, "pilot", request.Workspace, "codex", request.Model.Model, "fixed"); err != nil { t.Fatal(err) }
	if _, err := sessions.Update(request.SessionID, func(status *session.Status) error {
		status.Task = &session.ManagedExecutionRef{SchemaVersion: workflow.DurableSchemaVersion, TaskID: string(request.TaskID), WorkItemID: string(request.WorkItemID), AttemptID: string(request.AttemptID)}
		return nil
	}); err != nil { t.Fatal(err) }
	host := &CodexStageHost{ID: "codex-pilot", Store: store, Sessions: sessions}
	if err := host.proveMissingSourceNeverDispatched(request); err != nil { t.Fatal(err) }
	journal, err := host.invocationJournal(request)
	if err != nil { t.Fatal(err) }
	if err := journal.Prepare(); err != nil { t.Fatal(err) }
	if err := host.proveMissingSourceNeverDispatched(request); err == nil { t.Fatal("journaled request was misclassified as not dispatched") }
}

func codexPilotTestRequest() workflow.StageRequest {
	return workflow.StageRequest{
		ActorRequest: workflow.ActorRequest{SchemaVersion: workflow.ActorContractSchemaVersion, ID: "request-1", Actor: workflow.ActorCoder, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: ".", PreparedAt: "2026-09-28T00:00:00Z"},
		Phase: workflow.PhaseCoder, SessionID: "session-1", Turn: 1,
		InputDigest: strings.Repeat("a", 64),
		Model: &workflow.AISelection{Profile: "fast", Provider: "codex", Model: "gpt-5", Digest: "config-digest"},
	}
}

func TestConnectCodexPilotRefusesMissingActivationAuthority(t *testing.T) {
	store := workflow.NewStore(t.TempDir())
	host := &CodexStageHost{ID: "codex-pilot", Store: store, Sessions: session.NewStore(t.TempDir())}
	selection := workflow.AISelection{Profile: "fast", Provider: "codex", Model: "gpt-5", Digest: "config-digest"}
	policy := workflow.GenerationRoutingPolicy{Default: "fast", Profiles: []workflow.AISelection{selection}}
	if err := ConnectCodexPilot(store, host, policy, nil); err == nil { t.Fatal("pilot connected without activation authority") }
}

func TestConnectCodexPilotRejectsNonCodexProfile(t *testing.T) {
	store := workflow.NewStore(t.TempDir())
	host := &CodexStageHost{ID: "codex-pilot", Store: store, Sessions: session.NewStore(t.TempDir())}
	selection := workflow.AISelection{Profile: "fast", Provider: "copilot", Model: "model", Digest: "config-digest"}
	policy := workflow.GenerationRoutingPolicy{Default: "fast", Profiles: []workflow.AISelection{selection}}
	if err := ConnectCodexPilot(store, host, policy, func(workflow.RuntimeState, []workflow.ActorReference) error { return nil }); err == nil {
		t.Fatal("pilot accepted a non-Codex generation profile")
	}
}

func TestCodexInvocationRestartPreservesUnknownAndRejectsRedispatch(t *testing.T) {
	root := t.TempDir()
	request := codexPilotTestRequest()
	journal, err := NewCodexInvocationJournal(root, request)
	if err != nil { t.Fatal(err) }
	if err := journal.Prepare(); err != nil { t.Fatal(err) }

	restarted, err := NewCodexInvocationJournal(root, request)
	if err != nil { t.Fatal(err) }
	proof, err := restarted.Reconcile()
	if err != nil { t.Fatal(err) }
	if proof.Terminal || proof.Running || proof.Receipt.State != "prepared" { t.Fatalf("incomplete invocation was not preserved as unknown: %+v", proof) }
	if err := restarted.Prepare(); err == nil { t.Fatal("restart reserved a second invocation for the same request") }
}

func TestCodexInvocationWithoutJSONLRecordsUnknownExit(t *testing.T) {
	journal, err := NewCodexInvocationJournal(t.TempDir(), codexPilotTestRequest())
	if err != nil { t.Fatal(err) }
	if err := journal.Prepare(); err != nil { t.Fatal(err) }
	journal.current.State = "running"
	if err := journal.Finished(2); err != nil { t.Fatal(err) }
	proof, err := journal.Reconcile()
	if err != nil { t.Fatal(err) }
	if proof.Receipt.State != "unknown" || proof.Receipt.ExitCode == nil || *proof.Receipt.ExitCode != 2 || proof.Terminal { t.Fatalf("CLI rejection was misclassified: %+v", proof) }
}

func TestCLIArgumentRejectionNeedsStoppedZeroOutputAndExactParserError(t *testing.T) {
	exit := 2
	proof := CodexInvocationProof{Stopped: true, Receipt: CodexInvocation{State: "running", PID: 123, ProcessToken: "token", ExitCode: &exit}}
	if !codexCLIRejectionProof(proof) { t.Fatal("stopped zero-output parser failure was not eligible") }
	for _, altered := range []CodexInvocationProof{
		{Receipt: proof.Receipt},
		{Stopped: true, Running: true, Receipt: proof.Receipt},
		{Stopped: true, Events: []byte("{}\n"), Receipt: proof.Receipt},
		{Stopped: true, Receipt: CodexInvocation{State: "running", PID: 123, ProcessToken: "token", OutputBytes: 1, ExitCode: &exit}},
	} {
		if codexCLIRejectionProof(altered) { t.Fatalf("unsafe invocation was eligible: %+v", altered) }
	}
	message := "error: unexpected argument '--ask-for-approval' found\n\nUsage: codex exec [OPTIONS] [PROMPT]\n"
	if !codexCLIParserRejected(message) { t.Fatal("known parser error was rejected") }
	if codexCLIParserRejected("provider failed after dispatch\n" + message) || codexCLIParserRejected("error: unexpected argument '--ask-for-approval' found") { t.Fatal("ambiguous stderr was accepted") }
}

func TestCodexInvocationJournalRejectsChangedFrozenIdentity(t *testing.T) {
	root := t.TempDir()
	request := codexPilotTestRequest()
	journal, err := NewCodexInvocationJournal(root, request)
	if err != nil { t.Fatal(err) }
	if err := journal.Prepare(); err != nil { t.Fatal(err) }
	changed := request
	changed.InputDigest = strings.Repeat("b", 64)
	restarted, err := NewCodexInvocationJournal(root, changed)
	if err != nil { t.Fatal(err) }
	if _, err := restarted.Reconcile(); err == nil { t.Fatal("journal was reconciled against a changed frozen input") }
}

func TestCompileObservationWithoutReceiptRemainsUnknown(t *testing.T) {
	store := workflow.NewStore(t.TempDir())
	host := &CodexStageHost{ID: "codex-pilot", Store: store}
	request := workflow.StageRequest{ActorRequest: workflow.ActorRequest{ID: "compile-1", TaskID: "task-1"}, Phase: workflow.PhaseCompile}
	if _, _, err := host.Observe(request); err == nil || !strings.Contains(err.Error(), "never repeat") {
		t.Fatalf("compile without receipt did not remain unknown: %v", err)
	}
}
