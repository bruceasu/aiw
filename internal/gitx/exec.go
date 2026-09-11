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
	paths, err := DirtyPaths()
	if err != nil {
		return false, err
	}
	return len(paths) > 0, nil
}

// DirtyPaths returns normalized repository-relative paths with uncommitted changes.
// Renamed and copied entries return both paths so callers can safely check either
// side of the change for overlap.
func DirtyPaths() ([]string, error) {
	return dirtyPaths(readDirtyStatus)
}

func dirtyPaths(readStatus func() ([]byte, error)) ([]string, error) {
	out, err := readStatus()
	if err != nil {
		return nil, fmt.Errorf("read worktree status: %w", err)
	}
	return parseDirtyPaths(out)
}

func readDirtyStatus() ([]byte, error) {
	cmd := exec.Command("git", "status", "--porcelain=v1", "-z")
	return cmd.Output()
}

func parseDirtyPaths(status []byte) ([]string, error) {
	fields := strings.Split(string(status), "\x00")
	paths := make([]string, 0, len(fields))
	for index := 0; index < len(fields)-1; index++ {
		record := fields[index]
		if record == "" {
			continue
		}
		if len(record) < 4 || record[2] != ' ' {
			return nil, fmt.Errorf("parse worktree status record: %q", record)
		}
		paths = append(paths, normalizeStatusPath(record[3:]))
		if record[0] != 'R' && record[0] != 'C' && record[1] != 'R' && record[1] != 'C' {
			continue
		}
		if index+1 >= len(fields)-1 || fields[index+1] == "" {
			return nil, fmt.Errorf("parse renamed worktree status record: %q", record)
		}
		index++
		paths = append(paths, normalizeStatusPath(fields[index]))
	}
	return paths, nil
}

func normalizeStatusPath(path string) string {
	return filepath.Clean(filepath.FromSlash(path))
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
