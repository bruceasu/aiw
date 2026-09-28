//go:build windows

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// These providers are test doubles, not production activation evidence.
func durableFixture(t *testing.T) (*Store, RuntimeState, ActorReference) {
	t.Helper()
	root := t.TempDir()
	store := NewStore(filepath.Join(root, "runtime"))
	store.ExecutionServices = &ExecutionServices{
		Authorize: func(RuntimeState, StageRequest) error { return nil },
		ValidateResult: func(RuntimeState, StageRequest, StageResult) error { return nil },
		ValidateAcceptance: func(RuntimeState, AcceptanceCandidate) error { return nil },
		VerifyActivation: func(RuntimeState, []ActorReference) error { return nil },
		MigrateBudget: func(s *RuntimeState) error { s.Protocol.BudgetKnown = true; s.Protocol.Budget = json.RawMessage(`[]`); return nil },
		Budget: func(s *RuntimeState, r StageRequest, operation string) error {
			var operations []string
			if err := json.Unmarshal(s.Protocol.Budget, &operations); err != nil { return err }
			operations = append(operations, r.ID+":"+operation)
			data, err := json.Marshal(operations)
			s.Protocol.Budget = data
			return err
		},
	}
	state := NewCompatibleRuntime(TaskReference{ID: "task-1", Workspace: root, Kind: WorkspacePrimary}, PlanningDraft, DeliveryPending)
	state.WorkItems = []WorkItem{{ID: "wi-0001", State: WorkItemReady}}
	if _, err := store.Create(state); err != nil { t.Fatalf("create fixture Store: %v", err) }
	if err := store.PrepareDurableTaskLock("task-1", func() error { return nil }); err != nil { t.Fatalf("prepare durable Task lock: %v", err) }
	lock, err := store.lock("task-1")
	if err != nil { t.Fatalf("acquire durable Task lock: %v", err) }
	ref, err := store.persistProtocolArtifactLocked("task-1", "test-activation", map[string]string{"source": "test double"})
	unlock(lock)
	if err != nil { t.Fatalf("persist fixture activation artifact: %v", err) }
	state, err = store.MigrateDurableExecution("task-1", []ActorReference{ref})
	if err != nil { t.Fatalf("migrate fixture execution: %v", err) }
	return store, state, ref
}

func preparedStageIntentFixture(t *testing.T) (*Store, RuntimeState, StageRequest) {
	t.Helper()
	store, state, ref := durableFixture(t)
	state, err := store.BeginExecution("task-1", state.StateRevision, Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: state.Task.Workspace})
	if err != nil { t.Fatalf("begin fixture execution: %v", err) }
	prepared := PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: state.Task.Workspace, SessionID: "session-1", ExpectedSessionTurn: 1}
	requestID := ExecutionRequestID(prepared)
	selection := &AISelection{Profile: "test", Provider: "test", Model: "test", Digest: "fixed"}
	input := ExecutionInput{SchemaVersion: 1, RequestID: requestID, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: state.Task.Workspace, SessionID: "session-1", Turn: 1, Actor: ActorCoder, AllowedPaths: []string{"src"}, Prompt: "Implement the fixed requirement.", Sources: []InputSource{{Kind: "requirement", Path: "requirement.md", Required: true, Status: "loaded", Content: "fixed", SHA256: contentDigest([]byte("fixed"))}}}
	input.AISelection = selection
	inputRef, err := store.PersistExecutionInput(prepared, input)
	if err != nil { t.Fatalf("persist fixture execution input: %v", err) }
	request := StageRequest{ActorRequest: ActorRequest{SchemaVersion: 1, ID: requestID, Actor: ActorCoder, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: state.Task.Workspace, PreparedAt: "2026-09-18T00:00:00Z"}, Phase: PhaseCoder, SessionID: "session-1", Turn: 1, Model: selection, AllowedPaths: []string{"src"}, Input: inputRef, InputDigest: inputRef.SHA256, Plan: ref, Policy: ref}
	store.ExecutionServices.Authorize = func(candidate RuntimeState, request StageRequest) error {
		item, err := executionItem(&candidate, request.WorkItemID)
		if err != nil { return err }
		if item.CurrentRequest != request.ID || candidate.WriteLease == nil || candidate.WriteLease.RequestID != request.ID || candidate.WriteLease.Generation != request.LeaseGeneration {
			t.Fatal("Coder authorization did not receive its candidate cursor and writer lease")
		}
		return nil
	}
	state, err = store.PrepareStage("task-1", state.StateRevision, request)
	if err != nil { t.Fatalf("prepare fixture stage: %v", err) }
	request = state.Protocol.Requests[0].Request
	return store, state, request
}

