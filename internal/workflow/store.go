package workflow

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/repo"
)

const (
	defaultRuntimeRoot = ".ai"
	runtimeTasksDir    = "tasks"
	runtimeLocksDir    = "locks"
	runtimeStateFile   = "state.json"
	runtimeEventsFile  = "events.jsonl"
	legacyMigrationFile = "migrated-to"
	legacyMigrationPendingFile = ".migration-pending"
)

// Store persists local, high-churn Workflow Core state. It deliberately does
// not read or write OpenSpec artifacts; Task/OpenSpec adapters own that seam.
type Store struct {
	Root string
}

type Event struct {
	SchemaVersion int        `json:"schema_version"`
	Sequence      uint64     `json:"sequence"`
	Type          string     `json:"type"`
	At            string     `json:"at"`
	WorkItemID    WorkItemID `json:"work_item_id,omitempty"`
	AttemptID     AttemptID  `json:"attempt_id,omitempty"`
	Detail        string     `json:"detail,omitempty"`
}

func NewStore(root string) *Store {
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(repo.Root(), defaultRuntimeRoot)
	}
	return &Store{Root: root}
}

func (s *Store) Create(state RuntimeState) (RuntimeState, error) {
	if err := validateTaskID(state.Task.ID); err != nil {
		return RuntimeState{}, err
	}
	if err := normalizeRuntimeState(&state); err != nil {
		return RuntimeState{}, err
	}
	if err := ValidateRuntimeState(state); err != nil {
		return RuntimeState{}, err
	}
	statePath := s.path(state.Task.ID, runtimeStateFile)
	if _, err := os.Stat(statePath); err == nil {
		return RuntimeState{}, fmt.Errorf("workflow Task already exists: %s", state.Task.ID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return RuntimeState{}, err
	}
	if err := os.MkdirAll(s.taskDir(state.Task.ID), 0o755); err != nil {
		return RuntimeState{}, err
	}
	if err := s.save(state); err != nil {
		return RuntimeState{}, err
	}
	if err := s.ensureEventLog(state.Task.ID); err != nil {
		return RuntimeState{}, err
	}
	return s.Load(state.Task.ID)
}

func (s *Store) Load(id TaskID) (RuntimeState, error) {
	if err := validateTaskID(id); err != nil {
		return RuntimeState{}, err
	}
	dir := s.taskDir(id)
	if _, err := os.Stat(filepath.Join(dir, runtimeStateFile)); errors.Is(err, os.ErrNotExist) {
		dir = s.legacyTaskDir(id)
	} else if err != nil {
		return RuntimeState{}, err
	}
	return s.loadFromDir(id, dir)
}

func (s *Store) loadFromDir(id TaskID, dir string) (RuntimeState, error) {
	b, err := os.ReadFile(filepath.Join(dir, runtimeStateFile))
	if err != nil {
		return RuntimeState{}, err
	}
	var state RuntimeState
	if err := json.Unmarshal(b, &state); err != nil {
		return RuntimeState{}, fmt.Errorf("decode workflow state: %w", err)
	}
	if state.Task.ID != id {
		return RuntimeState{}, fmt.Errorf("workflow state id mismatch: %s", state.Task.ID)
	}
	if err := normalizeRuntimeState(&state); err != nil {
		return RuntimeState{}, err
	}
	if err := ValidateRuntimeState(state); err != nil {
		return RuntimeState{}, err
	}
	return state, nil
}

func normalizeRuntimeState(state *RuntimeState) error {
	switch state.SchemaVersion {
	case 1, 2, 3, 4, 5, 6, 7, 8:
		state.SchemaVersion = SchemaVersion
	case SchemaVersion:
	default:
		return fmt.Errorf("unsupported workflow schema version: %d", state.SchemaVersion)
	}
	for index := range state.WorkItems {
		item := &state.WorkItems[index]
		if item.RetryPolicy.MaxAttempts == 0 {
			item.RetryPolicy.MaxAttempts = DefaultRetryLimit
		}
	}
	if err := normalizeOperationalRetries(&state.OperationalRetries); err != nil {
		return err
	}
	return nil
}

// update is reserved for bootstrap, migration, and package-local recovery
// setup. Managed runtime transitions must use UpdateWithEvent.
func (s *Store) update(id TaskID, change func(*RuntimeState) error) (RuntimeState, error) {
	lock, err := s.lock(id)
	if err != nil {
		return RuntimeState{}, err
	}
	defer unlock(lock)
	if err := s.migrateLegacyLocked(id); err != nil {
		return RuntimeState{}, err
	}

	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if err := change(&state); err != nil {
		return RuntimeState{}, err
	}
	if state.Task.ID != id {
		return RuntimeState{}, errors.New("workflow Task ID cannot change")
	}
	if err := ValidateRuntimeState(state); err != nil {
		return RuntimeState{}, err
	}
	if err := s.save(state); err != nil {
		return RuntimeState{}, err
	}
	return s.Load(id)
}

// UpdateWithEvent makes an in-process state transition recoverable. It first
// records the fully formed pending event in the atomically written state,
// appends that exact event, then records the confirmed sequence. Recovery can
// therefore replay only a persisted event and never repeat a caller action.
func (s *Store) UpdateWithEvent(id TaskID, event Event, change func(*RuntimeState) error) (RuntimeState, error) {
	lock, err := s.lock(id)
	if err != nil {
		return RuntimeState{}, err
	}
	defer unlock(lock)
	if err := s.migrateLegacyLocked(id); err != nil {
		return RuntimeState{}, err
	}

	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if state.PendingEvent != nil {
		return RuntimeState{}, fmt.Errorf("workflow Task has pending event %d; recover it before another transition", state.PendingEvent.Sequence)
	}
	if err := change(&state); err != nil {
		return RuntimeState{}, err
	}
	if strings.TrimSpace(event.Type) == "" {
		return RuntimeState{}, errors.New("workflow event type is required")
	}
	event.SchemaVersion = SchemaVersion
	event.Sequence = state.LastEventSequence + 1
	if event.At == "" {
		event.At = time.Now().UTC().Format(time.RFC3339)
	}
	state.PendingEvent = &event
	if err := ValidateRuntimeState(state); err != nil {
		return RuntimeState{}, err
	}
	if err := s.save(state); err != nil {
		return RuntimeState{}, err
	}
	if err := s.appendEvent(event, id); err != nil {
		return RuntimeState{}, err
	}
	state.LastEventSequence = event.Sequence
	state.PendingEvent = nil
	if err := s.save(state); err != nil {
		return RuntimeState{}, err
	}
	return s.Load(id)
}

func (s *Store) appendEvent(event Event, id TaskID) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.path(id, runtimeEventsFile), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// ensureEventLog completes runtime bootstrap after an interrupted initial
// state write without replacing existing event history.
func (s *Store) ensureEventLog(id TaskID) error {
	path := s.path(id, runtimeEventsFile)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return atomicWrite(path, nil)
}

// RecoverPendingEvent appends only the exact event already persisted in state.
// It never reruns the transition or any caller-owned external operation.
func (s *Store) RecoverPendingEvent(id TaskID) (RuntimeState, error) {
	lock, err := s.lock(id)
	if err != nil {
		return RuntimeState{}, err
	}
	defer unlock(lock)
	if err := s.migrateLegacyLocked(id); err != nil {
		return RuntimeState{}, err
	}
	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if state.PendingEvent == nil {
		return state, nil
	}
	pending := *state.PendingEvent
	if pending.Sequence != state.LastEventSequence+1 || pending.Sequence == 0 {
		return RuntimeState{}, fmt.Errorf("pending event sequence %d cannot follow confirmed sequence %d", pending.Sequence, state.LastEventSequence)
	}
	events, err := s.readEvents(id)
	if err != nil {
		return RuntimeState{}, err
	}
	found := false
	for _, existing := range events {
		if existing.Sequence != pending.Sequence {
			continue
		}
		if existing != pending {
			return RuntimeState{}, fmt.Errorf("event sequence %d conflicts with pending event", pending.Sequence)
		}
		found = true
	}
	if !found {
		if err := s.appendEvent(pending, id); err != nil {
			return RuntimeState{}, err
		}
	}
	state.LastEventSequence = pending.Sequence
	state.PendingEvent = nil
	if err := s.save(state); err != nil {
		return RuntimeState{}, err
	}
	return s.Load(id)
}

func (s *Store) readEvents(id TaskID) ([]Event, error) {
	f, err := os.Open(s.path(id, runtimeEventsFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("decode workflow event: %w", err)
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}

// AcquireWriteLease grants the only durable write lease for a Task workspace.
// It deliberately does not infer an Attempt: callers must name the Attempt
// whose lifecycle owns the requested write access.
func (s *Store) AcquireWriteLease(id TaskID, attemptID AttemptID) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "write-lease.acquired", AttemptID: attemptID}, func(state *RuntimeState) error {
		attempt, ok := findAttempt(state.Attempts, attemptID)
		if !ok {
			return fmt.Errorf("write lease references unknown attempt %s", attemptID)
		}
		if state.WriteLease != nil && state.WriteLease.AttemptID != attemptID {
			return fmt.Errorf("workspace write lease is held by attempt %s", state.WriteLease.AttemptID)
		}
		state.WriteLease = &WriteLease{
			AttemptID:  attemptID,
			Workspace:  attempt.Workspace,
			AcquiredAt: time.Now().UTC().Format(time.RFC3339),
		}
		return nil
	})
}

