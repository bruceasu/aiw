package taskx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aiw/internal/workflow"
)

// TaskLocation separates storage lifecycle from Workflow execution state.
// Problems prohibit repair; callers can still display other discovered Tasks.
type TaskLocation struct {
	ID          string
	RuntimeDir  string
	ChangeDir   string
	ArchiveName string
	Archived    bool
	Problems    []error
}

func validLocationID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

func archiveLocationID(name string) (string, bool) {
	if len(name) < 12 || name[10] != '-' {
		return "", false
	}
	if _, err := time.Parse("2006-01-02", name[:10]); err != nil {
		return "", false
	}
	id := name[11:]
	return id, validLocationID(id)
}

// MetadataPathInDirectory preserves canonical filename precedence and reports
// unreadable or non-regular candidates instead of treating them as missing.
func MetadataPathInDirectory(dir string) (string, error) {
	for _, name := range []string{TaskMetaFile, LegacyTaskMetaFile} {
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("task metadata %s: not a regular file", path)
		}
		return path, nil
	}
	return "", nil
}

// DiscoverTaskLocations scans only direct children of the supported roots.
// It does not repair, migrate, or follow Session and evidence subdirectories.
func DiscoverTaskLocations() ([]TaskLocation, error) {
	tasks := map[string]*TaskLocation{}
	get := func(id string) *TaskLocation {
		if tasks[id] == nil {
			tasks[id] = &TaskLocation{ID: id}
		}
		return tasks[id]
	}
	roots := []struct {
		path     string
		runtime  bool
		archived bool
		legacy   bool
	}{
		{RuntimeTasksPath(), true, false, false},
		{filepath.Join(RuntimeRoot(), LegacyRuntimeTasksDir), true, false, true},
		{filepath.Join(RuntimeRoot(), ".ai", "archive"), true, true, false},
		{ChangesDir, false, false, false},
		{ArchiveDir, false, true, false},
		{LegacyArchiveDir, false, true, false},
	}
	var scanErrors []error
	for _, root := range roots {
		entries, err := os.ReadDir(root.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("discover %s: %w", root.path, err))
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "archive" {
				continue
			}
			if root.legacy && isRuntimeRootDirectory(entry.Name()) { continue }
			id := entry.Name()
			archiveName := ""
			if root.archived {
				var ok bool
				id, ok = archiveLocationID(entry.Name())
				if !ok {
					// Old standalone Session archives have no dated Task identity.
					continue
				}
				archiveName = entry.Name()
			}
			if !validLocationID(id) {
				continue
			}
			dir := filepath.Join(root.path, entry.Name())
			if root.runtime {
				metaPath, metaErr := MetadataPathInDirectory(dir)
				stateInfo, stateErr := os.Lstat(filepath.Join(dir, "state.json"))
				_, markerErr := os.Lstat(filepath.Join(dir, "migrated-to"))
				if !root.legacy && metaPath == "" && metaErr == nil && errors.Is(stateErr, os.ErrNotExist) {
					if errors.Is(markerErr, os.ErrNotExist) { continue }
				}
				location := get(id)
				if markerErr == nil { location.Problems = append(location.Problems, fmt.Errorf("migration marker in %s requires a one-time manual path adjustment after this Task is complete", dir)) }
				if markerErr != nil && !errors.Is(markerErr, os.ErrNotExist) { location.Problems = append(location.Problems, markerErr) }
				if root.legacy {
					location.Problems = append(location.Problems, fmt.Errorf("Task %s remains at legacy path %s; manually relocate the active Task after implementation is complete", id, dir))
				}
				if metaErr != nil {
					location.Problems = append(location.Problems, metaErr)
				}
				if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
					location.Problems = append(location.Problems, stateErr)
				} else if stateErr == nil && !stateInfo.Mode().IsRegular() {
					location.Problems = append(location.Problems, fmt.Errorf("state in %s: not a regular file", dir))
				}
				if location.RuntimeDir != "" {
					location.Problems = append(location.Problems, fmt.Errorf("duplicate runtime records: %s and %s", location.RuntimeDir, dir))
				} else {
					location.RuntimeDir = dir
				}
			} else {
				location := get(id)
				if location.ChangeDir != "" {
					location.Problems = append(location.Problems, fmt.Errorf("conflicting changes: %s and %s", location.ChangeDir, dir))
				} else {
					location.ChangeDir = dir
				}
			}
			location := get(id)
			if root.archived {
				if location.ArchiveName != "" && location.ArchiveName != archiveName {
					location.Problems = append(location.Problems, fmt.Errorf("archive names disagree: %s and %s", location.ArchiveName, archiveName))
				}
				location.Archived = true
				location.ArchiveName = archiveName
			}
		}
	}
	result := make([]TaskLocation, 0, len(tasks))
	for _, location := range tasks {
		if location.Archived && location.ChangeDir != "" && filepath.Dir(location.ChangeDir) == filepath.Clean(ChangesDir) {
			location.Problems = append(location.Problems, fmt.Errorf("active change conflicts with archived runtime: %s", location.ID))
		}
		if location.RuntimeDir == "" {
			location.RuntimeDir = filepath.Join(RuntimeTasksPath(), location.ID)
			if location.Archived {
				location.RuntimeDir = filepath.Join(RuntimeRoot(), ".ai", "archive", location.ArchiveName)
			} else {
				// Preserve an existing legacy directory even when its metadata
				// and state are gone but other runtime artifacts remain.
				info, err := os.Lstat(location.RuntimeDir)
				if errors.Is(err, os.ErrNotExist) {
					legacy := filepath.Join(RuntimeRoot(), LegacyRuntimeTasksDir, location.ID)
					legacyInfo, legacyErr := os.Lstat(legacy)
					if legacyErr == nil && legacyInfo.IsDir() {
						location.RuntimeDir = legacy
					} else if legacyErr != nil && !errors.Is(legacyErr, os.ErrNotExist) {
						location.Problems = append(location.Problems, legacyErr)
					} else if legacyErr == nil {
						location.Problems = append(location.Problems, fmt.Errorf("runtime path is not a directory: %s", legacy))
					}
				} else if err != nil {
					location.Problems = append(location.Problems, err)
				} else if !info.IsDir() {
					location.Problems = append(location.Problems, fmt.Errorf("runtime path is not a directory: %s", location.RuntimeDir))
				}
			}
		}
		result = append(result, *location)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, errors.Join(scanErrors...)
}

