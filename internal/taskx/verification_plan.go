package taskx

import "path/filepath"

const (
	VerificationPlanArtifact      = "verification-plan.json"
	VerificationSelectionArtifact = "focused-test-selection.json"
	FocusedTestResultArtifact     = "focused-test-result.json"
	FocusedTestOutputArtifact     = "focused-test-output.txt"
)

// VerificationPlanPath returns the task-local, human-authored plan path.
func VerificationPlanPath(taskID string) string {
	return filepath.Join(TaskDir(taskID), "artifacts", VerificationPlanArtifact)
}

// TaskRuntimeDir is the Task-local home for mutable execution artifacts.
// Authored OpenSpec inputs remain under TaskDir.
func TaskRuntimeDir(taskID string) string {
	return filepath.Join(".ai", "tasks", taskID)
}

// VerificationSelectionPath returns the auditable test-agent selection path.
func VerificationSelectionPath(taskID string) string {
	return filepath.Join(TaskRuntimeDir(taskID), "artifacts", VerificationSelectionArtifact)
}

// FocusedTestResultPath returns the controlled runner result artifact path.
func FocusedTestResultPath(taskID string) string {
	return filepath.Join(TaskRuntimeDir(taskID), "artifacts", FocusedTestResultArtifact)
}

// FocusedTestOutputPath returns the bounded process output captured by the
// controlled runner. The companion result artifact records its digest and
// whether the output was truncated.
func FocusedTestOutputPath(taskID string) string {
	return filepath.Join(TaskRuntimeDir(taskID), "artifacts", FocusedTestOutputArtifact)
}