func preparedStageFixture(t *testing.T) (*Store, RuntimeState, StageRequest) {
	t.Helper()
	store, state, request := preparedStageIntentFixture(t)
	state, err = store.ClaimStageDispatch("task-1", state.StateRevision, request.ID, "executor-1")
	if err != nil { t.Fatalf("claim fixture stage dispatch: %v", err) }
	return store, state, request
}

func TestRetryInvalidCoderReportPreservesResultAndRequiresUnchangedWorkspace(t *testing.T) {
	store, state, plan := durableFixture(t)
	state, err := store.BeginExecution("task-1", state.StateRevision, Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: state.Task.Workspace})
	if err != nil { t.Fatal(err) }
	if err := os.MkdirAll(filepath.Join(state.Task.Workspace, "work"), 0o755); err != nil { t.Fatal(err) }
	baseline, err := CaptureValidationInputs(state.Task.Workspace, []string{"work"}, nil, "fixed-toolchain")
	if err != nil { t.Fatal(err) }
	prepared := PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", ExpectedSessionTurn: 1, Workspace: state.Task.Workspace}
	requestID := ExecutionRequestID(prepared)
	input := ExecutionInput{SchemaVersion: 1, RequestID: requestID, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", Turn: 1, Actor: ActorCoder, Workspace: state.Task.Workspace, Prompt: "Implement the WorkItem", AllowedPaths: []string{"work"}, Sources: []InputSource{{Kind: "work-item", Path: "tasks.md#1.1", Required: true, Status: "loaded", Content: "fixed", SHA256: contentDigest([]byte("fixed"))}}, WorkspaceInputs: &baseline}
	inputRef, err := store.PersistExecutionInput(prepared, input)
	if err != nil { t.Fatal(err) }
	prepared.InputReference = &inputRef
	prepared.DispatchedAt = "2026-09-29T00:00:00Z"
	reportRef, err := store.PersistExecutionReport(prepared, "session/0001-final.txt", "**No work done**")
	if err != nil { t.Fatal(err) }
	request := StageRequest{ActorRequest: ActorRequest{SchemaVersion: ActorContractSchemaVersion, ID: requestID, Actor: ActorCoder, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: state.Task.Workspace, PreparedAt: "2026-09-29T00:00:00Z"}, Phase: PhaseCoder, SessionID: "session-1", Turn: 1, Model: &AISelection{Profile: "test", Provider: "codex", Model: "test", Digest: "fixed"}, AllowedPaths: []string{"work"}, Input: inputRef, InputDigest: baseline.Digest(), Plan: plan, Policy: plan, LeaseGeneration: 1}
	state.Protocol.LeaseGeneration = 1
	state.Protocol.Requests = []StageRecord{{Request: request, Executor: "codex-pilot", Dispatch: "terminal", Consumed: true}}
	result := StageResult{RequestID: requestID, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", Turn: 1, LeaseGeneration: 1, InputDigest: baseline.Digest(), Executor: "codex-pilot", Terminal: true, Status: "passed", Evidence: []ActorReference{plan, reportRef}}
	if err := store.save(state); err != nil { t.Fatal(err) }
	resultRef, err := store.PersistStageResult("task-1", result)
	if err != nil { t.Fatal(err) }
	state.Protocol.Requests[0].Result = &resultRef
	state.Protocol.Items[0].Phase, state.Protocol.Items[0].CurrentRequest = PhaseReport, requestID
	if err := store.save(state); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(state.Task.Workspace, "work", "changed.txt"), []byte("changed"), 0o644); err != nil { t.Fatal(err) }
	if _, err := store.RetryInvalidCoderReport("task-1", state.StateRevision, "wi-0001"); err == nil { t.Fatal("changed workspace allowed a new Coder request") }
	if err := os.Remove(filepath.Join(state.Task.Workspace, "work", "changed.txt")); err != nil { t.Fatal(err) }
	retried, err := store.RetryInvalidCoderReport("task-1", state.StateRevision, "wi-0001")
	if err != nil { t.Fatal(err) }
	if retried.Protocol.Items[0].Phase != PhaseCoder || retried.Protocol.Items[0].ReportRetry == nil || retried.Protocol.Requests[0].Result == nil || retried.Protocol.Requests[0].Dispatch != "terminal" { t.Fatal("report retry erased the original Coder result") }
	if _, err := store.RetryInvalidCoderReport("task-1", retried.StateRevision, "wi-0001"); err == nil { t.Fatal("second report retry was allowed") }
}

