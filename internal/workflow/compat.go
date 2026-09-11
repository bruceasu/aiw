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
	}
	state.Summary = DeriveSummary(state)
	return state
}

// EnsureCompatible returns an existing projection or creates the inactive
// projection supplied by a durable metadata adapter.
func (s *Store) EnsureCompatible(state RuntimeState) (RuntimeState, error) {
	loaded, err := s.Load(state.Task.ID)
	if err == nil {
		return loaded, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RuntimeState{}, err
	}
	return s.Create(state)
}
