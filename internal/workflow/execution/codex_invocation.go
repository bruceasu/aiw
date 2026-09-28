package execution

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aiw/internal/workflow"
)

const maxCodexInvocationOutput = 16 << 20

// CodexInvocation is an append-only receipt for one frozen StageRequest. It
// deliberately excludes prompt text, credentials, and mutable Session state.
type CodexInvocation struct {
	Version       int                 `json:"version"`
	RequestID     string              `json:"request_id"`
	TaskID        workflow.TaskID     `json:"task_id"`
	WorkItemID    workflow.WorkItemID `json:"work_item_id"`
	AttemptID     workflow.AttemptID  `json:"attempt_id"`
	SessionID     string              `json:"session_id"`
	Turn          int                 `json:"turn"`
	InputDigest   string              `json:"input_digest"`
	RequestDigest string              `json:"request_digest"`
	ProcessKey    string              `json:"process_key"`
	State         string              `json:"state"`
	PID           int                 `json:"pid,omitempty"`
	ProcessToken  string              `json:"process_token,omitempty"`
	ExitCode      *int                `json:"exit_code,omitempty"`
	TerminalEvent string              `json:"terminal_event,omitempty"`
	OutputDigest  string              `json:"output_digest,omitempty"`
	OutputBytes   int64               `json:"output_bytes"`
	StartedAt     time.Time           `json:"started_at"`
	CompletedAt   time.Time           `json:"completed_at,omitempty"`
	Error         string              `json:"error,omitempty"`
}

// CodexInvocationProof is the read-only result of reconciling a journal.
type CodexInvocationProof struct {
	Receipt  CodexInvocation
	Events   []byte
	Terminal bool
	Running  bool
	Stopped  bool
}

// CodexInvocationJournal implements ai.InvocationObserver. State and stdout
// are synced before success is reported to the CLI adapter.
type CodexInvocationJournal struct {
	mu          sync.Mutex
	root        string
	request     workflow.StageRequest
	requestHash string
	recordPath  string
	outputPath  string
	current     CodexInvocation
}

func NewCodexInvocationJournal(root string, request workflow.StageRequest) (*CodexInvocationJournal, error) {
	if strings.TrimSpace(root) == "" || request.ID == "" || request.TaskID == "" || request.WorkItemID == "" || request.AttemptID == "" || request.SessionID == "" || request.Turn < 1 || len(request.InputDigest) != 64 {
		return nil, errors.New("Codex invocation journal requires a complete frozen StageRequest")
	}
	if request.Phase != workflow.PhaseCoder || request.Model == nil || !strings.EqualFold(request.Model.Provider, "codex") {
		return nil, errors.New("Codex invocation journal accepts only a Codex Coder request")
	}
	encoded, err := json.Marshal(request)
	if err != nil { return nil, err }
	requestHash := sha256.Sum256(encoded)
	nameHash := sha256.Sum256([]byte(request.ID))
	root, err = filepath.Abs(root)
	if err != nil { return nil, err }
	base := hex.EncodeToString(nameHash[:])
	return &CodexInvocationJournal{root: root, request: request, requestHash: hex.EncodeToString(requestHash[:]),
		recordPath: filepath.Join(root, base+".jsonl"), outputPath: filepath.Join(root, base+".events.jsonl")}, nil
}

// Prepare reserves the request before process start. A prior receipt always
// blocks a second invocation, even if its final state is unknown.
func (j *CodexInvocationJournal) Prepare() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := os.MkdirAll(j.root, 0o700); err != nil { return err }
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil { return err }
	initial := CodexInvocation{Version: 1, RequestID: j.request.ID, TaskID: j.request.TaskID,
		WorkItemID: j.request.WorkItemID, AttemptID: j.request.AttemptID, SessionID: j.request.SessionID,
		Turn: j.request.Turn, InputDigest: j.request.InputDigest, RequestDigest: j.requestHash,
		ProcessKey: hex.EncodeToString(key), State: "prepared", StartedAt: time.Now().UTC()}
	if err := appendSyncedJSON(j.recordPath, initial, true); err != nil { return err }
	j.current = initial
	return nil
}

func (j *CodexInvocationJournal) ProcessKey() string {
	j.mu.Lock(); defer j.mu.Unlock()
	return j.current.ProcessKey
}

func (j *CodexInvocationJournal) Started(pid int) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.current.State != "prepared" || pid <= 0 { return errors.New("Codex process start has no prepared receipt or valid PID") }
	token, err := managedProcessToken(pid, j.current.ProcessKey)
	if err != nil { return fmt.Errorf("capture Codex process identity: %w", err) }
	j.current.PID, j.current.ProcessToken, j.current.State = pid, token, "running"
	j.current.StartedAt = time.Now().UTC()
	return j.appendCurrentLocked()
}

