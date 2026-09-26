// Package workflow owns the public Workflow CLI seam.
//
// The implementation behind the facade is supplied by an adapter so the
// command surface can be replaced without moving Workflow Core, Session,
// delivery, or Task projection code into this package.
package workflow

import "errors"

// Executor is the CLI adapter for Workflow operations.
type Executor func([]string) error

// HelpPrinter renders the canonical Workflow help text.
type HelpPrinter func()

// Facade is the replaceable public Workflow CLI module.
type Facade struct {
	execute Executor
	help    HelpPrinter
}

// New creates a Workflow facade around the current CLI adapter.
func New(execute Executor, help HelpPrinter) Facade {
	return Facade{execute: execute, help: help}
}

// Dispatch handles the canonical aiw wf command surface.
func (f Facade) Dispatch(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		if f.help != nil {
			f.help()
		}
		return nil
	}
	if f.execute == nil {
		return errors.New("workflow facade has no execution adapter")
	}
	return f.execute(args)
}

// OperationNames is the public operation registry used by help tooling and
// shell completion. Keep this list aligned with Workflow command dispatch.
var OperationNames = []string{
	"plan", "sync", "advance", "run", "supervise", "recommend-routing",
	"focused-test", "delivery", "local-merge", "delivery-failed", "report",
	"diagnose", "recover", "repair", "repair-metadata", "help",
}
