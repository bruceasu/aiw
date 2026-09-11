package task

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"aiw/internal/gitx"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// preflightForceCloseDelivery separates the non-destructive discarded path
// from the merge path. A discard still refuses an ambiguous isolated resource,
// while a merge additionally requires the parent and Task branches to be safe
// delivery targets.
func preflightForceCloseDelivery(meta taskx.TaskMeta, delivery workflow.DeliveryState) error {
	if delivery == workflow.DeliveryDiscarded {
		if resolvedWorkspaceKind(meta) == "isolated" && !verifiedTaskWorktree(meta) {
			return fmt.Errorf("discarded force-close cannot identify the isolated Task worktree")
		}
		if resolvedWorkspaceKind(meta) == "unknown" {
			return fmt.Errorf("discarded force-close requires a known Task workspace binding")
		}
		return nil
	}
	return preflightMergedForceClose(meta)
}

// preflightMergedForceClose verifies the Git facts required before a later
// delivery adapter may merge an isolated Task branch. It deliberately has no
// side effects so a failed preflight leaves every Task resource recoverable.
func preflightMergedForceClose(meta taskx.TaskMeta) error {
	if resolvedWorkspaceKind(meta) != "isolated" || !verifiedTaskWorktree(meta) {
		return fmt.Errorf("merged force-close requires a verified isolated Task worktree")
	}
	branch := strings.TrimSpace(meta.Branch)
	parent := strings.TrimSpace(meta.ParentBranch)
	if branch == "" || parent == "" || branch == parent {
		return fmt.Errorf("merged force-close requires distinct branch and parent_branch metadata")
	}

	primary, err := gitx.PrimaryWorktree()
	if err != nil {
		return fmt.Errorf("resolve primary worktree: %w", err)
	}
	worktree := strings.TrimSpace(meta.Worktree)
	if !filepath.IsAbs(worktree) {
		worktree = filepath.Join(primary, filepath.FromSlash(worktree))
	}
	if err := requireCleanWorktree("Task worktree", worktree); err != nil {
		return err
	}
	if err := requireBranch("Task worktree", worktree, branch); err != nil {
		return err
	}
	if err := requireCleanWorktree("parent worktree", primary); err != nil {
		return err
	}
	if err := requireBranch("parent worktree", primary, parent); err != nil {
		return err
	}
	if err := requireBranchRef(primary, branch); err != nil {
		return err
	}
	if err := requireBranchRef(primary, parent); err != nil {
		return err
	}

	output, err := gitOutput(primary, "rev-list", "--count", parent+".."+branch)
	if err != nil {
		return fmt.Errorf("verify committed Task content: %w", err)
	}
	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil || count < 1 {
		return fmt.Errorf("merged force-close requires committed Task-branch content not present on %s", parent)
	}
	return nil
}

func requireCleanWorktree(label, dir string) error {
	output, err := gitOutput(dir, "status", "--porcelain=v1")
	if err != nil {
		return fmt.Errorf("inspect %s cleanliness: %w", label, err)
	}
	if strings.TrimSpace(output) != "" {
		return fmt.Errorf("%s must be clean before merged force-close", label)
	}
	return nil
}

func requireBranch(label, dir, want string) error {
	got, err := gitOutput(dir, "branch", "--show-current")
	if err != nil {
		return fmt.Errorf("inspect %s branch: %w", label, err)
	}
	if strings.TrimSpace(got) != want {
		return fmt.Errorf("%s branch %q does not match required branch %q", label, strings.TrimSpace(got), want)
	}
	return nil
}

func requireBranchRef(dir, branch string) error {
	if _, err := gitOutput(dir, "rev-parse", "--verify", "refs/heads/"+branch); err != nil {
		return fmt.Errorf("required branch %q is unavailable: %w", branch, err)
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
