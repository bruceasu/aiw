package task

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"aiw/internal/taskx"
)

// preflightSupervisedGitWorkspace proves that the Task binding identifies a
// registered worktree and supplies the child process with trust for that one
// canonical directory only. It deliberately does not write global or local
// Git configuration.
func preflightSupervisedGitWorkspace(meta taskx.TaskMeta) ([]string, error) {
	worktree := strings.TrimSpace(meta.Worktree)
	if worktree == "" {
		return nil, fmt.Errorf("supervised Git preflight: Task worktree is empty")
	}
	if !filepath.IsAbs(worktree) {
		root, err := projectRoot()
		if err != nil {
			return nil, fmt.Errorf("supervised Git preflight: resolve project root: %w", err)
		}
		worktree = filepath.Join(root, filepath.FromSlash(worktree))
	}
	canonical, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		return nil, fmt.Errorf("supervised Git preflight: resolve Task worktree: %w", err)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return nil, fmt.Errorf("supervised Git preflight: normalize Task worktree: %w", err)
	}
	environment := scopedGitEnvironment(os.Environ(), canonical)
	registered, err := registeredGitWorktrees(canonical, environment)
	if err != nil {
		return nil, err
	}
	if !registered {
		return nil, fmt.Errorf("supervised Git preflight: Task worktree is not registered: %s", canonical)
	}
	cmd := exec.Command("git", "-C", canonical, "rev-parse", "--show-toplevel")
	cmd.Env = environment
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("supervised Git preflight: verify Git access for %s: %w", canonical, err)
	}
	gitRoot, err := filepath.Abs(strings.TrimSpace(string(output)))
	if err != nil || !samePath(gitRoot, canonical) {
		return nil, fmt.Errorf("supervised Git preflight: Git root does not match Task worktree: %s", canonical)
	}
	return environment, nil
}

func projectRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return filepath.Abs(strings.TrimSpace(string(output)))
}

func registeredGitWorktrees(canonical string, environment []string) (bool, error) {
	cmd := exec.Command("git", "-C", canonical, "worktree", "list", "--porcelain")
	cmd.Env = environment
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("supervised Git preflight: list registered worktrees: %w", err)
	}
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		if samePath(strings.TrimPrefix(line, "worktree "), canonical) {
			return true, nil
		}
	}
	return false, nil
}

func scopedGitEnvironment(base []string, directory string) []string {
	environment := make([]string, 0, len(base)+3)
	for _, entry := range base {
		key, _, found := strings.Cut(entry, "=")
		if !found || isGitConfigEnvironmentKey(key) {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=safe.directory",
		"GIT_CONFIG_VALUE_0="+directory,
	)
}

func isGitConfigEnvironmentKey(key string) bool {
	if os.PathSeparator == '\\' {
		key = strings.ToUpper(key)
	}
	return key == "GIT_CONFIG_COUNT" || strings.HasPrefix(key, "GIT_CONFIG_KEY_") || strings.HasPrefix(key, "GIT_CONFIG_VALUE_")
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(left, right)
	}
	return left == right
}
