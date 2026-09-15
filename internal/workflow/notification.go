package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// NotificationDispatcher is implemented by the adapter that invokes the
// aiw-notify Plugin. Workflow Core owns only the durable outbox protocol.
type NotificationDispatcher interface {
	Dispatch(context.Context, Notification) (receipt string, err error)
}

// EnqueueNotification persists an immutable notification before any Plugin is
// allowed to receive it. Repeating the same ID is idempotent only when its
// topic and payload match the original record.
func (s *Store) EnqueueNotification(id TaskID, notification Notification) (RuntimeState, error) {
	if err := validateNotificationRequest(notification); err != nil {
		return RuntimeState{}, err
	}
	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	for _, existing := range state.Notifications {
		if existing.ID != notification.ID {
			continue
		}
		if existing.Topic != notification.Topic || !jsonEqual(existing.Payload, notification.Payload) {
			return RuntimeState{}, fmt.Errorf("notification id %s is already bound to different content", notification.ID)
		}
		return state, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	notification.State = NotificationPending
	notification.CreatedAt = now
	notification.UpdatedAt = now
	notification.DispatchAttempts = 0
	notification.LastError = ""
	notification.Receipt = ""
	return s.UpdateWithEvent(id, Event{Type: "notification.enqueued", Detail: string(notification.ID)}, func(state *RuntimeState) error {
		state.Notifications = append(state.Notifications, notification)
		return nil
	})
}

// PrepareNotificationDispatch records the attempt before an external Plugin
// starts. Failed records consume only the notification retry budget; an
// initial pending dispatch consumes no retry.
func (s *Store) PrepareNotificationDispatch(id TaskID, notificationID NotificationID) (RuntimeState, Notification, error) {
	if notificationID == "" {
		return RuntimeState{}, Notification{}, fmt.Errorf("notification id is required")
	}
	var prepared Notification
	state, err := s.UpdateWithEvent(id, Event{Type: "notification.dispatch-prepared", Detail: string(notificationID)}, func(state *RuntimeState) error {
		for index := range state.Notifications {
			notification := &state.Notifications[index]
			if notification.ID != notificationID {
				continue
			}
			if notification.State == NotificationDispatching {
				return fmt.Errorf("notification %s already has a dispatch in progress; recover it first", notificationID)
			}
			if notification.State == NotificationDelivered {
				return fmt.Errorf("notification %s is already delivered", notificationID)
			}
			if notification.State != NotificationPending && notification.State != NotificationFailed {
				return fmt.Errorf("notification %s cannot be dispatched from %s", notificationID, notification.State)
			}
			if notification.State == NotificationFailed {
				counter := &state.OperationalRetries.Notification
				if counter.Used >= counter.MaxAttempts {
					return fmt.Errorf("notification retry limit of %d is exhausted", counter.MaxAttempts)
				}
				counter.Used++
			}
			notification.State = NotificationDispatching
			notification.DispatchAttempts++
			notification.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			notification.LastError = ""
			prepared = *notification
			return nil
		}
		return fmt.Errorf("unknown notification %s", notificationID)
	})
	return state, prepared, err
}

// CompleteNotificationDispatch records the Plugin result. A failed dispatch
// leaves only this outbox item eligible for a later independent retry.
func (s *Store) CompleteNotificationDispatch(id TaskID, notificationID NotificationID, receipt, failure string) (RuntimeState, error) {
	if notificationID == "" {
		return RuntimeState{}, fmt.Errorf("notification id is required")
	}
	if strings.TrimSpace(receipt) == "" && strings.TrimSpace(failure) == "" {
		return RuntimeState{}, fmt.Errorf("notification receipt or failure is required")
	}
	if strings.TrimSpace(receipt) != "" && strings.TrimSpace(failure) != "" {
		return RuntimeState{}, fmt.Errorf("notification cannot succeed and fail together")
	}
	eventType := "notification.dispatch-failed"
	if receipt != "" {
		eventType = "notification.delivered"
	}
	return s.UpdateWithEvent(id, Event{Type: eventType, Detail: string(notificationID)}, func(state *RuntimeState) error {
		for index := range state.Notifications {
			notification := &state.Notifications[index]
			if notification.ID != notificationID {
				continue
			}
			if notification.State != NotificationDispatching {
				return fmt.Errorf("notification %s is not dispatching", notificationID)
			}
			notification.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if receipt != "" {
				notification.State = NotificationDelivered
				notification.Receipt = receipt
			} else {
				notification.State = NotificationFailed
				notification.LastError = failure
			}
			return nil
		}
		return fmt.Errorf("unknown notification %s", notificationID)
	})
}

// RecoverNotificationDispatch makes an interrupted Plugin invocation
// retryable. Its stable ID is retained because the Plugin may have received it
// before the process stopped.
func (s *Store) RecoverNotificationDispatch(id TaskID, notificationID NotificationID) (RuntimeState, error) {
	return s.CompleteNotificationDispatch(id, notificationID, "", "Plugin dispatch interrupted; retry with the same notification id")
}

// DispatchNotification is the adapter-facing outbox flow. It never reruns the
// completed workflow transition that created the notification.
func (s *Store) DispatchNotification(ctx context.Context, id TaskID, notificationID NotificationID, dispatcher NotificationDispatcher) (RuntimeState, error) {
	if dispatcher == nil {
		return RuntimeState{}, fmt.Errorf("notification dispatcher is required")
	}
	_, notification, err := s.PrepareNotificationDispatch(id, notificationID)
	if err != nil {
		return RuntimeState{}, err
	}
	receipt, dispatchErr := dispatcher.Dispatch(ctx, notification)
	if dispatchErr != nil {
		return s.CompleteNotificationDispatch(id, notificationID, "", dispatchErr.Error())
	}
	return s.CompleteNotificationDispatch(id, notificationID, receipt, "")
}

func validateNotificationRequest(notification Notification) error {
	if notification.ID == "" || strings.TrimSpace(notification.Topic) == "" {
		return fmt.Errorf("notification id and topic are required")
	}
	if !json.Valid(notification.Payload) {
		return fmt.Errorf("notification payload must be valid JSON")
	}
	return nil
}

func jsonEqual(left, right json.RawMessage) bool {
	var leftValue any
	var rightValue any
	return json.Unmarshal(left, &leftValue) == nil && json.Unmarshal(right, &rightValue) == nil && reflect.DeepEqual(leftValue, rightValue)
}
