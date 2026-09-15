package task

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"aiw/internal/gitx"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// localMergeDelivery commits accepted Task changes, merges the isolated Task
// branch into its recorded parent, verifies the result, and only then removes
// the managed worktree and branch. It deliberately leaves a conflicting Task
// worktree intact as the candidate for resolving-merge-conflicts review.
func localMergeDelivery(id string, meta taskx.TaskMeta, store *workflow.Store, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("local-merge requires a commit message")
	}
	state, err := store.Load(workflow.TaskID(id))
	if err != nil {
		return err
	}
	summary := workflow.DeriveSummary(state)
	if summary.Execution != workflow.ExecutionCompleted {
		return fmt.Errorf("local-merge requires all Work Items to be completed")
	}
	if summary.Validation != workflow.ValidationPassed && summary.Validation != workflow.ValidationNotRequired && summary.Validation != workflow.ValidationWaived {
		return fmt.Errorf("local-merge requires completed validation, got %s", summary.Validation)
	}
	if summary.Delivery == workflow.DeliveryMerged {
		return fmt.Errorf("Task is already locally merged")
	}
	if summary.Delivery == workflow.DeliveryDiscarded {
		return fmt.Errorf("discarded Task cannot be locally merged")
	}
	if err := preflightLocalMerge(meta); err != nil {
		return recordLocalDeliveryFailure(store, id, "preflight", err)
	}

	primary, _ := gitx.PrimaryWorktree()
	taskWorktree := absoluteTaskWorktree(primary, meta.Worktree)
	if dirty, err := gitDirty(taskWorktree); err != nil {
		return recordLocalDeliveryFailure(store, id, "commit-preflight", err)
	} else if dirty {
		if err := gitRunAt(taskWorktree, "add", "--all"); err != nil {
			return recordLocalDeliveryFailure(store, id, "commit-stage", err)
		}
		if err := gitRunAt(taskWorktree, "commit", "-m", message); err != nil {
			return recordLocalDeliveryFailure(store, id, "commit", err)
		}
	}
	if err := requireTaskBranchCommit(primary, meta.ParentBranch, meta.Branch); err != nil {
		return recordLocalDeliveryFailure(store, id, "commit-verify", err)
	}

	if err := gitRunAt(primary, "merge", "--no-ff", meta.Branch, "-m", message); err != nil {
		return preserveConflictCandidate(store, id, primary, taskWorktree, meta, err)
	}
	if !isAncestorAt(primary, meta.Branch, meta.ParentBranch) {
		return recordLocalDeliveryFailure(store, id, "ancestry", fmt.Errorf("Task branch %s is not an ancestor of parent branch %s after merge", meta.Branch, meta.ParentBranch))
	}
	merged, err := store.SetDelivery(workflow.TaskID(id), workflow.DeliveryMerged)
	if err != nil {
		return err
	}
	if err := projectWorkflowState(id, merged); err != nil {
		return err
	}
	if err := gitRunAt(primary, "worktree", "remove", taskWorktree); err != nil {
		return recordLocalDeliveryFailure(store, id, "cleanup-worktree", err)
	}
	if err := gitRunAt(primary, "branch", "-d", meta.Branch); err != nil {
		return recordLocalDeliveryFailure(store, id, "cleanup-branch", err)
	}
	return nil
}

func preflightLocalMerge(meta taskx.TaskMeta) error {
	if resolvedWorkspaceKind(meta) != "isolated" || !verifiedTaskWorktree(meta) {
		return fmt.Errorf("local-merge requires a verified isolated Task worktree")
	}
	branch, parent := strings.TrimSpace(meta.Branch), strings.TrimSpace(meta.ParentBranch)
	if branch == "" || parent == "" || branch == parent {
		return fmt.Errorf("local-merge requires distinct branch and parent_branch metadata")
	}
	primary, err := gitx.PrimaryWorktree()
	if err != nil {
		return fmt.Errorf("resolve primary worktree: %w", err)
	}
	if err := requireCleanWorktree("parent worktree", primary); err != nil {
		return err
	}
	if err := requireBranch("parent worktree", primary, parent); err != nil {
		return err
	}
	if err := requireBranch("Task worktree", absoluteTaskWorktree(primary, meta.Worktree), branch); err != nil {
		return err
	}
	if err := requireBranchRef(primary, branch); err != nil {
		return err
	}
	return requireBranchRef(primary, parent)
}

func preserveConflictCandidate(store *workflow.Store, id, primary, taskWorktree string, meta taskx.TaskMeta, mergeErr error) error {
	// The parent merge has entered a conflict state. Abort only that parent
	// operation, then reproduce the conflict in the isolated Task worktree so
	// its branch and worktree remain the reviewable candidate.
	if err := gitRunAt(primary, "merge", "--abort"); err != nil {
		return recordLocalDeliveryFailure(store, id, "conflict-parent-restore", fmt.Errorf("merge failed: %v; abort failed: %w", mergeErr, err))
	}
	if err := gitRunAt(taskWorktree, "merge", "--no-commit", "--no-ff", meta.ParentBranch); err == nil {
		// This should be unusual after the parent merge conflicted. Restore the
		// clean candidate and retain the original failure for diagnosis.
		_ = gitRunAt(taskWorktree, "merge", "--abort")
		return recordLocalDeliveryFailure(store, id, "conflict-candidate", fmt.Errorf("parent merge conflicted but candidate merge did not; review %s with resolving-merge-conflicts", taskWorktree))
	}
	return recordLocalDeliveryFailure(store, id, "conflict-candidate", fmt.Errorf("preserved candidate branch %s in %s; use resolving-merge-conflicts and obtain human review before retrying local-merge: %w", meta.Branch, taskWorktree, mergeErr))
}

func requireTaskBranchCommit(dir, parent, branch string) error {
	output, err := gitOutput(dir, "rev-list", "--count", parent+".."+branch)
	if err != nil {
		return fmt.Errorf("verify committed Task content: %w", err)
	}
	if count := strings.TrimSpace(output); count == "" || count == "0" {
		return fmt.Errorf("local-merge requires committed Task-branch content not present on %s", parent)
	}
	return nil
}

func absoluteTaskWorktree(primary, worktree string) string {
	if filepath.IsAbs(worktree) {
		return filepath.Clean(worktree)
	}
	return filepath.Join(primary, filepath.FromSlash(worktree))
}

func gitDirty(dir string) (bool, error) {
	output, err := gitOutput(dir, "status", "--porcelain=v1")
	if err != nil {
		return false, fmt.Errorf("inspect Task worktree cleanliness: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

func gitRunAt(dir string, args ...string) error {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func isAncestorAt(dir, ancestor, descendant string) bool {
	command := exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant)
	command.Dir = dir
	return command.Run() == nil
}

func recordLocalDeliveryFailure(store *workflow.Store, id, stage string, cause error) error {
	if _, recordErr := store.RecordDeliveryFailure(workflow.TaskID(id), stage, cause.Error()); recordErr != nil {
		return fmt.Errorf("%w; record delivery recovery evidence: %v", cause, recordErr)
	}
	return cause
}
