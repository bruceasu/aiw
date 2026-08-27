package gitx

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Run(name string, args ...string) error {
	fmt.Fprintf(os.Stderr, "+ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func HasRemote(name string) bool {
	cmd := exec.Command("git", "remote", "get-url", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func RefExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func CurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read current branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", errors.New("current checkout is not on a branch")
	}
	return branch, nil
}

func ProjectRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil { return "", fmt.Errorf("resolve project root: %w", err) }
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

func PrimaryWorktree() (string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil { return "", fmt.Errorf("list worktrees: %w", err) }
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") { return filepath.Clean(strings.TrimSpace(strings.TrimPrefix(line, "worktree "))), nil }
	}
	return "", errors.New("primary worktree not found")
}

func IsPrimaryWorktree() (bool, string, error) {
	root, err := ProjectRoot(); if err != nil { return false, "", err }
	primary, err := PrimaryWorktree(); if err != nil { return false, "", err }
	return strings.EqualFold(root, primary), primary, nil
}

func IsDirty() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil { return false, fmt.Errorf("read worktree status: %w", err) }
	return len(strings.TrimSpace(string(out))) > 0, nil
}

func IsAncestor(ancestor, descendant string) bool {
	return exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant).Run() == nil
}

func WorktreeRegistered(path string) bool {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	out, err := cmd.Output(); if err != nil { return false }
	target, err := filepath.Abs(path); if err != nil { return false }
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "worktree ") { continue }
		candidate, err := filepath.Abs(strings.TrimSpace(strings.TrimPrefix(line, "worktree "))); if err == nil && strings.EqualFold(filepath.Clean(candidate), filepath.Clean(target)) { return true }
	}
	return false
}

func DetectBaseBranch() (string, error) {
	for _, candidate := range []string{"origin/main", "origin/master", "main", "master"} {
		if RefExists(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("cannot detect base branch; pass one explicitly, e.g.: aiw wt <task-id> main")
}