func isRuntimeRootDirectory(name string) bool {
	switch name {
	case "tasks", "archive", "sessions", "locks", "migrations", "requirements", "issue", "tmp", "compile-cache":
		return true
	default:
		return false
	}
}

func readLocationMeta(path, id string) (TaskMeta, error) {
	meta, err := ReadTaskMeta(path)
	if err != nil {
		return meta, err
	}
	if strings.TrimSpace(meta.ID) == "" {
		return meta, fmt.Errorf("invalid task metadata %s: missing task id", path)
	}
	if !validLocationID(meta.ID) || meta.ID != id {
		return meta, fmt.Errorf("invalid task metadata %s: task id %q does not match directory id %q", path, meta.ID, id)
	}
	return meta, nil
}

// EnsureRuntime preserves every existing record and reconstructs only missing
// metadata/state. The caller must filter archived Tasks before invoking it.
func (location TaskLocation) EnsureRuntime() (workflow.RuntimeState, bool, error) {
	if err := errors.Join(location.Problems...); err != nil {
		return workflow.RuntimeState{}, false, err
	}
	info, dirErr := os.Lstat(location.RuntimeDir)
	if dirErr != nil && !errors.Is(dirErr, os.ErrNotExist) {
		return workflow.RuntimeState{}, false, dirErr
	}
	if dirErr == nil && !info.IsDir() {
		return workflow.RuntimeState{}, false, fmt.Errorf("runtime path is not a directory: %s", location.RuntimeDir)
	}
	path, err := MetadataPathInDirectory(location.RuntimeDir)
	if err != nil {
		return workflow.RuntimeState{}, false, err
	}
	meta := TaskMeta{ID: location.ID, Type: "task", Status: "DRAFT", WorkspaceKind: "unassigned"}
	if path != "" {
		meta, err = readLocationMeta(path, location.ID)
		if err != nil {
			return workflow.RuntimeState{}, false, err
		}
	}
	state, stateErr := workflow.LoadFromDirectory(location.RuntimeDir, workflow.TaskID(location.ID))
	if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
		return state, false, stateErr
	}
	if path == "" && stateErr != nil && location.ChangeDir == "" {
		return state, false, fmt.Errorf("no identity evidence for task %s", location.ID)
	}
	if path == "" && stateErr == nil {
		meta.Status = string(workflow.DeriveSummary(state).Status)
		meta.Worktree = state.Task.Workspace
		meta.WorkspaceKind = string(state.Task.Kind)
		meta.Delivery = string(state.Delivery)
	}
	rebuilt := false
	if path == "" {
		path = filepath.Join(location.RuntimeDir, TaskMetaFile)
		err = CreateTaskMeta(path, meta)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return state, false, err
		}
		rebuilt = err == nil
		// Read again if a concurrent creator supplied the metadata.
		meta, err = readLocationMeta(path, location.ID)
		if err != nil {
			return state, rebuilt, err
		}
	}
	if stateErr != nil {
		var created bool
		state, created, err = workflow.EnsureCompatibleInDirectory(location.RuntimeDir, WorkflowRuntimeFromMeta(meta))
		rebuilt = rebuilt || created
	}
	return state, rebuilt, err
}
