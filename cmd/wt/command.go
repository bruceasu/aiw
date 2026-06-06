package wt

import (
	"fmt"
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/taskx"
)

type RemoveOptions struct {
	DeleteBranch bool
	Force        bool
}

type ListOptions struct {
	Porcelain bool
}

type PruneOptions struct {
	DryRun bool
}

func Dispatch(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		usage()
		return nil
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		if len(rest) == 0 {
			return fmt.Errorf("usage: aiw wt add <task-id> [base-branch]")
		}
		base := ""
		if len(rest) >= 2 {
			base = rest[1]
		}
		return create(rest[0], base)
	case "rm":
		if len(rest) == 0 {
			return fmt.Errorf("usage: aiw wt rm <task-id> [--delete-branch] [--force]")
		}
		opts, err := parseRemoveOptions(rest[1:])
		if err != nil {
			return err
		}
		return remove(rest[0], opts)
	case "list", "ls":
		opts, err := parseListOptions(rest)
		if err != nil {
			return err
		}
		return list(opts)
	case "prune":
		opts, err := parsePruneOptions(rest)
		if err != nil {
			return err
		}
		return prune(opts)
	case "lock":
		if len(rest) == 0 {
			return fmt.Errorf("usage: aiw wt lock <task-id> [reason]")
		}
		reason := strings.TrimSpace(strings.Join(rest[1:], " "))
		return lock(rest[0], reason)
	case "unlock":
		if len(rest) == 0 {
			return fmt.Errorf("usage: aiw wt unlock <task-id>")
		}
		return unlock(rest[0])
	case "repair":
		return repair()
	case "ignore":
		return taskx.EnsureWorktreeIgnored()
	default:
		return fmt.Errorf("unknown wt subcommand: %s  (run: aiw wt help)", sub)
	}
}

func parseRemoveOptions(args []string) (RemoveOptions, error) {
	allowed := map[string]bool{
		"--delete-branch": true,
		"--force":         true,
	}
	for _, a := range args {
		if !allowed[a] {
			return RemoveOptions{}, fmt.Errorf("unknown wt-rm option: %s", a)
		}
	}
	return RemoveOptions{
		DeleteBranch: hasFlag(args, "--delete-branch"),
		Force:        hasFlag(args, "--force"),
	}, nil
}

func parseListOptions(args []string) (ListOptions, error) {
	allowed := map[string]bool{
		"--porcelain": true,
	}
	for _, a := range args {
		if !allowed[a] {
			return ListOptions{}, fmt.Errorf("unknown wt-list option: %s", a)
		}
	}
	return ListOptions{Porcelain: hasFlag(args, "--porcelain")}, nil
}

func parsePruneOptions(args []string) (PruneOptions, error) {
	allowed := map[string]bool{
		"--dry-run": true,
	}
	for _, a := range args {
		if !allowed[a] {
			return PruneOptions{}, fmt.Errorf("unknown wt-prune option: %s", a)
		}
	}
	return PruneOptions{DryRun: hasFlag(args, "--dry-run")}, nil
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func create(id, base string) error {
	taskDir := taskx.TaskDir(id)
	if !fsx.Exists(taskDir) {
		return fmt.Errorf("task not found: %s", id)
	}
	branch := "feature/" + id
	wt := filepath.Join(taskx.WorktreeDir, id)
	if gitx.HasRemote("origin") {
		if err := gitx.Run("git", "fetch", "origin"); err != nil {
			return err
		}
	}
	if base == "" {
		detected, err := gitx.DetectBaseBranch()
		if err != nil {
			return err
		}
		base = detected
		fmt.Println("base branch:", base)
	}
	if err := gitx.Run("git", "worktree", "add", wt, "-b", branch, base); err != nil {
		return err
	}
	metaPath := taskx.TaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(metaPath)
	if err != nil {
		return err
	}
	meta.Branch = branch
	meta.Worktree = filepath.ToSlash(wt)
	meta.Updated = taskx.Today()
	if err := taskx.WriteTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return taskx.WriteRegistry()
}

func remove(id string, opts RemoveOptions) error {
	taskDir := taskx.TaskDir(id)
	if !fsx.Exists(taskDir) {
		return fmt.Errorf("task not found: %s", id)
	}

	metaPath := taskx.TaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(metaPath)
	if err != nil {
		return err
	}

	branch := strings.TrimSpace(meta.Branch)
	if branch == "" {
		branch = "feature/" + id
	}

	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(taskx.WorktreeDir, id))
	}

	removeArgs := []string{"worktree", "remove", wt}
	if opts.Force {
		removeArgs = append(removeArgs, "--force")
	}
	if err := gitx.Run("git", removeArgs...); err != nil {
		return err
	}

	meta.Worktree = ""
	if opts.DeleteBranch {
		if err := gitx.Run("git", "branch", "-d", branch); err != nil {
			return err
		}
		meta.Branch = ""
	}

	meta.Updated = taskx.Today()
	if err := taskx.WriteTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return taskx.WriteRegistry()
}

func list(opts ListOptions) error {
	args := []string{"worktree", "list"}
	if opts.Porcelain {
		args = append(args, "--porcelain")
	}
	return gitx.Run("git", args...)
}

func prune(opts PruneOptions) error {
	args := []string{"worktree", "prune"}
	if opts.DryRun {
		args = append(args, "-n", "-v")
	}
	return gitx.Run("git", args...)
}

func lock(id, reason string) error {
	wt, err := resolveTaskWorktree(id)
	if err != nil {
		return err
	}
	args := []string{"worktree", "lock", wt}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	return gitx.Run("git", args...)
}

func unlock(id string) error {
	wt, err := resolveTaskWorktree(id)
	if err != nil {
		return err
	}
	return gitx.Run("git", "worktree", "unlock", wt)
}

func repair() error {
	return gitx.Run("git", "worktree", "repair")
}

func resolveTaskWorktree(id string) (string, error) {
	taskDir := taskx.TaskDir(id)
	if !fsx.Exists(taskDir) {
		return "", fmt.Errorf("task not found: %s", id)
	}
	meta, err := taskx.ReadTaskMeta(taskx.TaskMetaPath(id))
	if err != nil {
		return "", err
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(taskx.WorktreeDir, id))
	}
	return wt, nil
}

func usage() {
	fmt.Print(`aiw wt - worktree management

  add <task-id> [base]                       Create worktree + branch feature/<task-id>.
  rm  <task-id> [--delete-branch] [--force]  Remove worktree (safe by default).
  list [--porcelain]                         List all registered worktrees.
  prune [--dry-run]                          Clean stale worktree metadata.
  lock <task-id> [reason]                    Lock a worktree against removal/prune.
  unlock <task-id>                           Unlock a locked worktree.
  repair                                     Repair worktree links after repo/path moves.
  ignore                                     Add .wt/ ignore rule to .gitignore.

Examples:
  aiw wt add payment-retry
  aiw wt add payment-retry origin/main
  aiw wt lock payment-retry "hotfix in progress"
  aiw wt list
  aiw wt prune --dry-run
  aiw wt rm payment-retry --delete-branch --force
`)
}
