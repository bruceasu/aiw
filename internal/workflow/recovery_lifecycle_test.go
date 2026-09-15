package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCompatibleMigratesLegacyRuntimeExactlyOnce(t *testing.T) {
	store := NewStore(t.TempDir())
	legacy := compatibleState()
	legacy.SchemaVersion = 1
	legacy.WorkItems = []WorkItem{{ID: "wi-0001", Title: "legacy", State: WorkItemReady, RetryPolicy: RetryPolicy{MaxAttempts: DefaultRetryLimit}}}
	legacyBytes, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.legacyTaskDir("task-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.legacyTaskDir("task-1"), runtimeStateFile), legacyBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.legacyTaskDir("task-1"), runtimeEventsFile), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	migrated, err := store.EnsureCompatible(compatibleState())
	if err != nil {
		t.Fatal(err)
	}
	if migrated.SchemaVersion != SchemaVersion || migrated.WorkItems[0].Title != "legacy" {
		t.Fatalf("migration lost compatible runtime state: %#v", migrated)
	}
	if _, err := os.Stat(filepath.Join(store.taskDir("task-1"), runtimeStateFile)); err != nil {
		t.Fatalf("canonical state was not created: %v", err)
	}
	marker, err := os.ReadFile(filepath.Join(store.legacyTaskDir("task-1"), legacyMigrationFile))
	if err != nil || string(marker) != store.taskDir("task-1")+"\n" {
		t.Fatalf("legacy migration marker = %q, %v", marker, err)
	}
	if _, err := store.EnsureCompatible(compatibleState()); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	events, err := store.readEvents("task-1")
	if err != nil || len(events) != 1 || events[0].Type != "runtime.migrated" {
		t.Fatalf("migration event must be recorded once: %#v, %v", events, err)
	}
}

func TestOperationalRetriesAreIndependentAndBounded(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetOperationalRetryLimit("task-1", RetryNotification, 1); err != nil {
		t.Fatal(err)
	}
	state, err := store.ConsumeOperationalRetry("task-1", RetryNotification)
	if err != nil {
		t.Fatal(err)
	}
	if state.OperationalRetries.Notification.Used != 1 || state.OperationalRetries.Recovery.Used != 0 || state.OperationalRetries.Delivery.Used != 0 {
		t.Fatalf("operational retries are not independent: %#v", state.OperationalRetries)
	}
	if _, err := store.ConsumeOperationalRetry("task-1", RetryNotification); err == nil {
		t.Fatal("expected exhausted notification retry budget")
	}
	state, err = store.ConsumeOperationalRetry("task-1", RetryDelivery)
	if err != nil || state.OperationalRetries.Delivery.Used != 1 {
		t.Fatalf("delivery retry must remain available: %#v, %v", state.OperationalRetries, err)
	}
}

func TestForceClosePreservesEvidenceWithoutClaimingSkippedStagesPassed(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.Evidence = []Evidence{
		{ID: "compile", Kind: EvidenceCommand, State: EvidencePassed, Reference: "compile-only"},
		{ID: "tests", Kind: EvidenceCommand, State: EvidencePending, Reference: "not-run"},
	}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	closed, err := store.ForceClose("task-1", "operator stopped validation", DeliveryDiscarded)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Evidence[0].State != EvidencePassed || closed.Evidence[1].State != EvidencePending {
		t.Fatalf("force-close changed evidence or fabricated result: %#v", closed.Evidence)
	}
	if closed.Cancellation == nil || closed.Delivery != DeliveryPending {
		t.Fatalf("force-close did not retain pending delivery disposition: %#v", closed)
	}
}
