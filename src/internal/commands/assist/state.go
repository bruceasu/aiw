package assist

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aiw/internal/ai"
	"aiw/internal/session"
)

const metadataFile = "auxiliary.json"

type pending struct {
	Turn int `json:"turn"`
	Phase string `json:"phase"`
	Input string `json:"input"`
	BriefDigest string `json:"brief_digest"`
}

type record struct {
	Version int `json:"version"`
	ID string `json:"id"`
	Kind string `json:"kind"`
	Workspace string `json:"workspace"`
	Title string `json:"title"`
	Goal string `json:"goal"`
	Flavor string `json:"flavor,omitempty"`
	SkillDigest string `json:"skill_digest"`
	SourceDigest string `json:"source_digest"`
	SourceFile string `json:"source_file"`
	Brief string `json:"brief"`
	Ready bool `json:"ready"`
	LastTurn int `json:"last_turn"`
	Window int `json:"window"`
	Recall int `json:"recall,omitempty"`
	Pending *pending `json:"pending,omitempty"`
}

type historyTurn struct {
	Turn int `json:"turn"`
	Phase string `json:"phase"`
	Input string `json:"input"`
	Reply string `json:"reply"`
	Error string `json:"error,omitempty"`
	Files []string `json:"files,omitempty"`
}

func saveRecord(store *session.Store, r *record) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil { return err }
	return store.WriteArtifact(r.ID, metadataFile, append(data, '\n'))
}

func loadRecord(store *session.Store, workspace, kind, id string) (*record, session.Status, error) {
	s, err := store.Load(id)
	if err != nil { return nil, s, err }
	if err := session.RequireRunnable(s); err != nil { return nil, s, err }
	if err := rejectLinks(workspace, filepath.Join(store.Root, "sessions", id)); err != nil { return nil, s, err }
    if err := rejectLinks(workspace, filepath.Join(store.Root, "sessions", id, "artifacts", metadataFile)); err != nil { return nil, s, err }
	text, err := store.ReadArtifact(id, metadataFile)
	if err != nil { return nil, s, fmt.Errorf("not an auxiliary session: %w", err) }
	var r record
	if err := json.Unmarshal([]byte(text), &r); err != nil { return nil, s, err }
	if r.Version != 1 || r.ID != id || r.Kind != kind || r.Workspace != workspace || r.Window < 0 || r.Window > 6 || r.LastTurn < 0 || r.SourceFile == "" || filepath.Base(r.SourceFile) != r.SourceFile {
		return nil, s, errors.New("auxiliary identity, workspace or version mismatch")
	}
	expected := filepath.Join(store.Root, "sessions", id, "artifacts", "sandbox")
    if err := rejectLinks(workspace, expected); err != nil { return nil, s, err }
	if s.Workspace.Path != expected { return nil, s, errors.New("auxiliary model workspace mismatch") }
	return &r, s, nil
}

func createRecord(store *session.Store, workspace, kind string, o options) (*record, error) {
	skill, err := loadSkill(workspace, kind, o.flavor)
	if err != nil { return nil, err }
	sources, err := loadSources(workspace, o.sources)
	if err != nil { return nil, err }
	cfg, err := ai.ConfigFor(o.provider, o.model)
	if err != nil { return nil, err }
	if cfg.Name == "" || cfg.Name == "auto" { cfg.Name = "codex" }
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil { return nil, err }
	id := kind + "-" + time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(suffix[:])
	title := o.topic
	if title == "" { title = "New " + kind }
	titleRunes := []rune(title); if len(titleRunes) > 80 { title = string(titleRunes[:80]) }
	r := &record{Version: 1, ID: id, Kind: kind, Workspace: workspace, Title: title, Goal: o.topic,
		SourceFile: "aux-sources.md", Flavor: o.flavor, SkillDigest: digest(skill), SourceDigest: digest(sources), Window: 6}
	sandbox := filepath.Join(store.Root, "sessions", id, "artifacts", "sandbox")
	if _, err := store.Create(id, title, sandbox, cfg.Name, cfg.Model, "Independent auxiliary conversation. No Issue, FD or Git operations."); err != nil { return nil, err }
	if err := os.Mkdir(sandbox, 0o700); err != nil { return nil, err }
	for name, content := range map[string]string{"aux-skill.md": skill, "aux-sources.md": sources} {
		if err := store.WriteArtifact(id, name, []byte(content)); err != nil { return nil, err }
	}
	if err := saveRecord(store, r); err != nil { return nil, err }
	return r, nil
}

func records(store *session.Store, workspace, kind string) ([]record, error) {
	entries, err := os.ReadDir(filepath.Join(store.Root, "sessions"))
	if errors.Is(err, os.ErrNotExist) { return nil, nil }
	if err != nil { return nil, err }
	var out []record
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), kind+"-") { continue }
		r, _, err := loadRecord(store, workspace, kind, entry.Name())
		if err != nil { return nil, fmt.Errorf("inspect %s: %w", entry.Name(), err) }
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

// This lock differs from the Store.Update lock. Unknown locks are never stolen.
func acquire(store *session.Store, id string) (func(), error) {
	path := filepath.Join(store.Root, "sessions", id, "auxiliary.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil { return nil, fmt.Errorf("auxiliary session locked; preserve %s and verify its owner stopped before manual recovery: %w", path, err) }
	if _, err := fmt.Fprintf(f, "pid=%d\n", os.Getpid()); err != nil { _ = f.Close(); _ = os.Remove(path); return nil, err }
	if err := f.Close(); err != nil { _ = os.Remove(path); return nil, err }
	return func() { _ = os.Remove(path) }, nil
}

func consistent(r *record, s session.Status) error {
	if r.Pending != nil || r.LastTurn != s.Session.LastTurn || s.Session.State == session.StateRunning {
		return errors.New("unfinished auxiliary commit or unknown running turn; use recover <id>, never replay the model automatically")
	}
	return nil
}

func writeImmutable(store *session.Store, id, name string, data []byte) error {
	prior, err := store.ReadArtifact(id, name)
	if err == nil {
		if !bytes.Equal([]byte(prior), data) { return fmt.Errorf("existing auxiliary artifact conflicts: %s", name) }
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) { return err }
	return store.WriteArtifact(id, name, data)
}

func historyName(turn int) string { return fmt.Sprintf("aux-turn-%04d.json", turn) }

func readHistory(store *session.Store, id string, turn int) (historyTurn, error) {
	var h historyTurn
	text, err := store.ReadArtifact(id, historyName(turn))
	if err != nil { return h, err }
	if err := json.Unmarshal([]byte(text), &h); err != nil { return h, err }
	if h.Turn != turn { return h, errors.New("auxiliary history turn mismatch") }
	return h, nil
}
