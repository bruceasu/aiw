package task

import (
	"path/filepath"
	"strings"

	"aiw/internal/gitx"
)

// ResolvedWorkspaceKind resolves legacy Task metadata to an explicit workspace kind.
func ResolvedWorkspaceKind(meta TaskMeta) string {
	if kind := strings.TrimSpace(meta.WorkspaceKind); kind != "" {
		return kind
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		return "unassigned"
	}
	if wt == "." {
		return "primary"
	}
	root, err := gitx.ProjectRoot()
	if err != nil {
		return "unknown"
	}
	path := wt
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, filepath.FromSlash(path))
	}
	if gitx.WorktreeRegistered(path) {
		return "isolated"
	}
	return "unknown"
}
