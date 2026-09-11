package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/gitx"
	"aiw/internal/taskx"
)

type creationPreflight struct {
	TargetOverlaps      []string
	SharedWriteOverlaps []string
	UnrelatedDirtyPaths []string
}

func preflightTaskCreation(id string) (creationPreflight, error) {
	dirtyPaths, err := gitx.DirtyPaths()
	if err != nil {
		return creationPreflight{}, err
	}
	return classifyCreationDirtyPaths(dirtyPaths, []string{taskx.TaskDir(id), filepath.Join(taskx.RuntimeTasksDir, id)}, taskCreationSharedWritePaths()), nil
}

func authorizeTaskCreation(id string, allowUnrelatedDirty bool) error {
	primary, primaryPath, err := gitx.IsPrimaryWorktree()
	if err != nil {
		return err
	}
	if !primary {
		return fmt.Errorf("ordinary Tasks must be created from the primary workspace: %s", primaryPath)
	}

	preflight, err := preflightTaskCreation(id)
	if err != nil {
		return err
	}
	if err := preflight.authorize(allowUnrelatedDirty); err != nil {
		return err
	}
	if len(preflight.UnrelatedDirtyPaths) > 0 {
		fmt.Fprintf(os.Stderr, "allowing Task creation with unrelated dirty paths: %s\n", strings.Join(preflight.UnrelatedDirtyPaths, ", "))
	}
	return nil
}

func taskCreationSharedWritePaths() []string {
	return nil
}

func classifyCreationDirtyPaths(dirtyPaths, targetPaths, sharedWritePaths []string) creationPreflight {
	result := creationPreflight{}
	for _, dirtyPath := range dirtyPaths {
		switch {
		case overlapsAnyPath(dirtyPath, targetPaths):
			result.TargetOverlaps = append(result.TargetOverlaps, dirtyPath)
		case matchesAnyPath(dirtyPath, sharedWritePaths):
			result.SharedWriteOverlaps = append(result.SharedWriteOverlaps, dirtyPath)
		default:
			result.UnrelatedDirtyPaths = append(result.UnrelatedDirtyPaths, dirtyPath)
		}
	}
	return result
}

func (result creationPreflight) blockingError() error {
	if len(result.TargetOverlaps) > 0 {
		return fmt.Errorf("working tree has dirty target paths: %s", strings.Join(result.TargetOverlaps, ", "))
	}
	if len(result.SharedWriteOverlaps) > 0 {
		return fmt.Errorf("working tree has dirty shared write paths: %s", strings.Join(result.SharedWriteOverlaps, ", "))
	}
	return nil
}

func (result creationPreflight) authorize(allowUnrelatedDirty bool) error {
	if err := result.blockingError(); err != nil {
		return err
	}
	if len(result.UnrelatedDirtyPaths) > 0 && !allowUnrelatedDirty {
		return fmt.Errorf("working tree has unrelated uncommitted changes: %s; rerun with --allow-unrelated-dirty", strings.Join(result.UnrelatedDirtyPaths, ", "))
	}
	return nil
}

func overlapsAnyPath(path string, candidates []string) bool {
	for _, candidate := range candidates {
		if pathWithin(path, candidate) {
			return true
		}
	}
	return false
}

func matchesAnyPath(path string, candidates []string) bool {
	path = normalizeCreationPath(path)
	for _, candidate := range candidates {
		if path == normalizeCreationPath(candidate) {
			return true
		}
	}
	return false
}

func pathWithin(path, directory string) bool {
	path = normalizeCreationPath(path)
	directory = normalizeCreationPath(directory)
	if path == directory {
		return true
	}
	relative, err := filepath.Rel(directory, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func normalizeCreationPath(path string) string {
	return filepath.Clean(filepath.FromSlash(path))
}
