package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
)

// Host receipts are indexed by the original request, so a process exit followed
// by a failed Core commit is recoverable without executing the request again.
func (s *Store) PersistVerificationReceipt(id TaskID, receipt VerificationReceipt) error {
	if err := validateTaskID(id); err != nil { return err }
	lock, err := s.lock(id)
	if err != nil { return err }
	defer unlock(lock)
	state, err := s.Load(id)
	if err != nil { return err }
	if state.SchemaVersion != DurableSchemaVersion || !isSystemLock(lock) { return errors.New("host receipt requires the migrated Store") }
	record, err := stageRecord(&state, receipt.Request.ID)
	if err != nil { return err }
	if !equalJSON(record.Request, receipt.Request) || receipt.Request.TaskID != id || (record.Dispatch != "unknown" && record.Dispatch != "dispatched" && record.Dispatch != "terminal") { return errors.New("receipt has no exact managed dispatch") }
	if err := validateStageResultIdentity(id, *record, receipt.Result); err != nil { return err }
	data, err := json.Marshal(receipt)
	if err != nil { return err }
	path := s.path(id, "reports/verification-host/"+contentDigest([]byte(receipt.Request.ID))+".json")
	old, err := os.ReadFile(path)
	if err == nil && !bytes.Equal(old, data) { return errors.New("host receipt cannot be replaced") }
	if err != nil && !errors.Is(err, os.ErrNotExist) { return err }
	return durableWrite(path, data)
}

func (s *Store) ReadVerificationReceipt(id TaskID, requestID string) (VerificationReceipt, error) {
	var receipt VerificationReceipt
	if err := validateTaskID(id); err != nil { return receipt, err }
	path := "reports/verification-host/"+contentDigest([]byte(requestID))+".json"
	data, err := os.ReadFile(s.path(id, path))
	if err != nil { return receipt, err }
	ref := ActorReference{Kind: "verification-receipt", Path: path, SHA256: contentDigest(data)}
	if err := s.ReadExecutionArtifact(id, ref, &receipt); err != nil { return receipt, err }
	if receipt.Request.TaskID != id || receipt.Request.ID != requestID { return receipt, errors.New("host receipt request mismatch") }
	return receipt, nil
}
