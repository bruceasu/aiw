package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type notificationDispatcherStub struct {
	calls int
	err   error
}

func (d *notificationDispatcherStub) Dispatch(_ context.Context, notification Notification) (string, error) {
	d.calls++
	if d.err != nil {
		return "", d.err
	}
	return "receipt-" + string(notification.ID), nil
}

func TestDispatchNotificationFailureRetriesOnlyOutboxRecord(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create(compatibleState()); err != nil {
		t.Fatal(err)
	}
	notification := Notification{ID: "notification-1", Topic: "workflow.completed", Payload: json.RawMessage(`{"task":"task-1"}`)}
	if _, err := store.EnqueueNotification("task-1", notification); err != nil {
		t.Fatal(err)
	}
	dispatcher := &notificationDispatcherStub{err: errors.New("Plugin unavailable")}
	failed, err := store.DispatchNotification(context.Background(), "task-1", notification.ID, dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	if dispatcher.calls != 1 || failed.Notifications[0].State != NotificationFailed || failed.OperationalRetries.Notification.Used != 0 {
		t.Fatalf("first dispatch state = %+v", failed)
	}
	dispatcher.err = nil
	delivered, err := store.DispatchNotification(context.Background(), "task-1", notification.ID, dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	got := delivered.Notifications[0]
	if dispatcher.calls != 2 || got.State != NotificationDelivered || got.Receipt != "receipt-notification-1" {
		t.Fatalf("retry result = %+v, calls=%d", got, dispatcher.calls)
	}
	if delivered.OperationalRetries.Notification.Used != 1 || delivered.OperationalRetries.Delivery.Used != 0 || delivered.OperationalRetries.Recovery.Used != 0 {
		t.Fatalf("retry accounting was not independent: %+v", delivered.OperationalRetries)
	}
}
