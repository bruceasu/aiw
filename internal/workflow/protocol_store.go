package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// persistProtocolArtifactLocked is used only inside a Task lock. An orphaned
// content-addressed artifact is safe to replay; a reference is committed later.
func (s *Store) persistProtocolArtifactLocked(id TaskID, kind string, value any) (ActorReference, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil { return ActorReference{}, err }
	data = append(data, '\n')
	digest := contentDigest(data)
	ref := ActorReference{Kind: kind, Path: "reports/protocol/"+digest+".json", SHA256: digest}
	path := s.path(id, filepath.FromSlash(ref.Path))
	previous, err := os.ReadFile(path)
	if err == nil && !bytes.Equal(previous, data) { return ActorReference{}, errors.New("immutable protocol artifact conflict") }
	if err != nil && !errors.Is(err, os.ErrNotExist) { return ActorReference{}, err }
	// Re-publishing identical bytes also repairs an unconfirmed final flush.
	if err := durableWrite(path, data); err != nil { return ActorReference{}, err }
	return ref, nil
}

// MigrateDurableExecution never starts execution. All service providers and
// validation references must be present before schema 10 is written. The raw
// legacy snapshot is retained; unknown fields cannot silently disappear.
func (s *Store) MigrateDurableExecution(id TaskID, activation []ActorReference) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, fmt.Errorf("migration: require execution services: %w", err) }
	lock, err := s.lock(id)
	if err != nil { return RuntimeState{}, fmt.Errorf("migration: acquire Task lock: %w", err) }
	defer unlock(lock)
	if !isSystemLock(lock) { return RuntimeState{}, errors.New("migrate Task lock under verified maintenance before schema migration") }
	raw, err := os.ReadFile(s.path(id, runtimeStateFile))
	if err != nil { return RuntimeState{}, fmt.Errorf("migration: read source state: %w", err) }
	var original RuntimeState
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&original); err != nil { return RuntimeState{}, fmt.Errorf("preserve unsupported migration source: %w", err) }
	if original.SchemaVersion == DurableSchemaVersion {
		state, err := s.Load(id)
		if err != nil { return state, fmt.Errorf("migration: load already migrated state: %w", err) }
		return state, nil
	}
	if original.SchemaVersion < 1 || original.SchemaVersion > SchemaVersion || original.Task.ID != id { return RuntimeState{}, errors.New("unsupported migration source") }
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, fmt.Errorf("migration: load legacy state: %w", err) }
	if state.PendingEvent != nil || state.WriteLease != nil || activeAttempt(state.Attempts) != "" || state.Automation.PreparedRequest != nil || state.Automation.Supervisor.LeaseID != "" { return RuntimeState{}, errors.New("reconcile legacy writers and pending commits before migration") }
	if len(activation) == 0 { return RuntimeState{}, errors.New("activation evidence is required") }
	for _, ref := range activation {
		var evidence json.RawMessage
		if !validProtocolReference(ref) { return RuntimeState{}, errors.New("invalid activation evidence") }
		if err := s.ReadExecutionArtifact(id, ref, &evidence); err != nil { return RuntimeState{}, fmt.Errorf("migration: read activation artifact %q: %w", ref.Path, err) }
	}
	if err := s.ExecutionServices.VerifyActivation(state, activation); err != nil { return RuntimeState{}, fmt.Errorf("migration: verify activation: %w", err) }
	// The source envelope preserves original bytes, not a normalized projection.
	ref, err := s.persistProtocolArtifactLocked(id, "migration-source", struct {
		Schema int `json:"source_schema"`
		Digest string `json:"source_digest"`
		Content string `json:"source_content"`
	}{original.SchemaVersion, contentDigest(raw), string(raw)})
	if err != nil { return RuntimeState{}, fmt.Errorf("migration: persist source artifact: %w", err) }
	state.SchemaVersion = DurableSchemaVersion
	state.StateRevision = 1
	state.CommitID = fmt.Sprintf("%s-%d", id, state.StateRevision)
	state.Protocol = &ExecutionProtocol{Version: 1, Migration: ref, Activation: activation, Budget: json.RawMessage(`null`)}
	if original.Automation.Supervisor.StoppedAt != "" {
		state.Protocol.Stop = &ExecutionStop{Reason: "preserved legacy Stop; explicit reconciliation is required", Reference: ref}
	}
	// E04 must retain unknown balances instead of manufacturing a fresh budget.
	if err := s.ExecutionServices.MigrateBudget(&state); err != nil { return RuntimeState{}, fmt.Errorf("migration: migrate budget: %w", err) }
	event := Event{SchemaVersion: DurableSchemaVersion, Sequence: state.LastEventSequence+1, Type: "protocol.migrated", At: time.Now().UTC().Format(time.RFC3339), CommitID: state.CommitID, StateRevision: state.StateRevision}
	state.PendingEvent = &event
	if err := ValidateRuntimeState(state); err != nil { return RuntimeState{}, fmt.Errorf("migration: validate candidate state: %w", err) }
	if err := s.save(state); err != nil { return RuntimeState{}, fmt.Errorf("migration: save pending state: %w", err) }
	if err := s.appendEvent(event, id); err != nil { return RuntimeState{}, fmt.Errorf("migration: append committed event: %w", err) }
	state.PendingEvent, state.LastEventSequence = nil, event.Sequence
	if err := s.save(state); err != nil { return RuntimeState{}, fmt.Errorf("migration: save confirmed state: %w", err) }
	state, err = s.Load(id)
	if err != nil { return state, fmt.Errorf("migration: load confirmed state: %w", err) }
	return state, nil
}

