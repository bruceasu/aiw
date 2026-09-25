package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

// Legacy adapters remain source-compatible, but cannot dispatch managed text.
type NotificationDispatcher interface {
	Dispatch(context.Context, Notification) (string, error)
}

type ManagedNotificationDispatcher interface {
	Preflight(context.Context, Notification) (NotificationConfig, error)
	Send(context.Context, Notification) NotificationResult
}

type NotificationConfig struct {
	Enabled bool `json:"enabled"`
	Ready bool `json:"ready"`
	Channel string `json:"channel"`
	TargetReference string `json:"target_reference"`
	ConfigReference string `json:"config_reference"`
}

type NotificationResult struct {
	SchemaVersion int `json:"schema_version"`
	NotificationID NotificationID `json:"notification_id"`
	AttemptID string `json:"attempt_id"`
	ContentDigest string `json:"content_digest"`
	Attempted bool `json:"attempted"`
	Outcome string `json:"outcome"`
	ErrorCode string `json:"error_code"`
	HTTPStatus *int `json:"http_status"`
}

type NotificationAttempt struct {
	ID string `json:"id"`
	ReservedAt string `json:"reserved_at"`
	ObservedAt string `json:"observed_at,omitempty"`
	RetryDueAt string `json:"retry_due_at,omitempty"`
	Result *NotificationResult `json:"result,omitempty"`
}

type NotificationMessage struct {
	Version int `json:"version"`
	FactID string `json:"fact_id"`
	ContentDigest string `json:"content_digest"`
	Text string `json:"text"`
	GateID GateID `json:"gate_id,omitempty"`
	Related NotificationID `json:"related,omitempty"`
	Config NotificationConfig `json:"config"`
	ConfigSnapshot string `json:"config_snapshot,omitempty"`
	Attempts []NotificationAttempt `json:"attempts,omitempty"`
	RetryDueAt string `json:"retry_due_at,omitempty"`
	LastOutcome string `json:"last_outcome,omitempty"`
}

func (s *Store) notificationUpdate(id TaskID, change func(*RuntimeState) error) (RuntimeState, error) {
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, err }
	if state.SchemaVersion == DurableSchemaVersion {
		return s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.notification"}, change)
	}
	return s.UpdateWithEvent(id, Event{Type: "notification.updated"}, change)
}

func (s *Store) EnqueueNotification(id TaskID, n Notification) (RuntimeState, error) {
	if err := validateNotificationRequest(n); err != nil { return RuntimeState{}, err }
	return s.notificationUpdate(id, func(state *RuntimeState) error {
		for _, old := range state.Notifications {
			if n.Managed != nil && old.Managed != nil && old.Managed.FactID == n.Managed.FactID {
				// Rendering changes and replacement IDs cannot create a new
				// budget for the same source fact. A substantive update needs
				// its own immutable source identity and related notification.
				return errProtocolNoChange
			}
			if old.ID != n.ID { continue }
			if old.Topic != n.Topic || !jsonEqual(old.Payload, n.Payload) { return errors.New("notification identity already bound") }
			return errProtocolNoChange
		}
		n.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		n.UpdatedAt, n.State, n.DispatchAttempts = n.CreatedAt, NotificationPending, 0
		n.LastError, n.Receipt = "", ""
		if n.Managed != nil { n.Managed.Attempts = nil; n.Managed.RetryDueAt = ""; n.Managed.LastOutcome = "" }
		state.Notifications = append(state.Notifications, n)
		return nil
	})
}

func notificationActive(state RuntimeState, n Notification) bool {
	if n.Managed == nil { return false }
	if n.Managed.GateID == "" { return true }
	for _, gate := range state.Gates { if gate.ID == n.Managed.GateID { return gate.State == GateOpen } }
	return false
}

