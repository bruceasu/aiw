package task

import (
	"fmt"
	"path/filepath"
	"strings"

	"aiw/internal/gitx"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// WorkspaceCoordinator binds automated execution to one Task worktree and
// records the parent baseline before an Actor can receive a write lease.
// It deliberately treats parent movement as external drift until a caller has
// evidence that AIW performed a parent write.
type WorkspaceCoordinator struct {
	store *workflow.Store
}

func newWorkspaceCoordinator(store *workflow.Store) WorkspaceCoordinator {
	return WorkspaceCoordinator{store: store}
}

func (c WorkspaceCoordinator) Prepare(id string, meta taskx.TaskMeta) error {
	if resolvedWorkspaceKind(meta) != "isolated" || !verifiedTaskWorktree(meta) {
		return fmt.Errorf("Task %s requires a verified isolated worktree", id)
	}
	parent, err := gitx.PrimaryWorktree()
	if err != nil {
		return fmt.Errorf("resolve parent workspace: %w", err)
	}
	worktree := meta.Worktree
	if !filepath.IsAbs(worktree) {
		worktree = filepath.Join(parent, filepath.FromSlash(worktree))
	}
	worktree, err = filepath.Abs(worktree)
	if err != nil {
		return fmt.Errorf("resolve Task worktree: %w", err)
	}
	expected := filepath.Join(parent, taskx.WorktreeDir, id)
	if !strings.EqualFold(filepath.Clean(worktree), filepath.Clean(expected)) {
		return fmt.Errorf("Task %s worktree %s does not match managed path %s", id, worktree, expected)
	}
	branch, err := gitx.CurrentBranchAt(worktree)
	if err != nil {
		return fmt.Errorf("read Task worktree branch: %w", err)
	}
	if branch != meta.Branch {
		return fmt.Errorf("Task worktree branch %s does not match recorded branch %s", branch, meta.Branch)
	}
	parentBranch, err := gitx.CurrentBranchAt(parent)
	if err != nil {
		return fmt.Errorf("read parent branch: %w", err)
	}
	if parentBranch != meta.ParentBranch {
		return fmt.Errorf("parent branch %s does not match recorded parent %s", parentBranch, meta.ParentBranch)
	}
	commit, err := gitx.HeadAt(parent)
	if err != nil {
		return fmt.Errorf("read parent baseline: %w", err)
	}
	state, err := c.store.Load(workflow.TaskID(id))
	if err != nil {
		return err
	}
	if state.Workspace == nil {
		state, err = c.store.RecordWorkspaceBinding(workflow.TaskID(id), workflow.WorkspaceBinding{
			ParentPath: parent, ParentBranch: parentBranch, ParentCommit: commit,
			WorktreePath: worktree, TaskBranch: branch,
		})
		if err != nil {
			return err
		}
	}
	if state.Workspace.ParentPath != parent || state.Workspace.ParentBranch != parentBranch || state.Workspace.WorktreePath != worktree || state.Workspace.TaskBranch != branch {
		return fmt.Errorf("Task %s workspace binding conflicts with its recorded ancestry", id)
	}
	paths, err := gitx.DirtyPathsAt(parent)
	if err != nil {
		return err
	}
	if state.Workspace.ParentCommit != commit || len(paths) > 0 {
		_, err = c.store.RecordParentDrift(workflow.TaskID(id), commit, paths)
	}
	return err
}
