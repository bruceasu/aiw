package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/session"
	"aiw/internal/workflow"
)

type unexpectedCompiler struct { t *testing.T }

func (r unexpectedCompiler) Run(context.Context, string, workflow.CompileAdapter) (string, int, error) {
	r.t.Fatal("recovery must not rerun an already accepted compiler")
	return "", 0, nil
}

type sequenceCompiler struct { codes []int; calls int }

func (r *sequenceCompiler) Run(context.Context, string, workflow.CompileAdapter) (string, int, error) {
	code := r.codes[r.calls]
	r.calls++
	if code != 0 { return "main.go:1: undefined: missing", code, nil }
	return "", 0, nil
}

func TestSupervisedCompileRepairCallChain(t *testing.T) {
	for _, tc := range []struct { name string; codes []int; blocked bool }{
		{"three failures stop", []int{1, 1, 1}, true},
		{"success resets failures", []int{1, 0}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			previous, err := os.Getwd()
			if err != nil { t.Fatal(err) }
			if err := os.Chdir(root); err != nil { t.Fatal(err) }
			t.Cleanup(func() { if err := os.Chdir(previous); err != nil { t.Error(err) } })
			// Keep repository discovery local and never invoke an installed CLI.
			t.Setenv("PATH", "")
			if err := os.WriteFile("compile.py", []byte("# unused: fake runner"), 0o644); err != nil { t.Fatal(err) }
			artifacts := filepath.Join(root, ".ai", "task-1", "artifacts")
			if err := os.MkdirAll(artifacts, 0o755); err != nil { t.Fatal(err) }
			if err := os.WriteFile(filepath.Join(artifacts, "handoff.md"), []byte("Implement selected item."), 0o644); err != nil { t.Fatal(err) }
			sessions := session.NewStore("")
			if _, err := sessions.Create("session-1", "fixture", root, "codex", "", "fixture"); err != nil { t.Fatal(err) }
			store := workflow.NewStore(filepath.Join(root, ".ai"))
			state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: "task-1", Kind: workflow.WorkspacePrimary}, workflow.PlanningReady, workflow.DeliveryUnmanaged)
			state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemReady}}
			if _, err := store.Create(state); err != nil { t.Fatal(err) }
			if _, err := store.StartAttempt("task-1", workflow.Attempt{ID: "attempt-1", WorkItemID: "wi-0001", SessionID: "session-1", Workspace: root}); err != nil { t.Fatal(err) }
			request := &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", Workspace: root, ExpectedSessionTurn: 1}
			plan := workflow.CompilePlanForWorkspace(root)
			request.Compile = &workflow.SupervisedCompileState{Plan: &plan}
			if _, err := store.RecordAutomation("task-1", "", workflow.AutomationCursor{Result: string(workflow.RunnerPrepared)}, request); err != nil { t.Fatal(err) }
			runner := &sequenceCompiler{codes: tc.codes}
			outcome := workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeCompleted, EvidenceReference: "session-output.txt"}
			for index := range tc.codes {
				if _, err := sessions.Update("session-1", func(s *session.Status) error { s.Session.LastTurn = index + 1; return nil }); err != nil { t.Fatal(err) }
				state, err = store.Load("task-1")
				if err != nil { t.Fatal(err) }
				request = state.Automation.PreparedRequest
				// Freeze the fake turn's request without invoking Git or an Agent.
				request.Compile.Request = &workflow.CompilerRequest{
					ActorRequest: workflow.ActorRequest{SchemaVersion: workflow.ActorContractSchemaVersion, ID: fmt.Sprintf("compiler-attempt-1-%d", index+1), Actor: workflow.ActorCompiler, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Workspace: root, PreparedAt: request.PreparedAt},
					ImplementationReport: workflow.ActorReference{Kind: "implementation-report", Path: outcome.EvidenceReference},
					Plan: request.Compile.Plan, CompileRoot: root,
				}
				request.Compile.RepairPending = false
				if err := saveSupervisedCompileRequest(store, request); err != nil { t.Fatal(err) }
				state, err = store.DispatchPreparedAgentRequest("task-1", "attempt-1")
				if err != nil { t.Fatal(err) }
				got, repair, err := compileSupervisedOutcome("task-1", store, state.Automation.PreparedRequest, outcome, runner)
				if err != nil { t.Fatal(err) }
				state, err = store.Load("task-1")
				if err != nil { t.Fatal(err) }
				if len(state.Attempts) != 1 || state.WorkItems[0].NoProgressCount != 0 || state.WriteLease == nil { t.Fatalf("repair changed Attempt/retry ownership: %+v", state) }
				if index < len(tc.codes)-1 {
					request = state.Automation.PreparedRequest
					if !repair || request.ExpectedSessionTurn != index+2 || request.DispatchedAt != "" || request.CompilerResult != nil { t.Fatalf("repair did not prepare the next bound turn: %+v", request) }
					if request.Compile.Request != nil || request.Compile.Result != nil || !request.Compile.RepairPending || request.Compile.Plan == nil || request.Compile.Failures != index+1 { t.Fatalf("repair lost plan state or retained stale results: %+v", request.Compile) }
					content, err := os.ReadFile(request.Handoff)
					if err != nil || !strings.Contains(string(content), "wi-0001") || !strings.Contains(string(content), "diagnostics") { t.Fatalf("repair handoff missing evidence: %s, %v", content, err) }
					continue
				}
				if repair { t.Fatal("terminal compile result requested another repair") }
				want := workflow.SupervisedOutcomeCompleted
				if tc.blocked { want = workflow.SupervisedOutcomeBlocked }
				if got.Kind != want { t.Fatalf("outcome = %+v, want %s", got, want) }
				if !tc.blocked && state.WorkItems[0].CompileFailureCount != 0 { t.Fatal("successful compilation did not reset failures") }
				if !tc.blocked && state.Automation.PreparedRequest.Compile.Failures != 0 { t.Fatal("successful compilation did not reset the Attempt compiler count") }
				if tc.blocked && (state.WorkItems[0].CompileFailureCount != 3 || workflow.NextRunnerOutcome(state).Kind != workflow.RunnerGate) { t.Fatal("third failure did not stop scheduling") }
				if _, err := store.RecordSupervisedOutcome("task-1", "attempt-1", got); err != nil { t.Fatal(err) }
			}
			if runner.calls != len(tc.codes) { t.Fatalf("compiler calls = %d", runner.calls) }
		})
	}
}

func TestSupervisedCompileRecoveryUsesPersistedSuccess(t *testing.T) {
	request := &workflow.PreparedAgentRequest{CompilerResult: &workflow.CompilerResult{ActorResult: workflow.ActorResult{Status: workflow.ActorResultAccepted}}}
	outcome := workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeCompleted, EvidenceReference: "output.txt"}
	got, repair, err := compileSupervisedOutcome("task-1", nil, request, outcome, unexpectedCompiler{t})
	if err != nil || repair || got != outcome {
		t.Fatalf("recovery changed the accepted outcome: %+v, repair=%v, err=%v", got, repair, err)
	}
}
