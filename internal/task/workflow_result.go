package task

import "aiw/internal/workflow"

// FocusedTestResult is the small Task-owned result projection consumed by
// Workflow CLI. The detailed focused-test artifact remains Workflow-owned.
type FocusedTestResult struct {
	RuntimeState workflow.RuntimeState
}
