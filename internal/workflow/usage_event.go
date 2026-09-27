package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// RecordUsageEvent commits one immutable Provider result through the Task's
// ordered event path. Replays for the same frozen turn are idempotent.
func (s *Store) RecordUsageEvent(id TaskID, event UsageEvent) (RuntimeState, error) {
	if event.TaskID != id || event.WorkItemID == "" || event.AttemptID == "" || event.SessionID == "" || event.SessionTurn <= 0 || event.Provider == "" || event.Model == "" || event.Outcome == "" || !json.Valid(event.Usage) {
		return RuntimeState{}, errors.New("usage event requires a complete managed-call identity and Provider evidence")
	}
	identity, err := json.Marshal([]string{string(event.TaskID), string(event.WorkItemID), string(event.AttemptID), event.SessionID, fmt.Sprint(event.SessionTurn)})
	if err != nil { return RuntimeState{}, err }
	digest := sha256.Sum256(identity)
	event.Version = TaskUsageLedgerVersion
	event.ID = hex.EncodeToString(digest[:])
	if event.RecordedAt == "" { event.RecordedAt = time.Now().UTC().Format(time.RFC3339Nano) }
	encoded, err := json.Marshal(event)
	if err != nil { return RuntimeState{}, err }

	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, err }
	if state.SchemaVersion != DurableSchemaVersion || state.Protocol == nil {
		return RuntimeState{}, errors.New("usage accounting requires the migrated Schema 10 protocol")
	}
	if existing, found, err := matchingUsageEvent(state.Protocol.Usage, event.ID); err != nil {
		return RuntimeState{}, err
	} else if found {
		if !sameUsageEvent(existing, encoded) { return RuntimeState{}, errors.New("usage event identity is already bound to different Provider evidence") }
		return state, nil
	}
	revision := state.StateRevision
	updated, err := s.updateWithEvent(id, &revision, Event{Type: "protocol.usage.recorded", WorkItemID: event.WorkItemID, AttemptID: event.AttemptID, Detail: event.ID}, func(current *RuntimeState) error {
		if current.Protocol == nil { return errors.New("Schema 10 execution protocol is missing") }
		ledger := current.Protocol.Usage
		if ledger != nil {
			if existing, found, err := matchingUsageEvent(ledger, event.ID); err != nil { return err } else if found {
				if !sameUsageEvent(existing, encoded) { return errors.New("usage event identity is already bound to different Provider evidence") }
				return errProtocolNoChange
			}
		} else {
			ledger = &TaskUsageLedger{Version: TaskUsageLedgerVersion, Records: []json.RawMessage{}, Projection: TaskUsageProjection{KnownCostByCurrency: map[string]string{}}}
			current.Protocol.Usage = ledger
		}
		ledger.Records = append(ledger.Records, append(json.RawMessage(nil), encoded...))
		if err := addUsageProjection(&ledger.Projection, event.Usage); err != nil {
			ledger.Records = ledger.Records[:len(ledger.Records)-1]
			return err
		}
		return refreshUsageBudgetGate(current, ledger, time.Now().UTC())
	})
	if err == nil { return updated, nil }
	// A concurrent replay can lose the revision race after another writer has
	// committed this same idempotency key. Confirm its durable result once.
	latest, loadErr := s.Load(id)
	if loadErr == nil && latest.Protocol != nil {
		if existing, found, matchErr := matchingUsageEvent(latest.Protocol.Usage, event.ID); matchErr == nil && found && sameUsageEvent(existing, encoded) { return latest, nil }
	}
	return RuntimeState{}, err
}

func sameUsageEvent(left, right json.RawMessage) bool {
	var a, b UsageEvent
	if json.Unmarshal(left, &a) != nil || json.Unmarshal(right, &b) != nil { return false }
	a.RecordedAt, b.RecordedAt = "", ""
	encodedA, errA := json.Marshal(a)
	encodedB, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(encodedA) == string(encodedB)
}

func matchingUsageEvent(ledger *TaskUsageLedger, id string) (json.RawMessage, bool, error) {
	if ledger == nil { return nil, false, nil }
	for _, raw := range ledger.Records {
		var existing UsageEvent
		if err := json.Unmarshal(raw, &existing); err != nil { return nil, false, fmt.Errorf("decode existing usage event: %w", err) }
		if existing.ID == id { return raw, true, nil }
	}
	return nil, false, nil
}
