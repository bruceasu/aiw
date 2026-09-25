package workflow

import (
	"encoding/json"
	"errors"
)

type AuxiliarySourceSeal struct {
	TaskID TaskID `json:"task_id"`
	SourceVersion string `json:"source_version"`
	Sources []AuxiliarySource `json:"sources"`
	Jobs []AuxiliaryJob `json:"jobs"`
	References []ActorReference `json:"references"`
}

// SealAuxiliarySources preserves the complete fixed inputs and original Task
// facts outside the disposable workspace. Cleanup's read-only host validator
// must call VerifyAuxiliarySourceSeal immediately before deleting a worktree.
func (s *Store) SealAuxiliarySources(id TaskID) (ActorReference, error) {
	lock, err := s.auxiliaryProjectLock()
	if err != nil { return ActorReference{}, err }
	defer systemTaskUnlock(lock)
	r, err := s.readAuxiliaryResources()
	if err != nil { return ActorReference{}, err }
	state, err := s.Load(id)
	if err != nil { return ActorReference{}, err }
	if state.Protocol == nil || state.Protocol.Auxiliary == nil || state.PendingEvent != nil { return ActorReference{}, errors.New("reconciled Task sources are required before sealing") }
	a := state.Protocol.Auxiliary
	if len(a.Sources) == 0 { return ActorReference{}, errors.New("Task source versions are unavailable") }
	latest := a.Sources[len(a.Sources)-1]
	seal := AuxiliarySourceSeal{TaskID: id, SourceVersion: latest.Version, Sources: a.Sources, Jobs: a.Jobs}
	seen := map[ActorReference]bool{}
	add := func(ref ActorReference) error {
		if seen[ref] { return nil }
		var raw json.RawMessage
		if err := s.ReadExecutionArtifact(id, ref, &raw); err != nil { return err }
		seen[ref] = true
		seal.References = append(seal.References, ref)
		return nil
	}
	for _, source := range a.Sources { for _, ref := range source.References { if err := add(ref); err != nil { return ActorReference{}, err } } }
	for _, job := range a.Jobs { if job.Output != nil { if err := add(*job.Output); err != nil { return ActorReference{}, err } } }
	body, err := json.Marshal(seal)
	if err != nil { return ActorReference{}, err }
	key := contentDigest(append([]byte("source-seal:"), body...))
	reservation := AuxiliaryJob{Key: key, Kind: "source-seal", Owner: id, Sponsor: id, SourceID: latest.ID, SourceVersion: latest.Version, InputDigest: contentDigest(body), Prompt: string(body), State: "completed"}
	if err := s.reserveAuxiliaryQueue(&r, reservation, latest); err != nil { return ActorReference{}, err }
	var ref ActorReference
	_, err = s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.auxiliary.sources-sealed", Detail: key}, func(current *RuntimeState) error {
		ref, err = s.persistProtocolArtifactLocked(id, "auxiliary-source-seal", seal)
		if err != nil { return err }
		current.Protocol.Auxiliary.Sealed = &ref
		return nil
	})
	if err != nil { return ref, err }
	q := r.Queue[key]; q.Terminal = true; r.Queue[key] = q
	return ref, s.saveAuxiliaryResources(r)
}

func (s *Store) VerifyAuxiliarySourceSeal(state RuntimeState, ref ActorReference) error {
	if state.Protocol == nil || state.Protocol.Auxiliary == nil || ref.Kind != "auxiliary-source-seal" { return errors.New("cleanup requires an E05 source seal") }
	var seal AuxiliarySourceSeal
	if err := s.ReadExecutionArtifact(state.Task.ID, ref, &seal); err != nil { return err }
	a := state.Protocol.Auxiliary
	if seal.TaskID != state.Task.ID || len(a.Sources) == 0 || seal.SourceVersion != a.Sources[len(a.Sources)-1].Version { return errors.New("cleanup source seal is stale") }
	// New auxiliary outputs are already Task-owned. New sources or altered
	// frozen job inputs require a new seal; queue state alone does not.
	covered := map[ActorReference]bool{}
	for _, item := range seal.References {
		var raw json.RawMessage
		if err := s.ReadExecutionArtifact(state.Task.ID, item, &raw); err != nil { return err }
		covered[item] = true
	}
	for _, source := range a.Sources { for _, item := range source.References { if !covered[item] { return errors.New("source seal omitted an original Task artifact") } } }
	jobs := map[string]string{}
	for _, job := range seal.Jobs { jobs[job.Key] = job.InputDigest }
	for _, job := range a.Jobs { if jobs[job.Key] != job.InputDigest { return errors.New("source seal omitted fixed auxiliary input") } }
	return nil
}
