package assist

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"aiw/internal/session"
)

const (
	maxInput = 8 * 1024
	maxFile = 32 * 1024
	maxSources = 128 * 1024
	maxSkill = 48 * 1024
	maxMemory = 32 * 1024
	maxResponse = 512 * 1024
)

func digest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// Reject symlinks/junctions and traversal, including intermediate components.
func rejectLinks(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return errors.New("path is outside the current workspace")
	}
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "." { continue }
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) { continue }
		if err != nil { return err }
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil { return err }
		if info.Mode()&os.ModeSymlink != 0 || !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(current)) {
			return fmt.Errorf("linked path is not permitted: %s", current)
		}
	}
	return nil
}

func readText(path string, limit int) (string, error) {
	info, err := os.Lstat(path)
	if err != nil { return "", err }
	if !info.Mode().IsRegular() || info.Size() > int64(limit) { return "", fmt.Errorf("not a bounded regular text file: %s", path) }
	data, err := os.ReadFile(path)
	if err != nil { return "", err }
	if len(data) > limit || !utf8.Valid(data) || strings.ContainsRune(string(data), 0) { return "", fmt.Errorf("invalid or oversized UTF-8 text: %s", path) }
	return string(data), nil
}

func excluded(name string) bool {
	n := strings.ToLower(name)
	switch n {
	case ".git", ".ai", ".wt", ".ssh", ".aws", "node_modules", "dist", "bin", "aiw.toml", ".aiw.toml", ".npmrc", ".netrc": return true
	}
	return strings.HasPrefix(n, ".env") || strings.Contains(n, "credential") || strings.Contains(n, "secret") || strings.HasSuffix(n, ".pem") || strings.HasSuffix(n, ".key")
}

func loadSources(workspace string, paths []string) (string, error) {
	var out strings.Builder
	seen := map[string]bool{}
	for _, raw := range paths {
		path := raw
		if !filepath.IsAbs(path) { path = filepath.Join(workspace, path) }
		path = filepath.Clean(path)
		if err := rejectLinks(workspace, path); err != nil { return "", err }
		rel, _ := filepath.Rel(workspace, path)
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			if excluded(part) { return "", fmt.Errorf("excluded source: %s", raw) }
		}
		err := filepath.WalkDir(path, func(file string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil { return walkErr }
			if entry.Type()&os.ModeSymlink != 0 { return fmt.Errorf("linked source rejected: %s", file) }
			if excluded(entry.Name()) {
				if entry.IsDir() { return filepath.SkipDir }
				return nil
			}
			if err := rejectLinks(workspace, file); err != nil { return err }
			if entry.IsDir() { return nil }
			ext := strings.ToLower(filepath.Ext(file))
			if !strings.Contains("|.md|.txt|.go|.py|.js|.ts|.tsx|.jsx|.html|.css|.json|.toml|.yaml|.yml|.java|.xml|.sql|", "|"+ext+"|") || ext == "" { return nil }
			key := strings.ToLower(file)
			if seen[key] { return nil }
			if len(seen) >= 64 { return errors.New("source limit is 64 files; choose a narrower source") }
			text, err := readText(file, maxFile)
			if err != nil { return err }
			name, _ := filepath.Rel(workspace, file)
			chunk := fmt.Sprintf("\n--- SOURCE %s sha256=%s ---\n%s\n--- END SOURCE ---\n", filepath.ToSlash(name), digest(text), text)
			if out.Len()+len(chunk) > maxSources { return errors.New("sources exceed 128 KiB; choose a narrower source") }
			seen[key] = true
			out.WriteString(chunk)
			return nil
		})
		if err != nil { return "", err }
	}
	if len(paths) > 0 && len(seen) == 0 { return "", errors.New("no supported text sources selected") }
	return out.String(), nil
}

func loadSkill(workspace, kind, flavor string) (string, error) {
	var roots []string
	if configured := os.Getenv("AIW_SKILLS_SOURCE_ROOT"); configured != "" {
		roots = []string{configured}
	} else {
		roots = append(roots, filepath.Join(workspace, "src", "skills"))
		if exe, err := os.Executable(); err == nil { roots = append(roots, filepath.Join(filepath.Dir(exe), "skills")) }
		roots = append(roots, filepath.Join(workspace, ".agents", "skills"))
	}
	for _, root := range roots {
		absolute, err := filepath.Abs(root)
		if err != nil { return "", err }
		root, err = filepath.EvalSymlinks(absolute)
		if errors.Is(err, os.ErrNotExist) { continue }
		if err != nil { return "", err }
		dir := filepath.Join(root, kind)
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); errors.Is(err, os.ErrNotExist) { continue } else if err != nil { return "", err }
		names := []string{"SKILL.md", "references/auxiliary-cli.md"}
		if kind == "prototype" {
			if flavor == "ui" { names = append(names, "UI.md") } else { names = append(names, "LOGIC.md") }
		}
		var out strings.Builder
		for _, name := range names {
			path := filepath.Join(dir, name)
			if err := rejectLinks(root, path); err != nil { return "", err }
			text, err := readText(path, maxSkill)
			if err != nil { return "", err }
			out.WriteString(fmt.Sprintf("\n--- SKILL %s/%s ---\n%s\n", dir, name, text))
		}
		if out.Len() > maxSkill { return "", errors.New("selected Skill exceeds 48 KiB") }
		return out.String(), nil
	}
	return "", fmt.Errorf("Skill %s unavailable; install the Skill resources or set AIW_SKILLS_SOURCE_ROOT", kind)
}

func memory(store *session.Store, r *record) (string, error) {
	var out strings.Builder
	fmt.Fprintf(&out, "Goal: %s\nCurrent brief/progress: %s\nHistory window: last %d turns, full records remain local.\n", r.Goal, r.Brief, r.Window)
	start := r.LastTurn-r.Window+1
	if start < 1 { start = 1 }
	turns := []int{}
	if r.Recall > 0 && r.Recall < start { turns = append(turns, r.Recall) }
	for turn := start; turn <= r.LastTurn; turn++ { turns = append(turns, turn) }
	for _, turn := range turns {
		h, err := readHistory(store, r.ID, turn)
		if err != nil { return "", err }
		fmt.Fprintf(&out, "\nTurn %d [%s]\nUser: %s\nAssistant: %s\nError: %s\nFiles: %v\n", turn, h.Phase, h.Input, h.Reply, h.Error, h.Files)
	}
	if out.Len() > maxMemory { return "", errors.New("course context exceeds 32 KiB; reduce /window N or clear /recall 0; full history is retained") }
	return out.String(), nil
}
