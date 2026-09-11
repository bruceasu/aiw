package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Root returns the primary repository root so runtime data is shared by the
// primary checkout and linked worktrees. Non-Git temporary directories keep
// the current-directory behavior used by isolated unit tests.
func Root() string {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	if out, err := cmd.Output(); err == nil {
		commonDir := strings.TrimSpace(string(out))
		if !filepath.IsAbs(commonDir) {
			if cwd, cwdErr := os.Getwd(); cwdErr == nil {
				commonDir = filepath.Join(cwd, commonDir)
			}
		}
		return filepath.Clean(filepath.Dir(commonDir))
	}
	return "."
}
