package task

import (
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestPreparedManagedAttemptRequiresMatchingBinding(t *testing.T) {
	attemptID := workflow.AttemptID("attempt-1")
	state := workflow.RuntimeState{
		Automation: workflow.AutomationState{PreparedRequest: &workflow.PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: attemptID, SessionID: "session-1", Workspace: "."}},
		WriteLease: &workflow.WriteLease{AttemptID: attemptID, Workspace: "."},
		Attempts:   []workflow.Attempt{{ID: attemptID, WorkItemID: "wi-0001", State: workflow.AttemptRunning}},
	}
	got, prepared, err := preparedManagedAttempt("task-1", taskx.TaskMeta{Session: "session-1", Worktree: "."}, state)
	if err != nil || !prepared || got != attemptID {
		t.Fatalf("prepared attempt = %q, %t, %v", got, prepared, err)
	}
	_, _, err = preparedManagedAttempt("task-1", taskx.TaskMeta{Session: "other", Worktree: "."}, state)
	if err == nil {
		t.Fatal("expected Session mismatch to fail")
	}
}
