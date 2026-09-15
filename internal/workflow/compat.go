package workflow

import (
	"errors"
	"os"
)

// NewCompatibleRuntime creates the local runtime projection for a durable,
// previously existing Task. Compatibility initialization deliberately creates
// neither an Attempt nor a write lease: durable metadata cannot establish that
// a process is still entitled to write a workspace.
func NewCompatibleRuntime(task TaskReference, planning PlanningState, delivery DeliveryState) RuntimeState {
	if planning == "" {
		planning = PlanningDraft
	}
	if delivery == "" {
		delivery = DeliveryUnmanaged
	}
	state := RuntimeState{
		SchemaVersion: SchemaVersion,
		Task:          task,
		Planning:      planning,
		Delivery:      delivery,
		WorkItems:     []WorkItem{},
		Attempts:      []Attempt{},
		Gates:         []Gate{},
		Evidence:      []Evidence{},
		ActorHandoffs: []ActorHandoff{},
	}
	state.Summary = DeriveSummary(state)
	return state
}

// EnsureCompatible returns an existing projection or creates the inactive
// projection supplied by a durable metadata adapter.
func (s *Store) EnsureCompatible(state RuntimeState) (RuntimeState, error) {
	loaded, err := s.Load(state.Task.ID)
	if err == nil {
		if err := s.ensureEventLog(state.Task.ID); err != nil {
			return RuntimeState{}, err
		}
		if _, statErr := os.Stat(s.path(state.Task.ID, runtimeStateFile)); statErr == nil {
			return loaded, nil
		}
		lock, lockErr := s.lock(state.Task.ID)
		if lockErr != nil {
			return RuntimeState{}, lockErr
		}
		defer unlock(lock)
		if migrateErr := s.migrateLegacyLocked(state.Task.ID); migrateErr != nil {
			return RuntimeState{}, migrateErr
		}
		return s.Load(state.Task.ID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RuntimeState{}, err
	}
	return s.Create(state)
}
