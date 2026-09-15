package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SkillSource identifies the root that supplied a discovered Skill. Earlier
// sources win, so a repository Skill cannot be silently replaced by a user or
// legacy Skill with the same name.
type SkillSource string

const (
	SkillSourceRepository   SkillSource = "repository"
	SkillSourceInstallation SkillSource = "installation"
	SkillSourceUser         SkillSource = "user"
	SkillSourceLegacy       SkillSource = "legacy"
)

// SkillRegistry discovers metadata only. It neither executes a Skill nor
// grants an Actor any authority.
type SkillRegistry struct {
	RepositoryRoot   string
	InstallationRoot string
	UserRoots        []string
	LegacyRoots      []string
}

// SkillProvenance makes the selected file independently auditable.
type SkillProvenance struct {
	Source SkillSource `json:"source"`
	Root   string      `json:"root"`
	Path   string      `json:"path"`
}

// SkillSnapshot is the immutable, content-addressed representation provided
// to an Actor. Version is optional because SKILL.md front matter is optional.
type SkillSnapshot struct {
	Name       string          `json:"name"`
	Version    string          `json:"version,omitempty"`
	Hash       string          `json:"hash"`
	Provenance SkillProvenance `json:"provenance"`
}

// SkillManifest records role routing and all selected Skill snapshots. Digest
// excludes ResolvedAt so the same routing is reproducible across retries.
type SkillManifest struct {
	Routing    map[string]string `json:"routing"`
	Skills     []SkillSnapshot   `json:"skills"`
	Digest     string            `json:"digest"`
	ResolvedAt string            `json:"resolved_at"`
}

// Discover returns the first valid Skill of each name in the documented
// precedence order: repository, AIW installation, user, then legacy fallback.
func (r SkillRegistry) Discover() ([]SkillSnapshot, error) {
	type root struct { source SkillSource; path string }
	roots := []root{}
	if strings.TrimSpace(r.RepositoryRoot) != "" {
		roots = append(roots, root{SkillSourceRepository, filepath.Join(r.RepositoryRoot, ".agents", "skills")}, root{SkillSourceRepository, filepath.Join(r.RepositoryRoot, ".codex", "skills")})
	}
	if strings.TrimSpace(r.InstallationRoot) != "" {
		roots = append(roots, root{SkillSourceInstallation, r.InstallationRoot}, root{SkillSourceInstallation, filepath.Join(r.InstallationRoot, "skills")})
	}
	for _, path := range r.UserRoots { roots = append(roots, root{SkillSourceUser, path}) }
	for _, path := range r.LegacyRoots { roots = append(roots, root{SkillSourceLegacy, path}) }

	seen := map[string]bool{}
	var discovered []SkillSnapshot
	for _, candidate := range roots {
		if strings.TrimSpace(candidate.path) == "" { continue }
		files, err := skillFiles(candidate.path)
		if errors.Is(err, os.ErrNotExist) { continue }
		if err != nil { return nil, fmt.Errorf("discover Skills in %s: %w", candidate.path, err) }
		for _, path := range files {
			snapshot, err := readSkill(candidate.source, candidate.path, path)
			if err != nil { return nil, err }
			key := strings.ToLower(snapshot.Name)
			if seen[key] { continue }
			seen[key] = true
			discovered = append(discovered, snapshot)
		}
	}
	sort.Slice(discovered, func(i, j int) bool { return strings.ToLower(discovered[i].Name) < strings.ToLower(discovered[j].Name) })
	return discovered, nil
}

// Resolve chooses the requested named Skill for each Actor role and returns a
// content-addressed manifest suitable for persistence with its request.
func (r SkillRegistry) Resolve(routing map[string]string) (SkillManifest, error) {
	discovered, err := r.Discover()
	if err != nil { return SkillManifest{}, err }
	byName := make(map[string]SkillSnapshot, len(discovered))
	for _, skill := range discovered { byName[strings.ToLower(skill.Name)] = skill }
	manifest := SkillManifest{Routing: map[string]string{}, ResolvedAt: time.Now().UTC().Format(time.RFC3339)}
	for role, name := range routing {
		role, name = strings.TrimSpace(role), strings.TrimSpace(name)
		if role == "" || name == "" { return SkillManifest{}, errors.New("Skill routing role and name are required") }
		skill, ok := byName[strings.ToLower(name)]
		if !ok { return SkillManifest{}, fmt.Errorf("Skill %q routed to %q was not discovered", name, role) }
		manifest.Routing[role] = skill.Name
		manifest.Skills = append(manifest.Skills, skill)
	}
	if err := manifest.setDigest(); err != nil { return SkillManifest{}, err }
	return manifest, nil
}