func (s *Store) PrepareNotificationDispatch(id TaskID, notificationID NotificationID) (RuntimeState, Notification, error) {
	var prepared Notification
	state, err := s.notificationUpdate(id, func(state *RuntimeState) error {
		if state.Protocol != nil && state.Protocol.Stop != nil { return errors.New("notification dispatch stopped") }
		for i := range state.Notifications {
			n := &state.Notifications[i]
			if n.ID != notificationID { continue }
			m := n.Managed
			if m == nil || !m.Config.Enabled || !m.Config.Ready || !notificationActive(*state, *n) { return errors.New("notification is not eligible") }
			if n.State != NotificationPending && n.State != "waiting" && n.State != NotificationFailed { return errors.New("notification already settled or in flight") }
			if n.DispatchAttempts >= 2 { return errors.New("notification attempt limit reached") }
			if m.LastOutcome != "" && m.LastOutcome != "network_failure" && m.LastOutcome != "configuration_failure" { return errors.New("notification result is terminal") }
			if n.State == NotificationFailed && m.LastOutcome != "network_failure" { return errors.New("non-network failure cannot be retried") }
			if n.DispatchAttempts > 0 {
				due, err := time.Parse(time.RFC3339, m.RetryDueAt)
				if m.LastOutcome != "network_failure" || err != nil || time.Now().Before(due) { return errors.New("notification retry is not due") }
			}
			n.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			m.Attempts = append(m.Attempts, NotificationAttempt{ID: fmt.Sprintf("%s:%d", n.ID, len(m.Attempts)+1), ReservedAt: n.UpdatedAt})
			m.RetryDueAt = ""
			n.DispatchAttempts++
			n.State, n.LastError = NotificationDispatching, ""
			prepared = *n
			return nil
		}
		return errors.New("unknown notification")
	})
	return state, prepared, err
}

func UnknownNotificationResult(n Notification) NotificationResult {
	r := NotificationResult{SchemaVersion: 2, NotificationID: n.ID, Attempted: true, Outcome: "unknown", ErrorCode: "protocol"}
	if n.Managed != nil {
		r.ContentDigest = n.Managed.ContentDigest
		if count := len(n.Managed.Attempts); count > 0 { r.AttemptID = n.Managed.Attempts[count-1].ID }
	}
	return r
}

func ValidNotificationResult(n Notification, r NotificationResult) bool {
	if n.Managed == nil || len(n.Managed.Attempts) == 0 { return false }
	if r.SchemaVersion != 2 || r.NotificationID != n.ID || r.ContentDigest != n.Managed.ContentDigest || r.AttemptID != n.Managed.Attempts[len(n.Managed.Attempts)-1].ID { return false }
	if r.HTTPStatus != nil && (*r.HTTPStatus < 100 || *r.HTTPStatus > 599) { return false }
	switch r.ErrorCode { case "", "configuration", "permission", "network", "service", "protocol", "timeout", "tls": default: return false }
	switch r.Outcome {
	case "attempted", "network_failure", "service_failure": return r.Attempted
	case "configuration_failure", "permission_failure": return true
	case "unknown": return r.Attempted
	default: return false
	}
}

func (s *Store) ObserveNotification(id TaskID, notificationID NotificationID, result NotificationResult) (RuntimeState, error) {
	return s.notificationUpdate(id, func(state *RuntimeState) error {
		for i := range state.Notifications {
			n := &state.Notifications[i]
			if n.ID != notificationID { continue }
			if n.Managed == nil || n.State != NotificationDispatching { return errors.New("notification is not dispatching") }
			m := n.Managed
			if !ValidNotificationResult(*n, result) { result = UnknownNotificationResult(*n) }
			m.Attempts[len(m.Attempts)-1].Result = &result
			m.Attempts[len(m.Attempts)-1].ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
			m.LastOutcome = result.Outcome
			m.RetryDueAt = ""
			n.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			n.State, n.LastError = NotificationFailed, result.ErrorCode
			if result.Outcome == "attempted" { n.State = "attempted" }
			if result.Outcome == "unknown" { n.State = "unknown" }
			if !result.Attempted {
				n.DispatchAttempts--
				// A pre-send failure releases its reservation, never prior usage.
				// Only absent configuration is waiting; permission denial is terminal.
				if result.Outcome == "configuration_failure" && n.DispatchAttempts == 0 { n.State = "waiting" }
			}
			if result.Outcome == "network_failure" && n.DispatchAttempts == 1 {
				m.RetryDueAt = time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano)
				m.Attempts[len(m.Attempts)-1].RetryDueAt = m.RetryDueAt
			}
			return nil
		}
		return errors.New("unknown notification")
	})
}

