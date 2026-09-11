package ask

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
)

// readPolicy describes the paths this invocation may read. It deliberately
// contains no project files: a path is passed to a provider only when the user
// explicitly supplied --allow-path.
type readPolicy struct {
	workspace string
	private   string
	allowed   []string
	confirmed bool
}

func newReadPolicy(o options) (*readPolicy, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("find ask workspace: %w", err)
	}
	workspace, err := cleanExistingPath(wd)
	if err != nil {
		return nil, fmt.Errorf("resolve ask workspace: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find home directory for ask path policy: %w", err)
	}
	policy := &readPolicy{workspace: workspace, private: filepath.Join(home, ".aiw", "ask")}
	for _, raw := range o.allowPaths {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		path, err := cleanExistingPath(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid --allow-path %q: %w", raw, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("inspect --allow-path %q: %w", raw, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("--allow-path %q must be a directory", raw)
		}
		if !containsPath(policy.allowed, path) {
			policy.allowed = append(policy.allowed, path)
		}
	}
	return policy, nil
}

func (p *readPolicy) allowedDirs() []string {
	return append([]string(nil), p.allowed...)
}

// authorizeSystemPromptFile permits an explicit command-line file directly.
// Configured files remain constrained to private, workspace, or allowed roots.
func (p *readPolicy) authorizeSystemPromptFile(path string, explicit, interactive bool) (string, error) {
	resolved, err := cleanExistingPath(path)
	if err != nil {
		return "", err
	}
	if explicit || p.contains(resolved) {
		return resolved, nil
	}
	if !interactive {
		return "", fmt.Errorf("configured system prompt file %q is outside the workspace and private ask directory; authorize its directory with --allow-path", path)
	}
	if !p.confirmed {
		prompt := promptui.Prompt{Label: fmt.Sprintf("Allow AIW ask to read external file %s? (y/N)", resolved)}
		answer, promptErr := prompt.Run()
		if promptErr != nil || !strings.EqualFold(strings.TrimSpace(answer), "y") {
			return "", errors.New("external system prompt file was not authorized")
		}
		p.confirmed = true
	}
	return resolved, nil
}

func (p *readPolicy) contains(path string) bool {
	return within(path, p.workspace) || within(path, p.private) || containsPath(p.allowed, path)
}

func containsPath(roots []string, path string) bool {
	for _, root := range roots {
		if within(path, root) {
			return true
		}
	}
	return false
}

func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func cleanExistingPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}
