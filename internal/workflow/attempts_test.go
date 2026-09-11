package workflow

import "testing"

func TestAttemptFailureAllowsRetryAfterLeaseRelease(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", State: WorkItemReady}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: ".", State: AttemptCreated}); err != nil {
		t.Fatal(err)
	}
	failed, err := store.RecordAttemptOutcome("task-1", "attempt-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if failed.WriteLease != nil || failed.WorkItems[0].State != WorkItemReady || failed.Attempts[0].State != AttemptFailed {
		t.Fatalf("failed attempt did not become retryable: %+v", failed)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-2", WorkItemID: "wi-0001", Workspace: ".", State: AttemptCreated}); err != nil {
		t.Fatalf("retry attempt: %v", err)
	}
}

func TestAttemptExhaustionBlocksWorkItemAndReopenResetsOnlyThatItem(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{
		{ID: "wi-0001", State: WorkItemReady, RetryPolicy: RetryPolicy{MaxAttempts: 1}},
		{ID: "wi-0002", State: WorkItemReady, RetryPolicy: RetryPolicy{MaxAttempts: 3}, NoProgressCount: 2},
	}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: ".", State: AttemptCreated}); err != nil {
		t.Fatal(err)
	}

	exhausted, err := store.RecordAttemptOutcome("task-1", "attempt-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if exhausted.WriteLease != nil || exhausted.WorkItems[0].State != WorkItemBlocked || exhausted.WorkItems[0].NoProgressCount != 1 {
		t.Fatalf("exhausted work item retained automatic execution ownership: %+v", exhausted)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-2", WorkItemID: "wi-0001", Workspace: ".", State: AttemptCreated}); err == nil {
		t.Fatal("expected exhausted work item to reject another attempt")
	}

	reopened, err := store.ReopenWorkItem("task-1", "wi-0001", "operator reviewed the failure")
	if err != nil {
		t.Fatal(err)
	}
	if reopened.WorkItems[0].State != WorkItemReady || reopened.WorkItems[0].NoProgressCount != 0 {
		t.Fatalf("reopen did not reset exhausted item: %+v", reopened.WorkItems[0])
	}
	if reopened.WorkItems[1].NoProgressCount != 2 {
		t.Fatalf("reopen reset an unrelated item: %+v", reopened.WorkItems[1])
	}
}