// Old receipt callers cannot turn an acknowledgement into delivery evidence.
func (s *Store) CompleteNotificationDispatch(id TaskID, notificationID NotificationID, receipt, failure string) (RuntimeState, error) {
	state, err := s.Load(id)
	if err != nil { return state, err }
	for _, n := range state.Notifications { if n.ID == notificationID { return s.ObserveNotification(id, notificationID, UnknownNotificationResult(n)) } }
	return state, errors.New("unknown notification")
}

func (s *Store) RecoverNotificationDispatch(id TaskID, notificationID NotificationID) (RuntimeState, error) {
	return s.CompleteNotificationDispatch(id, notificationID, "", "")
}

func (s *Store) DispatchNotification(ctx context.Context, id TaskID, notificationID NotificationID, dispatcher NotificationDispatcher) (RuntimeState, error) {
	managed, ok := dispatcher.(ManagedNotificationDispatcher)
	if !ok { return RuntimeState{}, errors.New("managed notification adapter required") }
	return s.SendNotification(ctx, id, notificationID, managed)
}

func (s *Store) SendNotification(ctx context.Context, id TaskID, notificationID NotificationID, dispatcher ManagedNotificationDispatcher) (RuntimeState, error) {
	state, err := s.Load(id)
	if err != nil { return state, err }
	var n Notification
	for _, candidate := range state.Notifications { if candidate.ID == notificationID { n = candidate; break } }
	if n.Managed == nil { return state, errors.New("legacy notification cannot be replayed") }
	config, preflightErr := dispatcher.Preflight(ctx, n)
	state, err = s.notificationUpdate(id, func(current *RuntimeState) error {
		for i := range current.Notifications {
			item := &current.Notifications[i]
			if item.ID != notificationID { continue }
			if item.State == NotificationDispatching || item.State == "attempted" || item.State == "unknown" || item.State == "cancelled" { return errors.New("notification already settled or in flight") }
			if preflightErr != nil { item.LastError = "configuration"; return nil }
			if !notificationActive(*current, *item) || (!config.Enabled && item.Managed.GateID == "") { item.State = "cancelled"; return nil }
			if !config.Enabled || !config.Ready { item.State = "waiting"; item.LastError = "configuration"; return nil }
			fixed := item.Managed.Config
			if fixed.ConfigReference != "" && (fixed.ConfigReference != config.ConfigReference || fixed.TargetReference != config.TargetReference || fixed.Channel != config.Channel) { item.State = "waiting"; item.LastError = "configuration"; return nil }
			item.Managed.Config = config
			item.LastError = ""
			return nil
		}
		return errors.New("unknown notification")
	})
	if err != nil || preflightErr != nil { return state, errors.Join(err, preflightErr) }
	for _, item := range state.Notifications { if item.ID == notificationID && (item.State == "cancelled" || item.LastError != "") { return state, nil } }
	if err := ctx.Err(); err != nil { return state, err }
	_, n, err = s.PrepareNotificationDispatch(id, notificationID)
	if err != nil { return state, err }
	// Recheck Stop, validity and the exact reservation immediately before launch.
	current, err := s.Load(id)
	if err != nil { return state, err }
	if (current.Protocol != nil && current.Protocol.Stop != nil) || !notificationActive(current, n) {
		r := UnknownNotificationResult(n); r.Attempted, r.Outcome, r.ErrorCode = false, "permission_failure", "permission"
		return s.ObserveNotification(id, notificationID, r)
	}
	return s.ObserveNotification(id, notificationID, dispatcher.Send(ctx, n))
}

func validateNotificationRequest(n Notification) error {
	if n.ID == "" || strings.TrimSpace(n.Topic) == "" || !json.Valid(n.Payload) { return errors.New("notification id, topic and JSON payload required") }
	if n.Managed != nil {
		m := n.Managed
		if m.Version != 2 || m.FactID == "" || m.Text == "" || !utf8.ValidString(m.Text) || len(m.Text) > 16*1024 || m.ContentDigest != contentDigest([]byte(m.Text)) { return errors.New("invalid managed notification content") }
	}
	return nil
}

func jsonEqual(left, right json.RawMessage) bool {
	var a, b any
	return json.Unmarshal(left, &a) == nil && json.Unmarshal(right, &b) == nil && reflect.DeepEqual(a, b)
}
