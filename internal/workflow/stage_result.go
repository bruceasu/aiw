package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
)

func (s *Store) persistStageResultLocked(id TaskID, result StageResult) (ActorReference, error) {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil { return ActorReference{}, err }
	content = append(content, '\n')
	ref := ActorReference{Kind: "stage-result", Path: "reports/stage-results/"+contentDigest([]byte(result.RequestID))+".json", SHA256: contentDigest(content)}
	path := s.path(id, ref.Path)
	old, err := os.ReadFile(path)
	if err == nil && !bytes.Equal(old, content) { return ActorReference{}, errors.New("stage terminal result cannot be overwritten") }
	if err != nil && !errors.Is(err, os.ErrNotExist) { return ActorReference{}, err }
	if err := durableWrite(path, content); err != nil { return ActorReference{}, err }
	return ref, nil
}

// PersistStageResult is the executor's save-before-return boundary. A result
// without a Core reference remains discoverable by its exact request ID.
func (s *Store) PersistStageResult(id TaskID, result StageResult) (ActorReference, error) {
	if err := s.requireExecutionServices(); err != nil { return ActorReference{}, err }
	lock, err := s.lock(id)
	if err != nil { return ActorReference{}, err }
	defer unlock(lock)
	state, err := s.Load(id)
	if err != nil { return ActorReference{}, err }
	if state.SchemaVersion != DurableSchemaVersion || !isSystemLock(lock) { return ActorReference{}, errors.New("durable stage result requires the migrated Store") }
	record, err := stageRecord(&state, result.RequestID)
	if err != nil { return ActorReference{}, err }
	if err := validateStageResultIdentity(id, *record, result); err != nil { return ActorReference{}, err }
	if err := s.ExecutionServices.ValidateResult(state, record.Request, result); err != nil { return ActorReference{}, err }
	return s.persistStageResultLocked(id, result)
}

func validateStageResultIdentity(id TaskID, record StageRecord, result StageResult) error {
	r := record.Request
	if result.RequestID != r.ID || result.TaskID != id || result.WorkItemID != r.WorkItemID || result.AttemptID != r.AttemptID || result.SessionID != r.SessionID || result.Turn != r.Turn || result.InputDigest != r.InputDigest || result.LeaseGeneration != r.LeaseGeneration || !result.Terminal || result.Executor == "" || result.Executor != record.Executor { return errors.New("stage result identity mismatch") }
	return nil
}

func (s *Store) ReadStageResult(id TaskID, requestID string) (StageResult, error) {
	var result StageResult
	if err := validateTaskID(id); err != nil { return result, err }
	path := "reports/stage-results/"+contentDigest([]byte(requestID))+".json"
	content, err := os.ReadFile(s.path(id, path))
	if err != nil { return result, err }
	ref := ActorReference{Kind: "stage-result", Path: path, SHA256: contentDigest(content)}
	if err := s.ReadExecutionArtifact(id, ref, &result); err != nil { return result, err }
	state, err := s.Load(id)
	if err != nil { return result, err }
	record, err := stageRecord(&state, requestID)
	if err != nil { return result, err }
	if err := validateStageResultIdentity(id, *record, result); err != nil { return result, err }
	return result, nil
}
