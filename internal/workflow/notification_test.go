package workflow

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type notificationDispatcherStub struct {
	calls int
	outcome string
	config NotificationConfig
}

func (d *notificationDispatcherStub) Dispatch(context.Context, Notification) (string, error) { panic("legacy dispatch must not run") }
func (d *notificationDispatcherStub) Preflight(context.Context, Notification) (NotificationConfig, error) { return d.config, nil }
func (d *notificationDispatcherStub) Send(_ context.Context, n Notification) NotificationResult {
	d.calls++
	r := UnknownNotificationResult(n)
	r.Outcome, r.ErrorCode = d.outcome, ""
	return r
}

func notificationFixture(t *testing.T) (*Store, Notification, *notificationDispatcherStub) {
	t.Helper()
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil { t.Fatal(err) }
	n := Notification{ID: "notification-1", Topic: "workflow.completed", Payload: json.RawMessage(`{"fact":"complete"}`), Managed: &NotificationMessage{Version: 2, FactID: "complete", Text: "Task completed", ContentDigest: contentDigest([]byte("Task completed"))}}
	if _, err := store.EnqueueNotification("task-1", n); err != nil { t.Fatal(err) }
	d := &notificationDispatcherStub{outcome: "network_failure", config: NotificationConfig{Enabled: true, Ready: true, Channel: "console", TargetReference: "target", ConfigReference: "config"}}
	return store, n, d
}

func TestNotificationNetworkRetryKeepsDeadlineAndMessageBudget(t *testing.T) {
	store, n, dispatcher := notificationFixture(t)
	before := time.Now()
	state, err := store.DispatchNotification(context.Background(), "task-1", n.ID, dispatcher)
	if err != nil { t.Fatal(err) }
	first := state.Notifications[0]
	due, err := time.Parse(time.RFC3339, first.Managed.RetryDueAt)
	if err != nil || due.Before(before.Add(time.Minute)) || first.DispatchAttempts != 1 { t.Fatalf("invalid retry reservation: %+v", first) }
	reloaded := NewStore(store.Root)
	if _, err := reloaded.DispatchNotification(context.Background(), "task-1", n.ID, dispatcher); err == nil { t.Fatal("early retry permitted") }
	state, err = reloaded.Load("task-1")
	if err != nil { t.Fatal(err) }
	if state.Notifications[0].Managed.RetryDueAt != first.Managed.RetryDueAt || dispatcher.calls != 1 { t.Fatal("restart reset deadline or sent early") }
	// Advance the saved deadline instead of sleeping or using real transports.
	if _, err := reloaded.notificationUpdate("task-1", func(s *RuntimeState) error { s.Notifications[0].Managed.RetryDueAt = time.Now().Add(-time.Second).UTC().Format(time.RFC3339); return nil }); err != nil { t.Fatal(err) }
	state, err = reloaded.DispatchNotification(context.Background(), "task-1", n.ID, dispatcher)
	if err != nil { t.Fatal(err) }
	if state.Notifications[0].DispatchAttempts != 2 || state.Notifications[0].Managed.RetryDueAt != "" || state.OperationalRetries.Notification.Used != 0 { t.Fatal("message budget not independent") }
	if _, err := reloaded.DispatchNotification(context.Background(), "task-1", n.ID, dispatcher); err == nil || dispatcher.calls != 2 { t.Fatal("third send permitted") }
}

func TestNotificationOtherResultsNeverRetry(t *testing.T) {
	for _, outcome := range []string{"unknown", "permission_failure", "service_failure", "configuration_failure", "attempted"} {
		t.Run(outcome, func(t *testing.T) {
			store, n, d := notificationFixture(t)
			d.outcome = outcome
			state, err := store.DispatchNotification(context.Background(), "task-1", n.ID, d)
			if err != nil { t.Fatal(err) }
			if state.Notifications[0].Managed.RetryDueAt != "" || state.Notifications[0].State == NotificationDelivered { t.Fatal("unexpected retry or delivery claim") }
			if _, err := store.DispatchNotification(context.Background(), "task-1", n.ID, d); err == nil || d.calls != 1 { t.Fatal("non-network result retried") }
		})
	}
}

func TestNotificationDisabledAndFrozenTarget(t *testing.T) {
	store, n, d := notificationFixture(t)
	d.config.Ready = false
	state, err := store.DispatchNotification(context.Background(), "task-1", n.ID, d)
	if err != nil || state.Notifications[0].DispatchAttempts != 0 || d.calls != 0 { t.Fatal("missing configuration consumed attempt") }
	d.config.Ready = true
	if _, err := store.DispatchNotification(context.Background(), "task-1", n.ID, d); err != nil { t.Fatal(err) }
	d.config.ConfigReference = "changed-target"
	state, err = store.DispatchNotification(context.Background(), "task-1", n.ID, d)
	if err != nil || state.Notifications[0].Managed.Config.ConfigReference != "config" || d.calls != 1 { t.Fatal("frozen target replaced") }
	d.config.Enabled = false
	state, err = store.DispatchNotification(context.Background(), "task-1", n.ID, d)
	if err != nil || state.Notifications[0].State != "cancelled" { t.Fatal("disabled completion not cancelled") }
	d.config.Enabled = true
	if _, err := store.DispatchNotification(context.Background(), "task-1", n.ID, d); err == nil || d.calls != 1 { t.Fatal("historical completion replayed") }
}

func TestNotificationInterruptedResultIsUnknown(t *testing.T) {
	store, n, d := notificationFixture(t)
	if _, err := store.notificationUpdate("task-1", func(s *RuntimeState) error { s.Notifications[0].Managed.Config = d.config; return nil }); err != nil { t.Fatal(err) }
	if _, _, err := store.PrepareNotificationDispatch("task-1", n.ID); err != nil { t.Fatal(err) }
	state, err := store.RecoverNotificationDispatch("task-1", n.ID)
	if err != nil { t.Fatal(err) }
	if state.Notifications[0].State != "unknown" || state.Notifications[0].DispatchAttempts != 1 { t.Fatal("unknown reservation released") }
	if _, err := store.DispatchNotification(context.Background(), "task-1", n.ID, d); err == nil || d.calls != 0 { t.Fatal("interrupted send retried") }
}
