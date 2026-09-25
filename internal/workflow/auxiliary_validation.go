package workflow

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// LockAuxiliaryHost fences the whole bounded host, separately from the resource
// transaction lock. A crashed process releases the OS lock, never its budget.
func (s *Store) LockAuxiliaryHost() (func(), error) {
	if !durablePlatformSupported() { return nil, errors.New("auxiliary host requires verified local Windows storage") }
	path := filepath.Join(s.Root, "locks", "auxiliary-host.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return nil, err }
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil { return nil, err }
	if err := systemTaskLock(f); err != nil { _ = f.Close(); return nil, err }
	return func() { _ = systemTaskUnlock(f) }, nil
}

func (s *Store) RecordAuxiliaryHostGap(id TaskID, cause error) error {
	state, err := s.Load(id)
	if err != nil { return err }
	if state.Protocol == nil { return nil }
	reason := ""
	if cause != nil { reason = cause.Error(); if len(reason) > 1024 { reason = reason[:1024] } }
	_, err = s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.auxiliary.host-gap"}, func(current *RuntimeState) error {
		if current.Protocol.Auxiliary == nil { return errProtocolNoChange }
		if current.Protocol.Auxiliary.HostGap == reason { return errProtocolNoChange }
		current.Protocol.Auxiliary.HostGap = reason
		return nil
	})
	return err
}

func (s *Store) startAuxiliaryHost(id TaskID) {
	if s.AuxiliaryServices == nil { return }
	var err error
	if s.AuxiliaryServices.StartHost == nil { err = errors.New("managed detached auxiliary launcher is unavailable") } else { err = s.AuxiliaryServices.StartHost(id) }
	if err != nil && s.AuxiliaryServices.ReportGap != nil { s.AuxiliaryServices.ReportGap(id, err) }
}

// ResumeAuxiliaryHost is a startup hook, including completed Tasks and Tasks
// with a persistent Stop. The child reconciles before considering any dispatch.
func (s *Store) ResumeAuxiliaryHost(id TaskID) {
	state, err := s.Load(id)
	if err == nil && state.Protocol != nil { s.startAuxiliaryHost(id) }
}

func validateTaskAuxiliary(state RuntimeState) error {
	if state.Protocol == nil || state.Protocol.Auxiliary == nil { return nil }
	a := state.Protocol.Auxiliary
	if a.Cursor < 0 || a.Cursor > len(a.Sources) { return errors.New("invalid auxiliary source cursor") }
	sources := map[string]AuxiliarySource{}
	for _, source := range a.Sources {
		if source.Owner != state.Task.ID || source.ID != string(source.Owner)+":"+source.Version { return errors.New("auxiliary source owner mismatch") }
		copy := source; copy.ID, copy.Version = "", ""
		body, _ := json.Marshal(copy)
		if contentDigest(body) != source.Version { return errors.New("auxiliary source version mismatch") }
		for _, ref := range source.References { if !validProtocolReference(ref) { return errors.New("auxiliary source has an invalid original reference") } }
		sources[source.ID] = source
	}
	keys := map[string]bool{}
	memorySources := map[string]bool{}
	for _, job := range a.Jobs {
		source, ok := sources[job.SourceID]
		if !ok || job.Owner != state.Task.ID || job.SourceVersion != source.Version || job.Sponsor == "" || keys[job.Key] { return errors.New("auxiliary job source or identity conflict") }
		if job.InputDigest != contentDigest([]byte(job.Prompt)) { return errors.New("auxiliary input changed") }
		identity, _ := json.Marshal([]string{job.Kind, job.SourceID, job.InputDigest})
		if job.Key != contentDigest(identity) { return errors.New("auxiliary fixed key changed") }
		switch job.Kind { case "memory", "knowledge-extraction", "knowledge-summary", "verifier": default: return errors.New("unknown auxiliary operation") }
		switch job.State { case "pending", "running", "reconcile", "waiting", "completed", "unavailable": default: return errors.New("unknown auxiliary queue state") }
		if len(job.Calls) > 2 || len(job.Calls) > 1 && !job.RecoveryUsed || job.PublishAttempts < 0 || job.PublishAttempts > 2 || job.PublishAttempts > 1 && !job.RecoveryUsed { return errors.New("auxiliary shared recovery budget exceeded") }
		if len(job.Calls) > 1 && job.PublishAttempts > 1 { return errors.New("model and save recovery cannot spend separate allowances") }
		if job.State == "completed" && (job.Output == nil || !validProtocolReference(*job.Output)) { return errors.New("completed auxiliary work has no immutable output") }
		if job.Kind == "verifier" && job.State == "completed" {
			if job.Result == nil || job.Verifier == nil { return errors.New("completed Verifier has no version-bound coverage report") }
			report, err := parseVerifierOutput(job, job.Result.Text)
			if err != nil { return err }
			if !equalJSON(report, *job.Verifier) { return errors.New("Verifier projection differs from saved output") }
		}
		for _, call := range job.Calls { if call.Key != job.Key || call.Owner != job.Owner || call.Sponsor != job.Sponsor || call.Measurement.PromptDigest != job.InputDigest || call.Executor == "" { return errors.New("auxiliary dispatch reservation mismatch") } }
		if job.Result != nil && (len(job.Calls) == 0 || job.Result.RequestID != job.Calls[len(job.Calls)-1].ID || job.Result.Key != job.Key || job.Result.InputDigest != job.InputDigest || len(job.Result.Text) > AuxiliaryOutputBytes) { return errors.New("auxiliary result identity or size mismatch") }
		keys[job.Key] = true
		if job.Kind == "memory" { memorySources[job.SourceID] = true }
	}
	for i := 0; i < a.Cursor; i++ { if !memorySources[a.Sources[i].ID] { return errors.New("auxiliary source cursor advanced without registration") } }
	if a.Memory != nil && (!keys[a.Memory.JobKey] || !validProtocolReference(a.Memory.Output)) { return errors.New("Task memory does not reference a completed source job") }
	return validateTaskKnowledge(a)
}
