package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type RemoveWorktreeOptions struct {
	DeleteBranch bool
	Force        bool
}

type WorktreeListOptions struct {
	Porcelain bool
}

type WorktreePruneOptions struct {
	DryRun bool
}

func createWorktree(id, base string) error {
	taskDir := filepath.Join(changesDir, id)
	if !exists(taskDir) {
		return fmt.Errorf("task not found: %s", id)
	}
	branch := "feature/" + id
	wt := filepath.Join(worktreeDir, id)
	if hasRemote("origin") {
		if err := run("git", "fetch", "origin"); err != nil {
			return err
		}
	}
	if base == "" {
		detected, err := detectBaseBranch()
		if err != nil {
			return err
		}
		base = detected
		fmt.Println("base branch:", base)
	}
	if err := run("git", "worktree", "add", wt, "-b", branch, base); err != nil {
		return err
	}
	metaPath := filepath.Join(taskDir, "task.toml")
	meta, err := readTaskMeta(metaPath)
	if err != nil {
		return err
	}
	meta.Branch = branch
	meta.Worktree = filepath.ToSlash(wt)
	meta.Updated = today()
	if err := writeTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return writeRegistry()
}

func removeWorktree(id string, opts RemoveWorktreeOptions) error {
	taskDir := filepath.Join(changesDir, id)
	if !exists(taskDir) {
		return fmt.Errorf("task not found: %s", id)
	}

	metaPath := filepath.Join(taskDir, "task.toml")
	meta, err := readTaskMeta(metaPath)
	if err != nil {
		return err
	}

	branch := strings.TrimSpace(meta.Branch)
	if branch == "" {
		branch = "feature/" + id
	}

	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(worktreeDir, id))
	}

	removeArgs := []string{"worktree", "remove", wt}
	if opts.Force {
		removeArgs = append(removeArgs, "--force")
	}
	if err := run("git", removeArgs...); err != nil {
		return err
	}

	meta.Worktree = ""

	if opts.DeleteBranch {
		if err := run("git", "branch", "-d", branch); err != nil {
			return err
		}
		meta.Branch = ""
	}

	meta.Updated = today()
	if err := writeTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return writeRegistry()
}

func listWorktrees(opts WorktreeListOptions) error {
	args := []string{"worktree", "list"}
	if opts.Porcelain {
		args = append(args, "--porcelain")
	}
	return run("git", args...)
}

func pruneWorktrees(opts WorktreePruneOptions) error {
	args := []string{"worktree", "prune"}
	if opts.DryRun {
		args = append(args, "-n", "-v")
	}
	return run("git", args...)
}

func lockWorktree(id, reason string) error {
	wt, err := resolveTaskWorktree(id)
	if err != nil {
		return err
	}
	args := []string{"worktree", "lock", wt}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	return run("git", args...)
}

func unlockWorktree(id string) error {
	wt, err := resolveTaskWorktree(id)
	if err != nil {
		return err
	}
	return run("git", "worktree", "unlock", wt)
}

func repairWorktrees() error {
	return run("git", "worktree", "repair")
}

func resolveTaskWorktree(id string) (string, error) {
	taskDir := filepath.Join(changesDir, id)
	if !exists(taskDir) {
		return "", fmt.Errorf("task not found: %s", id)
	}
	meta, err := readTaskMeta(filepath.Join(taskDir, "task.toml"))
	if err != nil {
		return "", err
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(worktreeDir, id))
	}
	return wt, nil
}

func hasRemote(name string) bool {
	cmd := exec.Command("git", "remote", "get-url", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func refExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func detectBaseBranch() (string, error) {
	for _, candidate := range []string{"origin/main", "origin/master", "main", "master"} {
		if refExists(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("cannot detect base branch; pass one explicitly, e.g.: aiw wt <task-id> main")
}

func parseRemoveWorktreeOptions(args []string) (RemoveWorktreeOptions, error) {
	allowed := map[string]bool{
		"--delete-branch": true,
		"--force":         true,
	}
	for _, a := range args {
		if !allowed[a] {
			return RemoveWorktreeOptions{}, fmt.Errorf("unknown wt-rm option: %s", a)
		}
	}
	return RemoveWorktreeOptions{
		DeleteBranch: hasFlag(args, "--delete-branch"),
		Force:        hasFlag(args, "--force"),
	}, nil
}

func parseWorktreeListOptions(args []string) (WorktreeListOptions, error) {
	allowed := map[string]bool{
		"--porcelain": true,
	}
	for _, a := range args {
		if !allowed[a] {
			return WorktreeListOptions{}, fmt.Errorf("unknown wt-list option: %s", a)
		}
	}
	return WorktreeListOptions{Porcelain: hasFlag(args, "--porcelain")}, nil
}

func parseWorktreePruneOptions(args []string) (WorktreePruneOptions, error) {
	allowed := map[string]bool{
		"--dry-run": true,
	}
	for _, a := range args {
		if !allowed[a] {
			return WorktreePruneOptions{}, fmt.Errorf("unknown wt-prune option: %s", a)
		}
	}
	return WorktreePruneOptions{DryRun: hasFlag(args, "--dry-run")}, nil
}

// dispatchWt routes `aiw wt <subcommand> [args]`.
func dispatchWt(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		wtUsage()
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
		return createWorktree(rest[0], base)
	case "rm":
		if len(rest) == 0 {
			return fmt.Errorf("usage: aiw wt rm <task-id> [--delete-branch] [--force]")
		}
		opts, err := parseRemoveWorktreeOptions(rest[1:])
		if err != nil {
			return err
		}
		return removeWorktree(rest[0], opts)
	case "list", "ls":
		opts, err := parseWorktreeListOptions(rest)
		if err != nil {
			return err
		}
		return listWorktrees(opts)
	case "prune":
		opts, err := parseWorktreePruneOptions(rest)
		if err != nil {
			return err
		}
		return pruneWorktrees(opts)
	case "lock":
		if len(rest) == 0 {
			return fmt.Errorf("usage: aiw wt lock <task-id> [reason]")
		}
		reason := strings.TrimSpace(strings.Join(rest[1:], " "))
		return lockWorktree(rest[0], reason)
	case "unlock":
		if len(rest) == 0 {
			return fmt.Errorf("usage: aiw wt unlock <task-id>")
		}
		return unlockWorktree(rest[0])
	case "repair":
		return repairWorktrees()
	case "ignore":
		return ensureWorktreeIgnored()
	default:
		return fmt.Errorf("unknown wt subcommand: %s  (run: aiw wt help)", sub)
	}
}

func wtUsage() {
	fmt.Print(`aiw wt — worktree management

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
