package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/ai"
	"aiw/internal/repo"
)

type Store struct{ Root string }

func NewStore(root string) *Store {
	if root == "" {
		root = filepath.Join(repo.Root(), ".ai")
	}
	return &Store{Root: root}
}
func (s *Store) sessionDir(id string) string { return filepath.Join(s.Root, "sessions", id) }
func (s *Store) path(id string, names ...string) string {
	return filepath.Join(append([]string{s.sessionDir(id)}, names...)...)
}
func (s *Store) lock(id string) (*os.File, error) {
	if err := s.requireWritable(id); err != nil { return nil, err }
	if err := os.MkdirAll(filepath.Join(s.Root, "locks"), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(s.Root, "locks", id+".lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
}
func unlock(f *os.File) {
	if f != nil {
		name := f.Name()
		_ = f.Close()
		_ = os.Remove(name)
	}
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
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func (s *Store) Create(id, title, workspace, backend, model, instructions string) (Status, error) {
	if !validSessionID(id) {
		return Status{}, errors.New("invalid session id")
	}
	if _, err := s.Resolve(id); err == nil { return Status{}, errors.New("session already exists") } else if !errors.Is(err, ErrSessionNotFound) { return Status{}, err }
	dir := s.sessionDir(id)
	if _, err := os.Stat(dir); err == nil {
		return Status{}, errors.New("session already exists")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	status := Status{SchemaVersion: 1, Session: SessionInfo{ID: id, Title: title, State: StateCreated, CreatedAt: now, UpdatedAt: now}, Backend: BackendInfo{Name: backend, Model: model}, Workspace: WorkspaceInfo{Path: workspace}, Instructions: InstructionsInfo{File: "instructions.md", MemoryFile: "memory.md"}, Result: ResultInfo{Status: "not_started"}}
	if err := os.MkdirAll(filepath.Join(dir, "prompts"), 0o755); err != nil {
		return Status{}, err
	}
	for _, child := range []string{"outputs", "artifacts"} {
		if err := os.MkdirAll(filepath.Join(dir, child), 0o755); err != nil {
			return Status{}, err
		}
	}
	if instructions == "" {
		instructions = "Preserve Task scope. Read the task artifacts before acting. Report validation."
	}
	if err := atomicWrite(filepath.Join(dir, "instructions.md"), []byte(instructions+"\n")); err != nil {
		return Status{}, err
	}
	if err := atomicWrite(filepath.Join(dir, "memory.md"), []byte("# Session Memory\n\n")); err != nil {
		return Status{}, err
	}
	if err := atomicWrite(filepath.Join(dir, "events.jsonl"), nil); err != nil {
		return Status{}, err
	}
	if err := s.saveActive(status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func (s *Store) Load(id string) (Status, error) {
	location, err := s.Resolve(id)
	return location.Status, err
}
func (s *Store) Save(status Status) error {
	if status.archived { return errors.New("archived session is read-only") }
	if err := s.requireWritable(status.Session.ID); err != nil { return err }
	return s.saveActive(status)
}
func (s *Store) saveActive(status Status) error {
	status.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(s.path(status.Session.ID, "status.json"), append(b, '\n'))
}
func (s *Store) Update(id string, fn func(*Status) error) (Status, error) {
	lock, err := s.lock(id)
	if err != nil {
		return Status{}, fmt.Errorf("session is locked: %w", err)
	}
	defer unlock(lock)
	status, err := s.Load(id)
	if err != nil {
		return Status{}, err
	}
	if err := fn(&status); err != nil {
		return Status{}, err
	}
	if err := s.Save(status); err != nil {
		return Status{}, err
	}
	return status, nil
}
func (s *Store) ReadText(id, name string) (string, error) {
	path, err := s.readPath(id, name)
	if err != nil { return "", err }
	b, err := os.ReadFile(path)
	return string(b), err
}

// ReadTurnUsage returns the normalized Provider envelope for a persisted turn.
// Older managed turns without a sidecar remain readable as explicit unknown
// usage and are never rewritten as a side effect of this read.
func (s *Store) ReadTurnUsage(id string, turn int, provider, model string) (*ai.UsageEnvelope, error) {
	if !validSessionID(id) || turn <= 0 { return nil, errors.New("invalid Session turn identity") }
	path := s.path(id, "outputs", fmt.Sprintf("%04d-usage.json", turn))
	content, err := os.ReadFile(path)
	if err == nil {
		var usage ai.UsageEnvelope
		if err := json.Unmarshal(content, &usage); err != nil { return nil, fmt.Errorf("decode Session turn usage: %w", err) }
		if usage.Version != ai.UsageEnvelopeVersion { return nil, errors.New("unsupported Session turn usage version") }
		if ai.ExpireUsageEvidence(&usage, time.Now().UTC()) {
			updated, err := json.Marshal(&usage)
			if err != nil { return nil, fmt.Errorf("encode expired Session turn usage: %w", err) }
			if err := atomicWrite(path, updated); err != nil { return nil, fmt.Errorf("expire Session turn raw usage evidence: %w", err) }
		}
		return &usage, nil
	}
	if !errors.Is(err, os.ErrNotExist) { return nil, err }
	status, loadErr := s.Load(id)
	if loadErr != nil { return nil, loadErr }
	result := TurnResult{}
	result.StartedAt, _ = time.Parse(time.RFC3339Nano, status.Execution.LastStartedAt)
	result.CompletedAt, _ = time.Parse(time.RFC3339Nano, status.Execution.LastCompletedAt)
	ensureTurnUsage(&result, provider, model)
	return result.Usage, nil
}

func (s *Store) ReadArtifact(id, name string) (string, error) {
	if filepath.Base(name) != name {
		return "", errors.New("invalid artifact name")
	}
	return s.ReadText(id, filepath.Join("artifacts", name))
}
func (s *Store) WriteArtifact(id, name string, content []byte) error {
	if err := s.requireWritable(id); err != nil { return err }
	if filepath.Base(name) != name {
		return errors.New("invalid artifact name")
	}
	return atomicWrite(s.path(id, "artifacts", name), content)
}
func (s *Store) AppendMemory(id, text string) error {
	_, err := s.Update(id, func(status *Status) error {
		path := s.path(id, "memory.md")
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = fmt.Fprintf(f, "\n%s\n", strings.TrimSpace(text))
		return err
	})
	return err
}
func (s *Store) SavePrompt(id string, turn int, phase, prompt string) error {
	if err := s.requireWritable(id); err != nil { return err }
	sum := sha256.Sum256([]byte(prompt))
	name := fmt.Sprintf("%04d-%s-%s.md", turn, safeName(phase), hex.EncodeToString(sum[:])[:8])
	return atomicWrite(s.path(id, "prompts", name), []byte(prompt))
}
func safeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "turn"
	}
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}
func (s *Store) AppendEvent(id string, event interface{}) error {
	if err := s.requireWritable(id); err != nil { return err }
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.path(id, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
func (s *Store) Delete(id string) error {
	if err := s.requireWritable(id); err != nil { return err }
	return os.RemoveAll(s.sessionDir(id))
}