func TestStopBeforeStageClaimPreventsDispatch(t *testing.T) {
	store, state, request := preparedStageIntentFixture(t)
	state, err := store.RequestExecutionStop("task-1", state.StateRevision, "operator stop", "test")
	if err != nil { t.Fatal(err) }
	if _, err := store.ClaimStageDispatch("task-1", state.StateRevision, request.ID, "executor-1"); err == nil {
		t.Fatal("Stop allowed a new model dispatch")
	}
	loaded, err := store.Load("task-1")
	if err != nil { t.Fatal(err) }
	if loaded.Protocol.Stop == nil || loaded.Protocol.Requests[0].Dispatch != "intent" {
		t.Fatal("refused dispatch changed the Stop or original request")
	}
}

func TestCompileResultStopsAtTesterWithoutAcceptance(t *testing.T) {
	store, state, ref := durableFixture(t)
	state, err := store.BeginExecution("task-1", state.StateRevision, Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: state.Task.Workspace})
	if err != nil { t.Fatal(err) }
	state, err = store.ConfigureTaskUsageBudget("task-1", TaskUsageBudget{TokenLimit: 1})
	if err != nil { t.Fatal(err) }
	state, err = store.RecordUsageEvent("task-1", usageBudgetEvent("task-1", "wi-0001", "attempt-1", "session-1", 1, 10, "0"))
	if err != nil { t.Fatal(err) }
	state.Protocol.Items[0].Phase = PhaseCompile
	state.Protocol.Items[0].ValidatedReport = &ref
	if err := store.save(state); err != nil { t.Fatal(err) }
	inputRef, err := store.PersistVerificationArtifact("task-1", "verification-input", map[string]string{"frozen": "input"})
	if err != nil { t.Fatal(err) }
	planRef, err := store.PersistVerificationArtifact("task-1", "verification-plan", map[string]string{"frozen": "plan"})
	if err != nil { t.Fatal(err) }
	policyRef, err := store.PersistVerificationArtifact("task-1", "verification-policy", map[string]string{"frozen": "policy"})
	if err != nil { t.Fatal(err) }
	request := StageRequest{ActorRequest: ActorRequest{SchemaVersion: ActorContractSchemaVersion, ID: "compile-1", Actor: ActorCompiler, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: state.Task.Workspace, PreparedAt: "2026-09-28T00:00:00Z"}, Phase: PhaseCompile, Input: inputRef, InputDigest: inputRef.SHA256, Plan: planRef, Policy: policyRef, AllowedPaths: []string{"."}}
	store.ExecutionServices.Authorize = func(candidate RuntimeState, request StageRequest) error {
		item, err := executionItem(&candidate, request.WorkItemID)
		if err != nil { return err }
		if item.CurrentRequest != request.ID || candidate.WriteLease != nil { t.Fatal("compile authorization did not receive its candidate cursor without a writer lease") }
		return nil
	}
	state, err = store.PrepareStage("task-1", state.StateRevision, request)
	if err != nil { t.Fatal(err) }
	request = state.Protocol.Requests[0].Request
	state, err = store.ClaimStageDispatch("task-1", state.StateRevision, request.ID, "compiler-1")
	if err != nil { t.Fatal(err) }
	result := StageResult{RequestID: request.ID, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, LeaseGeneration: request.LeaseGeneration, InputDigest: request.InputDigest, Executor: "compiler-1", Terminal: true, Status: "passed"}
	if _, err := store.PersistStageResult("task-1", result); err != nil { t.Fatal(err) }
	state, err = store.Load("task-1")
	if err != nil { t.Fatal(err) }
	state, err = store.ConsumeStageResult("task-1", state.StateRevision, result)
	if err != nil { t.Fatal(err) }
	item := state.Protocol.Items[0]
	if item.Phase != PhaseTester || item.ValidatedReport == nil || state.WorkItems[0].AcceptedReference != nil || state.Attempts[0].State != AttemptRunning {
		t.Fatalf("compile result crossed the Tester/acceptance boundary: %+v", item)
	}
}

