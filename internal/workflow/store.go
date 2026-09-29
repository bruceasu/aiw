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
	"aiw/internal/taskpath"
)

const (
	defaultRuntimeRoot = ".ai"
	runtimeTasksDir    = "tasks"
	runtimeLocksDir    = "locks"
	runtimeStateFile   = "state.json"
	runtimeEventsFile  = "events.jsonl"
	legacyMigrationFile = "migrated-to"
)

// Store persists local, high-churn Workflow Core state. It deliberately does
// not read or write OpenSpec artifacts; Task/OpenSpec adapters own that seam.
type Store struct {
	Root string
}

type Event struct {
	CommitID      string     `json:"commit_id,omitempty"`
	StateRevision uint64     `json:"state_revision,omitempty"`
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
	if err := s.legacyTaskError(state.Task.ID); err != nil { return RuntimeState{}, err }
	lock, err := s.lock(state.Task.ID)
	if err != nil { return RuntimeState{}, err }
	defer unlock(lock)
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
	if err := s.legacyTaskError(id); err != nil { return RuntimeState{}, err }
	dir := s.taskDir(id)
	if _, err := os.Stat(filepath.Join(dir, runtimeStateFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
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
	if err := s.rejectLegacyTask(id); err != nil {
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
	return s.updateWithEvent(id, nil, event, change)
}

func (s *Store) updateWithEvent(id TaskID, expected *uint64, event Event, change func(*RuntimeState) error) (RuntimeState, error) {
	lock, err := s.lock(id)
	if err != nil {
		return RuntimeState{}, err
	}
	defer unlock(lock)
	if err := s.rejectLegacyTask(id); err != nil {
		return RuntimeState{}, err
	}

	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if state.PendingEvent != nil {
		return RuntimeState{}, fmt.Errorf("workflow Task has pending event %d; recover it before another transition", state.PendingEvent.Sequence)
	}
	if expected != nil && state.StateRevision != *expected { return RuntimeState{}, errors.New("workflow state revision changed; reconcile before retry") }
	originalSchema, originalRevision := state.SchemaVersion, state.StateRevision
	if err := change(&state); err != nil {
		return RuntimeState{}, err
	}
	if state.SchemaVersion != originalSchema || state.StateRevision != originalRevision { return RuntimeState{}, errors.New("transition cannot replace the schema or commit revision") }
	if strings.TrimSpace(event.Type) == "" {
		return RuntimeState{}, errors.New("workflow event type is required")
	}
	if state.Task.ID != id { return RuntimeState{}, errors.New("workflow Task ID cannot change") }
	event.SchemaVersion = state.SchemaVersion
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
	n, writeErr := f.Write(append(b, '\n'))
	if writeErr == nil && n != len(b)+1 { writeErr = errors.New("short workflow event write") }
	return errors.Join(writeErr, f.Sync(), f.Close())
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
	if err := s.rejectLegacyTask(id); err != nil {
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
	info, err := f.Stat()
	if err != nil { return nil, err }
	if info.Size() > 0 {
		last := []byte{0}
		if _, err := f.ReadAt(last, info.Size()-1); err != nil { return nil, err }
		if last[0] != '\n' { return nil, errors.New("workflow event log has an incomplete tail") }
	}
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
	return taskpath.ActiveTaskDir(s.Root, string(id))
}

func (s *Store) legacyTaskDir(id TaskID) string {
	return taskpath.LegacyTaskDir(s.Root, string(id))
}

func (s *Store) legacyTaskError(id TaskID) error {
	if err := taskpath.ValidateID(string(id)); err != nil { return err }
	marker := filepath.Join(s.taskDir(id), legacyMigrationFile)
	if _, err := os.Lstat(marker); err == nil {
		return fmt.Errorf("workflow Task %s has a migration marker at %s; stop AIW writers and manually relocate the complete Task directory to %s", id, marker, s.taskDir(id))
	} else if !errors.Is(err, os.ErrNotExist) { return err }
	path := s.legacyTaskDir(id)
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("workflow Task %s remains at legacy path %s; stop AIW writers and manually relocate it to %s", id, path, s.taskDir(id))
	} else if !errors.Is(err, os.ErrNotExist) { return err }
	return nil
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
	path := filepath.Join(s.Root, runtimeLocksDir, string(id)+".lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err == nil || !errors.Is(err, os.ErrExist) { return file, err }
	file, err = os.OpenFile(path, os.O_RDWR, 0)
	if err != nil { return nil, err }
	if !isSystemLock(file) { _ = file.Close(); return nil, errors.New("legacy or incomplete Task lock requires managed reconciliation") }
	if err := systemTaskLock(file); err != nil { _ = file.Close(); return nil, err }
	return file, nil
}

// TaskLocked reports whether a Core writer currently holds the Task lock. It
// is intended for read-only repair previews; callers that will write must use
// AcquireTaskLock to avoid a check-then-write race.
func (s *Store) TaskLocked(id TaskID) (bool, error) {
	if err := validateTaskID(id); err != nil {
		return false, err
	}
	file, err := os.OpenFile(filepath.Join(s.Root, runtimeLocksDir, string(id)+".lock"), os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil { return false, err }
	if !isSystemLock(file) { _ = file.Close(); return true, nil }
	if err := systemTaskLock(file); err != nil { _ = file.Close(); return true, nil }
	return false, systemTaskUnlock(file)
}

// AcquireTaskLock serializes a metadata repair with all Workflow Core state
// writers. The returned release function must be called after the repair.
func (s *Store) AcquireTaskLock(id TaskID) (func() error, error) {
	file, err := s.lock(id)
	if err != nil {
		return nil, err
	}
	return func() error {
		return releaseTaskLock(file)
	}, nil
}

func (s *Store) save(state RuntimeState) error {
	state.Summary = DeriveSummary(state)
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(s.path(state.Task.ID, runtimeStateFile), append(b, '\n'))
}

// rejectLegacyTask keeps normal Core reads and writes from migrating old
// runtime records implicitly. The explicit storage command owns data moves.
func (s *Store) rejectLegacyTask(id TaskID) error {
	return s.legacyTaskError(id)
}

func validateTaskID(id TaskID) error {
	if strings.TrimSpace(string(id)) == "" || id == "." || id == ".." || strings.ContainsAny(string(id), `/\\`) {
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
	_ = releaseTaskLock(file)
}

func findAttempt(attempts []Attempt, id AttemptID) (Attempt, bool) {
	for _, attempt := range attempts {
		if attempt.ID == id {
			return attempt, true
		}
	}
	return Attempt{}, false
}
