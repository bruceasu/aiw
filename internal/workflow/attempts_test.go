package workflow

import (
	"fmt"
	"testing"
)

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

func TestRecordSupervisedBlockedOutcomeCreatesGateWithoutRetry(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "work", State: WorkItemReady, RetryPolicy: RetryPolicy{MaxAttempts: 3}}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: "."}); err != nil {
		t.Fatal(err)
	}

	updated, err := store.RecordSupervisedOutcome("task-1", "attempt-1", SupervisedOutcome{Kind: SupervisedOutcomeBlocked, BlockedCategory: BlockedOutcomeDependency, Detail: "waiting for prerequisite", EvidenceReference: "output.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.WorkItems[0].State != WorkItemBlocked || updated.WorkItems[0].NoProgressCount != 0 {
		t.Fatalf("blocked outcome = %#v", updated.WorkItems[0])
	}
	if len(updated.Gates) != 1 || updated.Gates[0].Kind != GateDependency {
		t.Fatalf("blocked outcome Gates = %#v", updated.Gates)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-2", WorkItemID: "wi-0001", Workspace: "."}); err == nil {
		t.Fatal("blocked outcome must reject automatic retry")
	}
	if next := NextRunnerOutcome(updated); next.Kind != RunnerGate {
		t.Fatalf("blocked outcome was scheduled: %+v", next)
	}
}

func TestSupervisedNoProgressRetriesUntilPolicyLimit(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", State: WorkItemReady, RetryPolicy: RetryPolicy{MaxAttempts: 2}}}
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	for n := 1; n <= 2; n++ {
		attempt := AttemptID(fmt.Sprintf("attempt-%d", n))
		if _, err := store.StartAttempt("task-1", Attempt{ID: attempt, WorkItemID: "wi-0001", Workspace: "."}); err != nil { t.Fatal(err) }
		updated, err := store.RecordSupervisedOutcome("task-1", attempt, SupervisedOutcome{Kind: SupervisedOutcomeNoProgress, Detail: "no changes", EvidenceReference: "output.txt"})
		if err != nil { t.Fatal(err) }
		want := WorkItemReady
		if n == 2 { want = WorkItemBlocked }
		if updated.WorkItems[0].State != want || updated.WorkItems[0].NoProgressCount != n || updated.WriteLease != nil {
			t.Fatalf("attempt %d did not follow retry policy: %+v", n, updated)
		}
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-3", WorkItemID: "wi-0001", Workspace: "."}); err == nil {
		t.Fatal("no-progress retried beyond its configured limit")
	}
}

func TestReopenBlockedOutcomeRequiresResolvedGateWithoutResettingRetryCount(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "work", State: WorkItemReady, RetryPolicy: RetryPolicy{MaxAttempts: 3}}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: "."}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordAttemptOutcome("task-1", "attempt-1", false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-2", WorkItemID: "wi-0001", Workspace: "."}); err != nil {
		t.Fatal(err)
	}
	blocked, err := store.RecordSupervisedOutcome("task-1", "attempt-2", SupervisedOutcome{Kind: SupervisedOutcomeBlocked, BlockedCategory: BlockedOutcomeDependency, Detail: "waiting for prerequisite", EvidenceReference: "output.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReopenWorkItem("task-1", "wi-0001", "reviewed"); err == nil {
		t.Fatal("expected reopen to reject an unresolved Gate")
	}
	if _, err := store.ResolveGate("task-1", blocked.Gates[0].ID, GateResolved); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.ReopenWorkItem("task-1", "wi-0001", "dependency resolved")
	if err != nil {
		t.Fatal(err)
	}
	if reopened.WorkItems[0].State != WorkItemReady || reopened.WorkItems[0].NoProgressCount != 1 {
		t.Fatalf("reopened blocked outcome = %#v", reopened.WorkItems[0])
	}
}

func TestSuccessfulCompatibilityOutcomeDoesNotConsumeNoProgress(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", Title: "work", State: WorkItemReady}}
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	if _, err := store.StartAttempt("task-1", Attempt{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: "."}); err != nil { t.Fatal(err) }
	updated, err := store.RecordAttemptOutcome("task-1", "attempt-1", true)
	if err != nil { t.Fatal(err) }
	if updated.WorkItems[0].NoProgressCount != 0 || updated.WorkItems[0].State != WorkItemReady {
		t.Fatalf("successful outcome consumed retry state: %#v", updated.WorkItems[0])
	}
}
