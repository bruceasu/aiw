package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrSessionNotFound = fmt.Errorf("session record not found: %w", os.ErrNotExist)

// Location describes storage without changing the historical Session state.
type Location struct {
	Dir string
	Archived bool
	TaskID string
	Status Status
}

func validSessionID(id string) bool {
	return strings.TrimSpace(id) != "" && id != "." && id != ".." && id != "archive" && !strings.ContainsAny(id, `/\\:`)
}

// Resolve scans only the active, legacy and paired archive locations.
// Missing status in an existing directory is corruption, not a missing Session.
func (s *Store) Resolve(id string) (Location, error) {
	if !validSessionID(id) { return Location{}, errors.New("invalid session id") }
	candidates := []Location{{Dir: s.sessionDir(id)}, {Dir: filepath.Join(s.Root, "archive", id), Archived: true}}
	root := filepath.Join(s.Root, "sessions", "archive")
	entries, err := os.ReadDir(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) { return Location{}, err }
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || len(name) < 12 || name[10] != '-' { continue }
		if _, err := time.Parse("2006-01-02", name[:10]); err != nil { continue }
		candidates = append(candidates, Location{Dir: filepath.Join(root, name, id), Archived: true, TaskID: name[11:]})
	}
	var found *Location
	for _, candidate := range candidates {
		info, err := os.Lstat(candidate.Dir)
		if errors.Is(err, os.ErrNotExist) { continue }
		if err != nil { return Location{}, err }
		if !info.IsDir() { return Location{}, fmt.Errorf("session path is not a directory: %s", candidate.Dir) }
		statusPath := filepath.Join(candidate.Dir, "status.json")
		statusInfo, err := os.Lstat(statusPath)
		if err != nil { return Location{}, fmt.Errorf("read session at %s: %w", candidate.Dir, err) }
		if !statusInfo.Mode().IsRegular() { return Location{}, fmt.Errorf("session status is not a regular file: %s", statusPath) }
		data, err := os.ReadFile(statusPath)
		if err != nil { return Location{}, fmt.Errorf("read session at %s: %w", candidate.Dir, err) }
		if err := json.Unmarshal(data, &candidate.Status); err != nil { return Location{}, fmt.Errorf("decode session at %s: %w", candidate.Dir, err) }
		if candidate.Status.Session.ID != id { return Location{}, fmt.Errorf("session identity conflict at %s", candidate.Dir) }
		if candidate.TaskID != "" && candidate.Status.Task != nil && candidate.Status.Task.TaskID != "" && candidate.Status.Task.TaskID != candidate.TaskID {
			return Location{}, fmt.Errorf("session Task identity conflict at %s", candidate.Dir)
		}
		if found != nil { return Location{}, fmt.Errorf("duplicate session records: %s and %s", found.Dir, candidate.Dir) }
		candidate.Status.archived = candidate.Archived
		found = &candidate
	}
	if found == nil { return Location{}, ErrSessionNotFound }
	return *found, nil
}

func (s *Store) requireWritable(id string) error {
	location, err := s.Resolve(id)
	if errors.Is(err, ErrSessionNotFound) { return nil }
	if err != nil { return err }
	if location.Archived { return fmt.Errorf("archived session is read-only: %s", location.Dir) }
	return nil
}

func (s *Store) readPath(id, name string) (string, error) {
	location, err := s.Resolve(id)
	if err != nil { return "", err }
	if filepath.IsAbs(name) {
		matched := false
		for _, base := range []string{s.sessionDir(id), filepath.Join(s.Root, "archive", id), location.Dir} {
			absolute, err := filepath.Abs(base)
			if err != nil { return "", err }
			relative, err := filepath.Rel(absolute, name)
			if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				name, matched = relative, true
				break
			}
		}
		if !matched { return "", errors.New("session read path is outside its verified locations") }
	}
	name = filepath.Clean(name)
	if name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) { return "", errors.New("invalid session read path") }
	return filepath.Join(location.Dir, name), nil
}
