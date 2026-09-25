package workflow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExecutionReport preserves the original response independently of mutable
// Session projections. It is an execution fact, not proof of acceptance.
type ExecutionReport struct {
	RequestID string `json:"request_id,omitempty"`
	InputReference *ActorReference `json:"input_reference,omitempty"`
	OriginalRequestID string `json:"original_request_id,omitempty"`
	SchemaVersion int `json:"schema_version"`
	TaskID TaskID `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	SessionID string `json:"session_id"`
	Turn int `json:"turn"`
	Actor ActorKind `json:"actor"`
	Workspace string `json:"workspace"`
	AISelection *AISelection `json:"ai_selection,omitempty"`
	PreparedAt string `json:"prepared_at"`
	DispatchedAt string `json:"dispatched_at"`
	Source string `json:"source"`
	OutputSHA256 string `json:"output_sha256"`
	Output string `json:"output"`
}

// PersistExecutionReport is idempotent for an identical bound response and
// rejects replacement of a prior response. Invalid outcome text is preserved
// too: report validation must not erase the original execution facts.
func (s *Store) PersistExecutionReport(request PreparedAgentRequest, source, output string) (ActorReference, error) {
	if err := validateTaskID(request.TaskID); err != nil { return ActorReference{}, err }
	if request.TaskID == "." || request.TaskID == ".." || request.WorkItemID == "" || request.AttemptID == "" ||
		strings.TrimSpace(request.SessionID) == "" || request.ExpectedSessionTurn <= 0 ||
		strings.TrimSpace(request.Workspace) == "" || request.DispatchedAt == "" || strings.TrimSpace(source) == "" {
		return ActorReference{}, errors.New("execution report requires exact Task, Work Item, Attempt, Session, turn, workspace and dispatch bindings")
	}
	sum := sha256.Sum256([]byte(output))
	report := ExecutionReport{
		InputReference: request.InputReference,
		SchemaVersion: 1, TaskID: request.TaskID, WorkItemID: request.WorkItemID,
		AttemptID: request.AttemptID, SessionID: request.SessionID, Turn: request.ExpectedSessionTurn,
		Actor: ActorCoder, Workspace: request.Workspace, AISelection: request.AISelection,
		PreparedAt: request.PreparedAt, DispatchedAt: request.DispatchedAt, Source: source,
		OutputSHA256: hex.EncodeToString(sum[:]), Output: output,
	}
	if request.InputReference != nil { report.RequestID = ExecutionRequestID(request) }
	if request.ReportOrigin != nil { report.OriginalRequestID = ExecutionRequestID(*request.ReportOrigin) }
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil { return ActorReference{}, err }
	content = append(content, '\n')
	// JSON encoding avoids ambiguous separators and path injection in IDs.
	identity, err := json.Marshal([]any{request.TaskID, request.WorkItemID, request.AttemptID, request.SessionID, request.ExpectedSessionTurn})
	if err != nil { return ActorReference{}, err }
	key := sha256.Sum256(identity)
	relative := filepath.ToSlash(filepath.Join("reports", "executions", hex.EncodeToString(key[:])+".json"))
	lock, err := s.lock(request.TaskID)
	if err != nil { return ActorReference{}, err }
	defer unlock(lock)
	state, err := s.Load(request.TaskID)
	if err != nil { return ActorReference{}, err }
	attempt, found := findAttempt(state.Attempts, request.AttemptID)
	if !found || attempt.WorkItemID != request.WorkItemID || attempt.Workspace != request.Workspace {
		return ActorReference{}, errors.New("execution report does not match the persisted Attempt")
	}
	if request.InputReference != nil {
		if state.SchemaVersion == DurableSchemaVersion {
			record, err := stageRecord(&state, ExecutionRequestID(request))
			if err != nil || record.Request.Input != *request.InputReference || record.Dispatch == "intent" || record.Dispatch == "not-dispatched" { return ActorReference{}, errors.New("execution report does not match its durable dispatch") }
			report.Actor = record.Request.Actor
			content, err = json.MarshalIndent(report, "", "  ")
			if err != nil { return ActorReference{}, err }
			content = append(content, '\n')
		} else {
			current := state.Automation.PreparedRequest
			if current == nil || ExecutionRequestID(*current) != ExecutionRequestID(request) || current.InputReference == nil || *current.InputReference != *request.InputReference || current.DispatchedAt != request.DispatchedAt { return ActorReference{}, errors.New("execution report does not match the prepared input version") }
		}
	}
	// Legacy directory promotion belongs to the Store migration protocol, not
	// this artifact writer. Never create a second partial Task directory.
	if _, err := os.Stat(s.path(request.TaskID, runtimeStateFile)); err != nil { return ActorReference{}, err }
	target := s.path(request.TaskID, filepath.FromSlash(relative))
	previous, err := os.ReadFile(target)
	if err == nil {
		if !bytes.Equal(previous, content) { return ActorReference{}, fmt.Errorf("execution report conflicts with preserved response: %s", relative) }
	} else if errors.Is(err, os.ErrNotExist) {
		if err := writeExecutionReport(target, content); err != nil { return ActorReference{}, err }
	} else {
		return ActorReference{}, err
	}
	// Also sync on a replay: a previous call may have published the name but
	// failed before confirming the final flush. Never return a reference then.
	file, err := os.OpenFile(target, os.O_RDWR, 0)
	if err != nil { return ActorReference{}, err }
	err = file.Sync()
	closeErr := file.Close()
	if err != nil { return ActorReference{}, err }
	if closeErr != nil { return ActorReference{}, closeErr }
	digest := sha256.Sum256(content)
	return ActorReference{Kind: "execution-report", Path: relative, SHA256: hex.EncodeToString(digest[:])}, nil
}

func (s *Store) ReadExecutionReport(request PreparedAgentRequest) (ExecutionReport, error) {
	var report ExecutionReport
	if err := validateTaskID(request.TaskID); err != nil { return report, err }
	if request.TaskID == "." || request.TaskID == ".." { return report, errors.New("invalid Task identity") }
	relative := "reports/executions/" + ExecutionRequestID(request) + ".json"
	content, err := os.ReadFile(s.path(request.TaskID, filepath.FromSlash(relative)))
	if err != nil { return report, err }
	if err := json.Unmarshal(content, &report); err != nil { return report, err }
	if report.SchemaVersion != 1 || report.TaskID != request.TaskID || report.WorkItemID != request.WorkItemID || report.AttemptID != request.AttemptID || report.SessionID != request.SessionID || report.Turn != request.ExpectedSessionTurn || report.OutputSHA256 != contentDigest([]byte(report.Output)) { return report, errors.New("archived execution report binding is invalid") }
	return report, nil
}

// The Task lock excludes competing writers. Platform-specific replacement
// and directory durability for the new protocol remain E02 activation work.
func writeExecutionReport(target string, content []byte) error {
	if durablePlatformSupported() { return durableWrite(target, content) }
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { return err }
	file, err := os.CreateTemp(filepath.Dir(target), ".report-*")
	if err != nil { return err }
	name := file.Name()
	defer os.Remove(name)
	if _, err := file.Write(content); err != nil { _ = file.Close(); return err }
	if err := file.Sync(); err != nil { _ = file.Close(); return err }
	if err := file.Close(); err != nil { return err }
	return os.Rename(name, target)
}
