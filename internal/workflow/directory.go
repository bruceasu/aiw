package workflow

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// LoadFromDirectory reads exactly the discovered location, without migration,
// event-log creation, or fallback to an active Task with the same identity.
func LoadFromDirectory(dir string, id TaskID) (RuntimeState, error) {
	if err := validateTaskID(id); err != nil {
		return RuntimeState{}, err
	}
	return (&Store{}).loadFromDir(id, dir)
}

// EnsureCompatibleInDirectory only creates a missing state file. Exclusive
// creation preserves concurrently appearing records and never starts execution.
func EnsureCompatibleInDirectory(dir string, state RuntimeState) (RuntimeState, bool, error) {
	loaded, err := LoadFromDirectory(dir, state.Task.ID)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return loaded, false, err
	}
	if err := normalizeRuntimeState(&state); err != nil {
		return RuntimeState{}, false, err
	}
	if err := ValidateRuntimeState(state); err != nil {
		return RuntimeState{}, false, err
	}
	state.Summary = DeriveSummary(state)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return RuntimeState{}, false, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return RuntimeState{}, false, err
	}
	file, err := os.OpenFile(filepath.Join(dir, runtimeStateFile), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		loaded, err = LoadFromDirectory(dir, state.Task.ID)
		return loaded, false, err
	}
	if err != nil {
		return RuntimeState{}, false, err
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return RuntimeState{}, false, err
	}
	return state, true, nil
}
