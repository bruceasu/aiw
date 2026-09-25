package taskx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/workflow"
)

// EnsureRunnerHandoff refreshes generated context and preserves operator-authored handoffs.
func EnsureRunnerHandoff(id string, request *workflow.PreparedAgentRequest, supervised bool) error {
	if request == nil {
		return fmt.Errorf("prepared agent request is required")
	}
	path := filepath.Join(RuntimeTaskDir(id), "artifacts", "handoff.md")
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
	validationInstruction := `After editing, follow the implement Skill's
compile-only procedure: prefer a compile script under scripts/ or the repository
root, otherwise use the narrowest language-level compile command. If compilation
fails, fix the implementation and compile again before reporting completion.`
	if supervised {
		validationInstruction = `Do not run tests. Return a structured outcome after editing; the supervisor runs the frozen Compile Plan and will provide diagnostics if a repair turn is needed.`
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
follow the implement Skill if it is available, and do not resolve Gates.
%s
Mark the selected checklist item as completed when its implementation is
complete.
Report the evidence required to complete the Work Item.
`, request.TaskID, request.WorkItemID, request.AttemptID, validationInstruction)
	return os.WriteFile(path, []byte(content), 0o644)
}

// WriteCompilerRepairHandoff adds compile diagnostics to the Task handoff.
func WriteCompilerRepairHandoff(id string, request *workflow.PreparedAgentRequest, result *workflow.CompilerResult, failures int) (string, error) {
	base := filepath.Join(RuntimeTaskDir(id), "artifacts", "handoff.md")
	content, err := os.ReadFile(base)
	if err != nil {
		return "", err
	}
	diagnostics := filepath.Join(RuntimeTaskDir(id), filepath.FromSlash(result.Diagnostics.Path))
	content = append(content, []byte(fmt.Sprintf("\n## Compiler repair\n\nStay in Attempt %s and Work Item %s. Read diagnostics at %s. Fix only these compile failures. Do not start another Attempt or run compilers or tests; the supervisor runs the frozen Compile Plan. Return a structured outcome. Consecutive compile failures: %d of %d.\n", request.AttemptID, request.WorkItemID, diagnostics, failures, workflow.MaxConsecutiveCompileFailures))...)
	path := filepath.Join(filepath.Dir(base), "compiler-repair.md")
	return path, os.WriteFile(path, content, 0o644)
}
