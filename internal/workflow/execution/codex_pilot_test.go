package execution

import (
	"strings"
	"testing"

	"aiw/internal/session"
	"aiw/internal/workflow"
)

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