func (j *CodexInvocationJournal) Write(p []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.current.State != "running" { return 0, errors.New("Codex output arrived without a durable running receipt") }
	if j.current.OutputBytes+int64(len(p)) > maxCodexInvocationOutput { return 0, errors.New("Codex JSONL exceeded the host output bound") }
	f, err := os.OpenFile(j.outputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil { return 0, err }
	n, writeErr := f.Write(p)
	if writeErr == nil && n != len(p) { writeErr = io.ErrShortWrite }
	if writeErr == nil { writeErr = f.Sync() }
	closeErr := f.Close()
	if writeErr != nil { return n, errors.Join(writeErr, closeErr) }
	if closeErr != nil { return n, closeErr }
	j.current.OutputBytes += int64(n)
	return n, nil
}

func (j *CodexInvocationJournal) Finished(exitCode int) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.current.State != "running" { return errors.New("Codex process finished without a running receipt") }
	events, err := os.ReadFile(j.outputPath)
	if errors.Is(err, os.ErrNotExist) && j.current.OutputBytes == 0 { events, err = nil, nil }
	if err != nil { return err }
	terminal, found := codexTerminalEvent(events)
	if !found {
		j.current.State, j.current.Error = "unknown", "Codex process exited without a complete terminal turn event"
		j.current.ExitCode = &exitCode
		j.current.CompletedAt = time.Now().UTC()
	} else {
		j.current.State, j.current.TerminalEvent = "terminal", terminal
		j.current.ExitCode = &exitCode
		j.current.CompletedAt = time.Now().UTC()
		digest := sha256.Sum256(events)
		j.current.OutputDigest = hex.EncodeToString(digest[:])
	}
	return j.appendCurrentLocked()
}

func (j *CodexInvocationJournal) appendCurrentLocked() error { return appendSyncedJSON(j.recordPath, j.current, false) }

// Reconcile does not start, signal, or mutate an invocation. A terminal result
// is returned only when the exact receipt, complete JSONL, and stopped process
// tree are all proven.
func (j *CodexInvocationJournal) Reconcile() (CodexInvocationProof, error) {
	data, err := os.ReadFile(j.recordPath)
	if err != nil { return CodexInvocationProof{}, err }
	receipt, err := lastCompleteReceipt(data)
	if err != nil { return CodexInvocationProof{}, err }
	if receipt.RequestID != j.request.ID || receipt.RequestDigest != j.requestHash || receipt.TaskID != j.request.TaskID || receipt.WorkItemID != j.request.WorkItemID || receipt.AttemptID != j.request.AttemptID || receipt.SessionID != j.request.SessionID || receipt.Turn != j.request.Turn || receipt.InputDigest != j.request.InputDigest {
		return CodexInvocationProof{}, errors.New("Codex invocation receipt does not match the frozen StageRequest")
	}
	if receipt.PID == 0 || receipt.ProcessToken == "" || receipt.ProcessKey == "" { return CodexInvocationProof{Receipt: receipt}, nil }
	running, conclusive, err := managedProcessState(receipt.PID, receipt.ProcessToken, receipt.ProcessKey)
	if err != nil { return CodexInvocationProof{}, err }
	if running { return CodexInvocationProof{Receipt: receipt, Running: true}, nil }
	if !conclusive { return CodexInvocationProof{Receipt: receipt}, nil }
	events, err := os.ReadFile(j.outputPath)
	if errors.Is(err, os.ErrNotExist) && receipt.OutputBytes == 0 { events, err = nil, nil }
	if err != nil { return CodexInvocationProof{}, err }
	terminal, found := codexTerminalEvent(events)
	if !found { return CodexInvocationProof{Receipt: receipt, Events: events, Stopped: true}, nil }
	if receipt.State == "terminal" && (receipt.TerminalEvent != terminal || receipt.OutputBytes != int64(len(events))) {
		return CodexInvocationProof{}, errors.New("Codex terminal receipt conflicts with its durable JSONL")
	}
	if receipt.State == "terminal" && receipt.OutputDigest != digestHex(events) { return CodexInvocationProof{}, errors.New("Codex terminal output digest mismatch") }
	return CodexInvocationProof{Receipt: receipt, Events: events, Terminal: true, Stopped: true}, nil
}

func codexTerminalEvent(events []byte) (string, bool) {
	if len(events) == 0 || events[len(events)-1] != '\n' { return "", false }
	scanner := bufio.NewScanner(bytes.NewReader(events))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	terminal := ""
	for scanner.Scan() {
		var event struct { Type string `json:"type"` }
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil { return "", false }
		terminal = event.Type
	}
	if scanner.Err() != nil { return "", false }
	return terminal, terminal == "turn.completed" || terminal == "turn.failed"
}

func lastCompleteReceipt(data []byte) (CodexInvocation, error) {
	var receipt CodexInvocation
	seen := false
	lines := bytes.SplitAfter(data, []byte{'\n'})
	for index, line := range lines {
		complete := len(line) > 0 && line[len(line)-1] == '\n'
		if !complete {
			if index == len(lines)-1 { break }
			return CodexInvocation{}, errors.New("Codex invocation journal has an incomplete interior record")
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 { continue }
		var candidate CodexInvocation
		if err := json.Unmarshal(line, &candidate); err != nil { return CodexInvocation{}, fmt.Errorf("Codex invocation journal is corrupt: %w", err) }
		if candidate.Version != 1 || candidate.RequestID == "" { return CodexInvocation{}, errors.New("Codex invocation journal has an unsupported receipt") }
		receipt, seen = candidate, true
	}
	if !seen { return CodexInvocation{}, errors.New("Codex invocation journal has no complete receipt") }
	return receipt, nil
}

func appendSyncedJSON(path string, value any, exclusive bool) error {
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if exclusive { flags |= os.O_EXCL }
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil { return err }
	data, marshalErr := json.Marshal(value)
	if marshalErr == nil { data = append(data, '\n') }
	var writeErr error
	if marshalErr == nil {
		n, err := f.Write(data)
		if err != nil { writeErr = err } else if n != len(data) { writeErr = io.ErrShortWrite }
		if writeErr == nil { writeErr = f.Sync() }
	}
	return errors.Join(marshalErr, writeErr, f.Close())
}

func digestHex(data []byte) string { digest := sha256.Sum256(data); return hex.EncodeToString(digest[:]) }
