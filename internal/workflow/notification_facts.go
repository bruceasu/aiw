package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"
)

// Register facts in the same commit as completion/Gates. No provider, model,
// config parser, or external process is involved in the completion transaction.
func (s *Store) captureNotificationFacts(state *RuntimeState) {
	if state.Protocol == nil { return }
	// Freeze the config source before asynchronous parsing. No credentials or
	// raw config enter state; changes cannot retarget an already committed fact.
	snapshot := "unavailable"
	file, err := os.Open(filepath.Join(filepath.Dir(s.Root), "aiw.toml"))
	if os.IsNotExist(err) { snapshot = "absent" }
	if err == nil {
		body, readErr := io.ReadAll(io.LimitReader(file, 64*1024+1))
		_ = file.Close()
		if readErr == nil && len(body) <= 64*1024 { snapshot = contentDigest(body) }
	}
	add := func(fact, topic, text string, gate GateID, related NotificationID) NotificationID {
		id := NotificationID("notice-"+contentDigest([]byte(fact)))
		for _, n := range state.Notifications { if n.ID == id { return id } }
		if len(text) > 16*1024 || !utf8.ValidString(text) { text = "Workflow event: "+topic+". Review the local Task record for the exact state, coverage and required action. Delivery and review conclusions remain separate." }
		now := time.Now().UTC().Format(time.RFC3339)
		payload, _ := json.Marshal(map[string]string{"fact": fact})
		status := NotificationPending
		if snapshot == "absent" && gate == "" { status = "cancelled" }
		state.Notifications = append(state.Notifications, Notification{ID: id, Topic: topic, Payload: payload, State: status, CreatedAt: now, UpdatedAt: now, Managed: &NotificationMessage{Version: 2, FactID: fact, Text: text, ContentDigest: contentDigest([]byte(text)), GateID: gate, Related: related, ConfigSnapshot: snapshot}})
		return id
	}
	var completion NotificationID
	if a := state.Protocol.Auxiliary; a != nil {
		for _, source := range a.Sources {
			if source.Kind != "completed" { continue }
			completion = add(string(state.Task.ID)+":completed", "workflow.completed", fmt.Sprintf("Task %s: development completed.\nDelivery: %s.\nKnowledge: pending or partial.\nVerifier: pending or unavailable.\nReview the local Task artifacts for follow-up.", state.Task.ID, state.Delivery), "", "")
			break
		}
		if completion != "" {
			if state.Delivery != "pending" && state.Delivery != "unmanaged" && state.Delivery != "" {
				add(string(state.Task.ID)+":delivery:"+string(state.Delivery), "workflow.update", fmt.Sprintf("Task %s: development remains completed.\nDelivery: %s.\nKnowledge and Verifier conclusions remain separate; review local records.\nRelated notification: %s.", state.Task.ID, state.Delivery, completion), "", completion)
			}
			for _, job := range a.Jobs {
				if job.State != "completed" || job.Output == nil || (job.Kind != "knowledge-summary" && job.Kind != "verifier") { continue }
				add(string(state.Task.ID)+":"+job.Key+":"+job.Output.SHA256, "workflow.update", fmt.Sprintf("Task %s: development remains completed.\nDelivery: %s.\n%s has a new local report; coverage and conclusions require review.\nRelated notification: %s.", state.Task.ID, state.Delivery, job.Kind, completion), "", completion)
			}
		}
	}
	for _, gate := range state.Gates {
		if gate.State != GateOpen || gate.Kind == GateDependency { continue }
		// Keep arbitrary diagnostic bodies and paths out of externally sent text.
		add(string(state.Task.ID)+":gate:"+string(gate.ID), "workflow.action", fmt.Sprintf("Task %s needs attention.\nOpen Gate: %s (%s).\nReview its reason and required action in the local Task record.", state.Task.ID, gate.ID, gate.Kind), gate.ID, completion)
	}
	for i := range state.Notifications {
		n := &state.Notifications[i]
		if n.Managed != nil && !notificationActive(*state, *n) && (n.State == NotificationPending || n.State == "waiting" || n.State == NotificationFailed) { n.State = "cancelled" }
	}
}

func validateManagedNotification(n Notification) error {
	if err := validateNotificationRequest(n); err != nil { return err }
	m := n.Managed
	if n.DispatchAttempts < 0 || n.DispatchAttempts > 2 { return fmt.Errorf("notification attempt limit violated") }
	used := 0
	for i, a := range m.Attempts {
		if a.ID != fmt.Sprintf("%s:%d", n.ID, i+1) { return fmt.Errorf("notification attempt identity mismatch") }
		if _, err := time.Parse(time.RFC3339, a.ReservedAt); err != nil { return err }
		if a.Result == nil || a.Result.Attempted { used++ }
		if a.Result == nil && (i != len(m.Attempts)-1 || n.State != NotificationDispatching) { return fmt.Errorf("notification has an unresolved reservation") }
	}
	if used != n.DispatchAttempts { return fmt.Errorf("notification attempt ledger mismatch") }
	if m.RetryDueAt != "" {
		if _, err := time.Parse(time.RFC3339, m.RetryDueAt); err != nil { return err }
		if n.DispatchAttempts != 1 || m.LastOutcome != "network_failure" { return fmt.Errorf("invalid notification retry") }
	}
	switch n.State { case NotificationPending, NotificationDispatching, NotificationFailed, "waiting", "cancelled", "attempted", "unknown": return nil }
	return fmt.Errorf("unsupported managed notification state")
}
