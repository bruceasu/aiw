package workflow

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAcquireWriteLeaseRejectsAnotherAttempt(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", State: WorkItemReady}}
	state.Attempts = []Attempt{
		{ID: "attempt-1", WorkItemID: "wi-0001", Workspace: ".", State: AttemptCreated},
		{ID: "attempt-2", WorkItemID: "wi-0001", Workspace: ".", State: AttemptCreated},
	}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireWriteLease("task-1", "attempt-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireWriteLease("task-1", "attempt-2"); err == nil {
		t.Fatal("expected conflicting lease error")
	}
	if _, err := store.ReleaseWriteLease("task-1", "attempt-2"); err == nil {
		t.Fatal("expected non-owner release error")
	}
	loaded, err := store.ReleaseWriteLease("task-1", "attempt-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.WriteLease != nil {
		t.Fatal("expected released write lease")
	}
	events, err := store.readEvents("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Sequence != 1 || events[0].Type != "write-lease.acquired" || events[1].Sequence != 2 || events[1].Type != "write-lease.released" {
		t.Fatalf("unexpected lease events: %#v", events)
	}
}

func TestEnsureCompatibleCreatesInactiveRuntime(t *testing.T) {
	store := NewStore(t.TempDir())
	state, err := store.EnsureCompatible(compatibleState())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Attempts) != 0 || state.WriteLease != nil {
		t.Fatalf("compatible state inferred live execution: %+v", state)
	}
}

func TestAtomicWriteLeavesReadableReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := atomicWrite(path, []byte("before")); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(path, []byte("after")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "after" {
		t.Fatalf("got %q, want after", got)
	}
}

func TestUpdateRejectedChangePreservesStoredState(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.update("task-1", func(*RuntimeState) error {
		return errors.New("reject")
	}); err == nil {
		t.Fatal("expected update rejection")
	}
	loaded, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Planning != PlanningReady {
		t.Fatalf("got planning %q, want ready", loaded.Planning)
	}
}

func TestRecoverPendingEventAppendsOnlyPersistedEvent(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	pending := Event{SchemaVersion: SchemaVersion, Sequence: 1, Type: "work-item.planned", At: "2026-09-08T00:00:00Z"}
	if _, err := store.update("task-1", func(state *RuntimeState) error {
		state.PendingEvent = &pending
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	state, err := store.RecoverPendingEvent("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if state.PendingEvent != nil || state.LastEventSequence != 1 {
		t.Fatalf("unexpected recovered state: %#v", state)
	}
	events, err := store.readEvents("task-1")
	if err != nil || len(events) != 1 || events[0] != pending {
		t.Fatalf("unexpected recovered events: %#v, %v", events, err)
	}
}

func TestRecoverPendingEventDoesNotDuplicateExistingEvent(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	pending := Event{SchemaVersion: SchemaVersion, Sequence: 1, Type: "work-item.planned", At: "2026-09-08T00:00:00Z"}
	if err := store.appendEvent(pending, "task-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.update("task-1", func(state *RuntimeState) error {
		state.PendingEvent = &pending
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecoverPendingEvent("task-1"); err != nil {
		t.Fatal(err)
	}
	events, err := store.readEvents("task-1")
	if err != nil || len(events) != 1 {
		t.Fatalf("pending event was duplicated: %#v, %v", events, err)
	}
}

func TestDiagnoseReportsUnsequencedDuplicateAndMismatchedEvents(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	for _, event := range []Event{
		{SchemaVersion: SchemaVersion, Type: "legacy"},
		{SchemaVersion: SchemaVersion, Sequence: 1, Type: "first"},
		{SchemaVersion: SchemaVersion, Sequence: 1, Type: "duplicate"},
	} {
		if err := store.appendEvent(event, "task-1"); err != nil {
			t.Fatal(err)
		}
	}
	diagnostics, err := store.Diagnose("task-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, wanted := range []string{"event-unsequenced", "event-duplicate", "event-sequence-mismatch"} {
		if !hasDiagnostic(diagnostics, wanted) {
			t.Fatalf("missing %s in diagnostics: %#v", wanted, diagnostics)
		}
	}
}

func TestEnsureLegacyManagedWorkItemRecordsCompatibilityEvent(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	state, itemID, err := store.EnsureLegacyManagedWorkItem("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if itemID != "wi-0001" || len(state.WorkItems) != 1 {
		t.Fatalf("unexpected legacy item: %q %#v", itemID, state.WorkItems)
	}
	events, err := store.readEvents("task-1")
	if err != nil || len(events) != 1 || events[0].Sequence != 1 || events[0].Type != "work-item.compatibility-created" {
		t.Fatalf("unexpected compatibility event: %#v, %v", events, err)
	}
}

func hasDiagnostic(diagnostics []Diagnostic, wanted string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == wanted {
			return true
		}
	}
	return false
}

func compatibleState() RuntimeState {
	return NewCompatibleRuntime(
		TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary},
		PlanningReady,
		DeliveryUnmanaged,
	)
}