// PersistPilotActivationEvidence publishes immutable evidence needed by the
// following migration transaction. It is limited to a verified system lock
// and a legacy Schema 9 state with no active writer; it never changes Task
// state or advances the event sequence.
func (s *Store) PersistPilotActivationEvidence(id TaskID, evidence any) (ActorReference, error) {
	lock, err := s.lock(id)
	if err != nil { return ActorReference{}, err }
	defer unlock(lock)
	if !isSystemLock(lock) { return ActorReference{}, errors.New("pilot activation evidence requires the verified durable Task lock") }
	state, err := s.Load(id)
	if err != nil { return ActorReference{}, err }
	if state.SchemaVersion != SchemaVersion || state.PendingEvent != nil || state.WriteLease != nil || activeAttempt(state.Attempts) != "" || state.Automation.PreparedRequest != nil || state.Automation.Supervisor.LeaseID != "" {
		return ActorReference{}, errors.New("pilot activation evidence requires an idle Schema 9 Task")
	}
	return s.persistProtocolArtifactLocked(id, "pilot-activation", evidence)
}

// repairPendingTail accepts only a final byte-prefix of the unique pending
// event after a complete, contiguous confirmed history. Original bytes remain
// available as evidence. No event or caller action is generated by recovery.
func (s *Store) repairPendingTail(id TaskID, state RuntimeState) error {
	if state.SchemaVersion != DurableSchemaVersion || state.PendingEvent == nil { return errors.New("event log requires manual reconciliation") }
	path := s.path(id, runtimeEventsFile)
	raw, err := os.ReadFile(path)
	if err != nil { return err }
	end := bytes.LastIndexByte(raw, '\n')+1
	prefix, tail := raw[:end], raw[end:]
	expected, err := json.Marshal(state.PendingEvent)
	if err != nil { return err }
	expected = append(expected, '\n')
	if len(tail) == 0 || !bytes.HasPrefix(expected, tail) { return errors.New("event log corruption is not the unique pending tail") }
	var sequence uint64
	for _, line := range bytes.Split(prefix, []byte{'\n'}) {
		if len(line) == 0 { continue }
		var event Event
		if err := json.Unmarshal(line, &event); err != nil { return err }
		sequence++
		if event.Sequence != sequence { return errors.New("event prefix has a gap or conflicting sequence") }
	}
	if sequence != state.LastEventSequence { return errors.New("event prefix does not match confirmed history") }
	if _, err := s.persistProtocolArtifactLocked(id, "event-tail-recovery", struct { Original []byte `json:"original"` }{raw}); err != nil { return err }
	return durableWrite(path, append(append([]byte(nil), prefix...), expected...))
}
