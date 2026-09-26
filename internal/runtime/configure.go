// Package runtime contains executable composition for optional AIW programs.
// The root aiw executable deliberately does not install these adapters;
// standalone plugins own their runtime composition.
package runtime

import (
	"aiw/internal/session"
	"aiw/internal/workflow"
	"aiw/internal/workflow/execution"
)

// ConfigureProduction installs the adapters required by the standalone wf
// and req programs. It is intentionally called only by those program entry
// points, never by the root CLI.
func ConfigureProduction() {
	workflow.ConfigureAuxiliaryStore = execution.ConfigureProductionAuxiliary
	session.TaskMemoryProjection = execution.ProjectSessionTaskMemory
}
