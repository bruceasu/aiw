package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type failedCompileRunner struct{}

func (failedCompileRunner) Run(context.Context, string, CompileAdapter) (string, int, error) {
	return "main.go:3: undefined: missing", 1, nil
}

func TestSupervisedCompilerPersistsResultWithCounter(t *testing.T) {
	store := NewStore(t.TempDir())
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "compile.py"), []byte("# fixture"), 0o644); err != nil { t.Fatal(err) }
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", State: WorkItemReady}}
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	state, err := store.StartAttempt("task-1", Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: workspace})
	if err != nil { t.Fatal(err) }
	prepared := &PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: workspace, DispatchedAt: "2026-01-01T00:00:00Z"}
	request := CompilerRequest{ActorRequest: ActorRequest{SchemaVersion: ActorContractSchemaVersion, ID: "compile-1", Actor: ActorCompiler, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: workspace, PreparedAt: "2026-01-01T00:00:00Z"}, ImplementationReport: ActorReference{Kind: "implementation-report", Path: "output.txt"}}
	plan := CompilePlanForWorkspace(workspace)
	request.Plan = &plan
	prepared.Compile = &SupervisedCompileState{Plan: &plan, Request: &request}
	if _, err := store.RecordAutomation("task-1", state.Automation.PlanFingerprint, AutomationCursor{Result: string(RunnerPrepared)}, prepared); err != nil { t.Fatal(err) }
	if _, _, err := store.RunCompiler(context.Background(), "task-1", request, failedCompileRunner{}); err == nil { t.Fatal("nonzero exit must fail even without a runner error") }
	loaded, err := store.Load("task-1")
	if err != nil { t.Fatal(err) }
	if loaded.Automation.PreparedRequest.CompilerResult == nil || loaded.Automation.PreparedRequest.CompilerResult.Status != ActorResultFailed {
		t.Fatal("recovery lost the completed compiler result")
	}
	if loaded.WorkItems[0].CompileFailureCount != 1 || loaded.WorkItems[0].NoProgressCount != 0 || loaded.WriteLease == nil {
		t.Fatalf("compiler must retain the Attempt and separate retry budgets: %+v", loaded)
	}
	compile := loaded.Automation.PreparedRequest.Compile
	if compile.Result == nil || compile.Failures != 1 || compile.Result.RequestID != request.ID {
		t.Fatalf("plan recovery lost the result or repair count: %+v", compile)
	}
	replayed, err := store.RecordCompilerRequestOutcome("task-1", request, *compile.Result)
	if err != nil { t.Fatal(err) }
	if replayed.WorkItems[0].CompileFailureCount != 1 || replayed.Automation.PreparedRequest.Compile.Failures != 1 {
		t.Fatal("recording the same compiler request twice consumed repair budget")
	}
}

func TestCompilerFailuresAreBoundedAndSuccessResetsTheCounter(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", State: WorkItemReady}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	diagnostics := ActorReference{Kind: "compile-diagnostics", Path: "compile-diagnostics/wi-0001.json"}

	for failure := 1; failure < MaxConsecutiveCompileFailures; failure++ {
		updated, err := store.RecordCompilerOutcome("task-1", "wi-0001", diagnostics, false)
		if err != nil {
			t.Fatal(err)
		}
		if updated.WorkItems[0].CompileFailureCount != failure || len(updated.Gates) != 0 {
			t.Fatalf("failure %d state = %#v", failure, updated)
		}
	}
	updated, err := store.RecordCompilerOutcome("task-1", "wi-0001", diagnostics, true)
	if err != nil {
		t.Fatal(err)
	}
	if updated.WorkItems[0].CompileFailureCount != 0 {
		t.Fatalf("successful compile did not reset counter: %#v", updated.WorkItems[0])
	}

	for failure := 1; failure <= MaxConsecutiveCompileFailures; failure++ {
		updated, err = store.RecordCompilerOutcome("task-1", "wi-0001", diagnostics, false)
		if err != nil {
			t.Fatal(err)
		}
	}
	if updated.WorkItems[0].CompileFailureCount != MaxConsecutiveCompileFailures {
		t.Fatalf("compile failure count = %d", updated.WorkItems[0].CompileFailureCount)
	}
	if len(updated.Gates) != 1 || updated.Gates[0].ID != "compiler-repair-limit-wi-0001" || updated.Gates[0].State != GateOpen {
		t.Fatalf("compiler repair gate = %#v", updated.Gates)
	}
}

func TestPrepareCoderRepairRequestUsesCompilerRepairLimit(t *testing.T) {
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", State: WorkItemRunning, CompileFailureCount: MaxConsecutiveCompileFailures}}
	report := ActorReference{Kind: "implementation-report", Path: "reports/wi-0001.json"}
	compiler := CompilerRequest{ActorRequest: ActorRequest{SchemaVersion: ActorContractSchemaVersion, ID: "compile-1", Actor: ActorCompiler, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: ".", PreparedAt: "2026-01-01T00:00:00Z"}, ImplementationReport: report}
	reference := ActorReference{Kind: "compile-diagnostics", Path: "compile-diagnostics/wi-0001.json"}
	result := CompilerResult{ActorResult: ActorResult{SchemaVersion: ActorContractSchemaVersion, RequestID: "compile-1", Actor: ActorCompiler, TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Status: ActorResultFailed, Outputs: []ActorReference{reference}, Summary: "compile failed", CompletedAt: "2026-01-01T00:00:00Z"}, Diagnostics: reference}
	if _, err := PrepareCoderRepairRequest(state, "coder-1", compiler, result, ActorReference{Kind: "implementation-contract", Path: "contracts/wi-0001.json"}, reference); err == nil {
		t.Fatal("expected exhausted compiler repair limit")
	}
}
