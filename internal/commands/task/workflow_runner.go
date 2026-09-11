package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// ensureRunnerHandoff creates only the minimal managed context needed by an
// explicit Runner execution. A user-authored handoff always takes precedence.
func ensureRunnerHandoff(id string, request *workflow.PreparedAgentRequest) error {
	if request == nil {
		return fmt.Errorf("prepared agent request is required")
	}
	path := filepath.Join(taskx.RuntimeTaskDir(id), "artifacts", "handoff.md")
	if _, err := os.Stat(path); err == nil {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		// Preserve an operator-authored handoff. AIW-generated handoffs are
		// refreshed for each Attempt so the Agent receives the current Work Item.
		if !strings.HasPrefix(string(content), "# Managed workflow handoff\n") {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`# Managed workflow handoff

Task: %s
Work Item: %s
Attempt: %s

Read the following Task artifacts before acting:

- proposal.md
- design.md
- tasks.md
- specs/

Implement only the selected Work Item. Preserve the managed Task and workspace,
follow the implement Skill if it is available, do not resolve Gates or run
tests or broad build workflows. After editing, follow the implement Skill's
compile-only procedure: prefer a compile script under scripts/ or the repository
root, otherwise use the narrowest language-level compile command. If compilation
fails, fix the implementation and compile again before reporting completion.
Mark the selected checklist item as completed when its implementation is
complete.
Report the evidence required to complete the Work Item.
`, request.TaskID, request.WorkItemID, request.AttemptID)
	return os.WriteFile(path, []byte(content), 0o644)
}