func (m SkillManifest) Validate() error {
	if len(m.Routing) == 0 || len(m.Skills) == 0 { return errors.New("Skill manifest routing and Skills are required") }
	if _, err := time.Parse(time.RFC3339, m.ResolvedAt); err != nil { return fmt.Errorf("Skill manifest resolved time: %w", err) }
	if len(m.Digest) != sha256.Size*2 { return errors.New("Skill manifest digest must be a SHA-256 hex value") }
	copy := m
	if err := copy.setDigest(); err != nil { return err }
	if copy.Digest != m.Digest { return errors.New("Skill manifest digest does not match manifest") }
	known := map[string]bool{}
	for _, skill := range m.Skills {
		if strings.TrimSpace(skill.Name) == "" || len(skill.Hash) != sha256.Size*2 || strings.TrimSpace(skill.Provenance.Path) == "" { return errors.New("Skill snapshot name, hash, and path are required") }
		known[strings.ToLower(skill.Name)] = true
	}
	for _, name := range m.Routing { if !known[strings.ToLower(name)] { return fmt.Errorf("routed Skill %q is absent from manifest", name) } }
	return nil
}

// ReadableSnapshot produces a human-readable, secret-free audit record.
func (m SkillManifest) ReadableSnapshot() string {
	var b strings.Builder
	b.WriteString("# Skill routing manifest\n\n")
	fmt.Fprintf(&b, "Digest: `%s`\nResolved: %s\n\n", m.Digest, m.ResolvedAt)
	roles := make([]string, 0, len(m.Routing))
	for role := range m.Routing { roles = append(roles, role) }
	sort.Strings(roles)
	for _, role := range roles { fmt.Fprintf(&b, "- %s: %s\n", role, m.Routing[role]) }
	b.WriteString("\n## Selected Skills\n\n")
	for _, skill := range m.Skills { fmt.Fprintf(&b, "- %s%s - %s, `%s`\n", skill.Name, versionSuffix(skill.Version), skill.Provenance.Source, skill.Hash) }
	return b.String()
}

func (m *SkillManifest) setDigest() error {
	sort.Slice(m.Skills, func(i, j int) bool { return strings.ToLower(m.Skills[i].Name) < strings.ToLower(m.Skills[j].Name) })
	type canonical struct { Routing [][2]string `json:"routing"`; Skills []SkillSnapshot `json:"skills"` }
	value := canonical{Skills: m.Skills}
	for role, name := range m.Routing { value.Routing = append(value.Routing, [2]string{role, name}) }
	sort.Slice(value.Routing, func(i, j int) bool { return value.Routing[i][0] < value.Routing[j][0] })
	b, err := json.Marshal(value)
	if err != nil { return err }
	sum := sha256.Sum256(b)
	m.Digest = hex.EncodeToString(sum[:])
	return nil
}

func skillFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil { return nil, err }
	if !info.IsDir() { return nil, fmt.Errorf("Skill root %s is not a directory", root) }
	var files []string
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil { return walkErr }
		if !entry.IsDir() && strings.EqualFold(entry.Name(), "SKILL.md") { files = append(files, path) }
		return nil
	})
	sort.Strings(files)
	return files, err
}

func readSkill(source SkillSource, root, path string) (SkillSnapshot, error) {
	content, err := os.ReadFile(path)
	if err != nil { return SkillSnapshot{}, fmt.Errorf("read Skill %s: %w", path, err) }
	name, version := skillFrontMatter(content)
	if name == "" { name = filepath.Base(filepath.Dir(path)) }
	sum := sha256.Sum256(content)
	return SkillSnapshot{Name: name, Version: version, Hash: hex.EncodeToString(sum[:]), Provenance: SkillProvenance{Source: source, Root: root, Path: path}}, nil
}

func skillFrontMatter(content []byte) (name, version string) {
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" { return "", "" }
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "---" { break }
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 { continue }
		key, value := strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		switch key { case "name": name = value; case "version": version = value }
	}
	return name, version
}

func versionSuffix(version string) string { if version == "" { return "" }; return " (" + version + ")" }
