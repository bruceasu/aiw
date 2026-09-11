package workflow

import (
	"testing"
	"time"
)

func TestSupervisorLeaseAndObservation(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	if _, err := store.StartSupervisor("task-1", "lease-1"); err != nil { t.Fatal(err) }
	if _, err := store.StartSupervisor("task-1", "lease-2"); err == nil { t.Fatal("expected competing lease rejection") }
	updated, err := store.Load("task-1")
	if err != nil { t.Fatal(err) }
	if _, err := store.RecordSupervisorObservation("task-1", "lease-1", updated.LastEventSequence, "idle", ""); err != nil { t.Fatal(err) }
	updated, err = store.Load("task-1")
	if err != nil || SupervisorDue(updated, nowForTest()) { t.Fatalf("self observation should not be due: %v", err) }
	if _, err := store.StopSupervisor("task-1", "lease-1"); err != nil { t.Fatal(err) }
}

func TestSupervisorPausePersistsAcrossReload(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	if _, err := store.StartSupervisor("task-1", "lease-1"); err != nil { t.Fatal(err) }
	deadline := time.Now().UTC().Add(time.Minute)
	if _, err := store.PauseSupervisor("task-1", "lease-1", "repair-paused", "tasks.md", deadline); err != nil { t.Fatal(err) }
	reloaded, err := store.Load("task-1")
	if err != nil || reloaded.Automation.Supervisor.Result != "repair-paused" || SupervisorDue(reloaded, time.Now().UTC()) {
		t.Fatalf("pause reload = %#v, %v", reloaded.Automation.Supervisor, err)
	}
}

func TestSupervisorLeaseRenewalAndExpiredTakeover(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	if _, err := store.Create(state); err != nil { t.Fatal(err) }
	if _, err := store.StartSupervisor("task-1", "lease-1"); err != nil { t.Fatal(err) }
	renewed, err := store.RenewSupervisorLease("task-1", "lease-1")
	if err != nil || renewed.Automation.Supervisor.LeaseExpiresAt == "" { t.Fatalf("renew = %#v, %v", renewed.Automation.Supervisor, err) }
	if _, err := store.UpdateWithEvent("task-1", Event{Type: "test.expire"}, func(current *RuntimeState) error {
		current.Automation.Supervisor.LeaseExpiresAt = time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
		return nil
	}); err != nil { t.Fatal(err) }
	taken, err := store.StartSupervisor("task-1", "lease-2")
	if err != nil || taken.Automation.Supervisor.LeaseID != "lease-2" { t.Fatalf("takeover = %#v, %v", taken.Automation.Supervisor, err) }
	if _, err := store.Load("task-1"); err != nil { t.Fatal(err) }
}

func nowForTest() (result time.Time) { return time.Now().UTC() }
