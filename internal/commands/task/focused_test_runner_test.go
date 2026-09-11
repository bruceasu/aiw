package task

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

type countingFocusedTestProcess struct {
	calls int
}

func (p *countingFocusedTestProcess) Run(context.Context, FocusedTestRun) ([]byte, int, error) {
	p.calls++
	return nil, 0, nil
}

func TestFocusedTestCommandDoesNotStartProcessBeforeValidation(t *testing.T) {
	tests := []struct {
		name          string
		prepare       func(t *testing.T, fixture focusedTestCommandFixture)
		wantErrorPart string
	}{
		{
			name:          "missing authorization",
			wantErrorPart: "requires authorization",
		},
		{
			name: "stale authorization",
			prepare: func(t *testing.T, fixture focusedTestCommandFixture) {
				t.Helper()
				oldPlan := fixture.plan
				oldPlan.Checks[0].Argv = []string{"go", "test", "./internal/taskx"}
				oldDigest, err := oldPlan.Digest()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := fixture.store.ActivateFocusedTestPlan(fixture.id, oldDigest); err != nil {
					t.Fatal(err)
				}
				if _, err := fixture.store.AuthorizeFocusedTest(fixture.id, oldDigest, "reviewer@example.test"); err != nil {
					t.Fatal(err)
				}
			},
			wantErrorPart: "requires authorization",
		},
		{
			name: "invalid plan",
			prepare: func(t *testing.T, fixture focusedTestCommandFixture) {
				t.Helper()
				fixture.plan.Checks[0].NetworkPolicy = "allow"
				fixture.writePlan(t)
			},
			wantErrorPart: "parse verification plan",
		},
		{
			name: "invalid selection",
			prepare: func(t *testing.T, fixture focusedTestCommandFixture) {
				t.Helper()
				fixture.selection.CheckID = "not-in-plan"
				fixture.writeSelection(t)
			},
			wantErrorPart: "parse focused-test selection",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFocusedTestCommandFixture(t)
			if test.prepare != nil {
				test.prepare(t, fixture)
			}
			process := &countingFocusedTestProcess{}
			_, err := runFocusedTestCommand(fixture.meta, fixture.state(t), fixture.store, fixture.attemptID, nil, process)
			if err == nil || !strings.Contains(err.Error(), test.wantErrorPart) {
				t.Fatalf("runFocusedTestCommand() error = %v, want %q", err, test.wantErrorPart)
			}
			if process.calls != 0 {
				t.Fatalf("process calls = %d, want 0", process.calls)
			}
		})
	}
}

type focusedTestCommandFixture struct {
	id        workflow.TaskID
	attemptID workflow.AttemptID
	meta      taskx.TaskMeta
	store     *workflow.Store
	plan      workflow.VerificationPlan
	selection workflow.VerificationPlanSelection
}

func newFocusedTestCommandFixture(t *testing.T) focusedTestCommandFixture {
	t.Helper()
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Error(err)
		}
	})

	const id = workflow.TaskID("task-1")
	worktree := filepath.Join(root, "attempt-worktree")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := focusedTestCommandFixture{
		id:        id,
		attemptID: "attempt-1",
		meta:      taskx.TaskMeta{ID: string(id), Type: "task", Worktree: worktree, WorkspaceKind: "isolated"},
		store:     workflow.NewStore(""),
		plan: workflow.VerificationPlan{
			SchemaVersion: workflow.VerificationPlanSchemaVersion,
			TaskID:        id,
			Checks: []workflow.VerificationCheck{{
				CheckID: "workflow-tests", Argv: []string{"go", "test", "./internal/workflow"}, WorkingDirectory: ".",
				TimeoutSeconds: 60, ExpectedExitCode: 0, EvidenceDestination: "workflow", NetworkPolicy: workflow.NetworkPolicyDeny,
				Profile: workflow.FocusedTestProfile,
			}},
		},
	}
	if err := os.MkdirAll(filepath.Dir(taskx.TaskMetaPath(string(id))), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath(string(id)), fixture.meta); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.EnsureCompatible(taskx.WorkflowRuntimeFromMeta(fixture.meta)); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.SyncChecklist(id, []workflow.ChecklistCandidate{{Item: "1.1", Title: "focused validation"}}, "fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.StartAttempt(id, workflow.Attempt{ID: fixture.attemptID, WorkItemID: "wi-0001", Workspace: worktree}); err != nil {
		t.Fatal(err)
	}
	digest, err := fixture.plan.Digest()
	if err != nil {
		t.Fatal(err)
	}
	fixture.selection = workflow.VerificationPlanSelection{
		SchemaVersion: workflow.VerificationPlanSelectionSchemaVersion,
		PlanDigest: digest, CheckID: "workflow-tests", Rationale: "covers the changed workflow code",
		EvidenceReferences: []string{"work-item:wi-0001"},
	}
	fixture.writePlan(t)
	fixture.writeSelection(t)
	return fixture
}

func (f focusedTestCommandFixture) state(t *testing.T) workflow.RuntimeState {
	t.Helper()
	state, err := f.store.Load(f.id)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func (f focusedTestCommandFixture) writePlan(t *testing.T) {
	t.Helper()
	data, err := json.Marshal(f.plan)
	if err != nil {
		t.Fatal(err)
	}
	path := taskx.VerificationPlanPath(string(f.id))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f focusedTestCommandFixture) writeSelection(t *testing.T) {
	t.Helper()
	data, err := json.Marshal(f.selection)
	if err != nil {
		t.Fatal(err)
	}
	path := taskx.VerificationSelectionPath(string(f.id))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