func TestDurableUnknownAndStopRetainWriter(t *testing.T) {
	store, state, request := preparedStageFixture(t)
	if _, err := store.ClaimStageDispatch("task-1", state.StateRevision, request.ID, "executor-2"); err == nil { t.Fatal("unknown dispatch was repeated") }
	stopped, err := store.RequestExecutionStop("task-1", state.StateRevision, "operator stop", "test")
	if err != nil { t.Fatal(err) }
	reopened := NewStore(store.Root)
	loaded, err := reopened.Load("task-1")
	if err != nil { t.Fatal(err) }
	if loaded.Protocol.Stop == nil || loaded.WriteLease == nil || loaded.WriteLease.RequestID != request.ID || loaded.StateRevision != stopped.StateRevision { t.Fatal("Stop or writer was lost across reopening") }
	if _, err := reopened.StartSupervisor("task-1", "new-supervisor"); err == nil { t.Fatal("legacy restart bypassed Stop") }
}

func TestDurableResultReplayConsumesOnceAndKeepsAttempt(t *testing.T) {
	store, state, request := preparedStageFixture(t)
	result := StageResult{RequestID: request.ID, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, SessionID: request.SessionID, Turn: request.Turn, LeaseGeneration: request.LeaseGeneration, InputDigest: request.InputDigest, Executor: "executor-1", Terminal: true, Status: "passed"}
	if _, err := store.PersistStageResult("task-1", result); err != nil { t.Fatal(err) }
	saved, err := store.ReadStageResult("task-1", request.ID)
	if err != nil { t.Fatal(err) }
	state, err = store.ConsumeStageResult("task-1", state.StateRevision, saved)
	if err != nil { t.Fatal(err) }
	replayed, err := store.ConsumeStageResult("task-1", state.StateRevision, saved)
	if err != nil { t.Fatal(err) }
	if replayed.StateRevision != state.StateRevision || string(replayed.Protocol.Budget) != string(state.Protocol.Budget) { t.Fatal("result replay consumed another commit or budget") }
	if replayed.WriteLease != nil || replayed.Attempts[0].State != AttemptRunning || replayed.WorkItems[0].State == WorkItemCompleted || replayed.Protocol.Items[0].Phase != PhaseReport { t.Fatal("Coder result closed the Attempt or retained its writer") }
	result.Turn++
	if _, err := store.PersistStageResult("task-1", result); err == nil { t.Fatal("a different turn replaced the terminal result") }
}

func TestDurablePendingTailRecoveryKeepsRevision(t *testing.T) {
	store, state, _ := durableFixture(t)
	pending := Event{SchemaVersion: DurableSchemaVersion, Sequence: state.LastEventSequence+1, Type: "protocol.test", At: "2026-09-18T00:00:00Z", StateRevision: state.StateRevision+1, CommitID: "task-1-2"}
	state.StateRevision, state.CommitID, state.PendingEvent = pending.StateRevision, pending.CommitID, &pending
	if err := store.save(state); err != nil { t.Fatal(err) }
	data, err := json.Marshal(pending)
	if err != nil { t.Fatal(err) }
	f, err := os.OpenFile(store.path("task-1", runtimeEventsFile), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil { t.Fatal(err) }
	if _, err := f.Write(data[:len(data)/2]); err != nil { t.Fatal(err) }
	if err := f.Close(); err != nil { t.Fatal(err) }
	recovered, err := store.RecoverPendingEvent("task-1")
	if err != nil { t.Fatal(err) }
	if recovered.PendingEvent != nil || recovered.StateRevision != pending.StateRevision || recovered.LastEventSequence != pending.Sequence { t.Fatal("recovery did not confirm the original commit") }
	again, err := store.RecoverPendingEvent("task-1")
	if err != nil || again.StateRevision != recovered.StateRevision { t.Fatal("recovery repeated the transition") }
}

func TestLegacyLockAndMissingProvidersFailClosed(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.MigrateDurableExecution("task-1", nil); err == nil { t.Fatal("migration without providers succeeded") }
	lock, err := store.lock("task-1")
	if err != nil { t.Fatal(err) }
	defer unlock(lock)
	if err := store.PrepareDurableTaskLock("task-1", func() error { t.Fatal("maintenance must not bypass a legacy lock"); return nil }); err == nil { t.Fatal("legacy lock was taken over") }
}
