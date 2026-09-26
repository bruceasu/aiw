package workflowadapter

import (
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/task"
	"aiw/internal/taskpath"
)

// SafeID validates the identifier accepted by Workflow operations.
func SafeID(id string) bool { return taskpath.ValidateID(id) == nil }

// ResolveWorkspaceKind keeps legacy Task metadata interpretation in the
// Task-owned adapter rather than in Workflow CLI code.
func ResolveWorkspaceKind(meta task.TaskMeta) string { return task.ResolvedWorkspaceKind(meta) }

// VerifiedTaskWorktree confirms that an isolated Task binding points to a
// registered and existing Git worktree.
func VerifiedTaskWorktree(meta task.TaskMeta) bool {
	worktree := strings.TrimSpace(meta.Worktree)
	if worktree == "" || worktree == "." {
		return false
	}
	if !filepath.IsAbs(worktree) {
		root, err := gitx.PrimaryWorktree()
		if err != nil {
			return false
		}
		worktree = filepath.Join(root, filepath.FromSlash(worktree))
	}
	return gitx.WorktreeRegistered(worktree) && fsx.Exists(worktree)
}