// ReleaseWriteLease releases a lease only for its owning Attempt. This avoids
// one recovered or stale Attempt releasing another writer's workspace.
func (s *Store) ReleaseWriteLease(id TaskID, attemptID AttemptID) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "write-lease.released", AttemptID: attemptID}, func(state *RuntimeState) error {
		if state.WriteLease == nil {
			return nil
		}
		if state.WriteLease.AttemptID != attemptID {
			return fmt.Errorf("workspace write lease is held by attempt %s", state.WriteLease.AttemptID)
		}
		state.WriteLease = nil
		return nil
	})
}

func (s *Store) taskDir(id TaskID) string {
	return filepath.Join(s.Root, string(id))
}

func (s *Store) legacyTaskDir(id TaskID) string {
	return filepath.Join(s.Root, runtimeTasksDir, string(id))
}

func (s *Store) path(id TaskID, names ...string) string {
	return filepath.Join(append([]string{s.taskDir(id)}, names...)...)
}

func (s *Store) lock(id TaskID) (*os.File, error) {
	if err := validateTaskID(id); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(s.Root, runtimeLocksDir), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(s.Root, runtimeLocksDir, string(id)+".lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
}

func (s *Store) save(state RuntimeState) error {
	state.Summary = DeriveSummary(state)
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(s.path(state.Task.ID, runtimeStateFile), append(b, '\n'))
}

// migrateLegacyLocked promotes the complete legacy Task aggregate only while
// the Task's exclusive state lock is held. A rename keeps state, events,
// handoffs, and adapter-owned payloads together. The migration event follows
// the same pending-event protocol as ordinary transitions, so an interruption
// can be recovered without replaying a caller action.
func (s *Store) migrateLegacyLocked(id TaskID) error {
	canonical := s.taskDir(id)
	legacy := s.legacyTaskDir(id)
	if _, err := os.Stat(filepath.Join(canonical, runtimeStateFile)); err == nil {
		if _, err := s.loadFromDir(id, canonical); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(canonical, legacyMigrationPendingFile)); err == nil {
			if err := s.recoverMigrationEvent(id); err != nil {
				return err
			}
			if err := s.recordMigrationEvent(id); err != nil {
				return err
			}
			if err := os.Remove(filepath.Join(canonical, legacyMigrationPendingFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return s.markLegacyMigrated(legacy, canonical)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(filepath.Join(legacy, runtimeStateFile)); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(legacy, legacyMigrationPendingFile), []byte("pending\n")); err != nil {
		return err
	}
	if err := os.Rename(legacy, canonical); err != nil {
		return fmt.Errorf("migrate legacy workflow Task %s: %w", id, err)
	}
	if err := s.recordMigrationEvent(id); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(canonical, legacyMigrationPendingFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return s.markLegacyMigrated(legacy, canonical)
}

func (s *Store) recordMigrationEvent(id TaskID) error {
	state, err := s.loadFromDir(id, s.taskDir(id))
	if err != nil {
		return err
	}
	if state.PendingEvent != nil {
		return fmt.Errorf("legacy workflow Task %s has pending event %d; recover it before migration", id, state.PendingEvent.Sequence)
	}
	events, err := s.readEvents(id)
	if err != nil {
		return err
	}
	for _, existing := range events {
		if existing.Type == "runtime.migrated" && existing.Detail == "from .ai/tasks" {
			return nil
		}
	}
	event := Event{SchemaVersion: SchemaVersion, Sequence: state.LastEventSequence + 1, Type: "runtime.migrated", At: time.Now().UTC().Format(time.RFC3339), Detail: "from .ai/tasks"}
	state.PendingEvent = &event
	if err := s.save(state); err != nil {
		return err
	}
	if err := s.appendEvent(event, id); err != nil {
		return err
	}
	state.LastEventSequence = event.Sequence
	state.PendingEvent = nil
	return s.save(state)
}

func (s *Store) recoverMigrationEvent(id TaskID) error {
	state, err := s.loadFromDir(id, s.taskDir(id))
	if err != nil {
		return err
	}
	if state.PendingEvent == nil {
		return nil
	}
	pending := *state.PendingEvent
	if pending.Type != "runtime.migrated" || pending.Sequence != state.LastEventSequence+1 {
		return fmt.Errorf("legacy workflow Task %s has unrelated pending event; recover it before migration", id)
	}
	events, err := s.readEvents(id)
	if err != nil {
		return err
	}
	found := false
	for _, existing := range events {
		if existing.Sequence == pending.Sequence {
			if existing != pending {
				return fmt.Errorf("migration event sequence %d conflicts with event history", pending.Sequence)
			}
			found = true
		}
	}
	if !found {
		if err := s.appendEvent(pending, id); err != nil {
			return err
		}
	}
	state.LastEventSequence = pending.Sequence
	state.PendingEvent = nil
	return s.save(state)
}

func (s *Store) markLegacyMigrated(legacy, canonical string) error {
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		return err
	}
	return atomicWrite(filepath.Join(legacy, legacyMigrationFile), []byte(canonical+"\n"))
}

func validateTaskID(id TaskID) error {
	if strings.TrimSpace(string(id)) == "" || strings.ContainsAny(string(id), `/\\`) {
		return errors.New("invalid workflow Task ID")
	}
	return nil
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".aiw-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	var renameErr error
	for attempt := 0; attempt < 5; attempt++ {
		if err := os.Rename(name, path); err == nil {
			return nil
		} else {
			renameErr = err
		}
		// Windows can briefly retain a handle to the previous state file after
		// a concurrent status reader or antivirus scan. Retrying preserves the
		// atomic replace protocol without weakening pending-event recovery.
		time.Sleep(time.Duration(attempt+1) * 25 * time.Millisecond)
	}
	return renameErr
}

func unlock(file *os.File) {
	if file == nil {
		return
	}
	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
}

func findAttempt(attempts []Attempt, id AttemptID) (Attempt, bool) {
	for _, attempt := range attempts {
		if attempt.ID == id {
			return attempt, true
		}
	}
	return Attempt{}, false
}
