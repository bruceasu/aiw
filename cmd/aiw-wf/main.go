package main

import (
	"fmt"
	"os"

	workflowcmd "aiw/internal/workflow/facade"
	workflowcli "aiw/internal/workflow/cli"
	taskadapter "aiw/internal/task/workflowadapter"
)

// The standalone workflow program is the implementation behind the aiw-wf
// plugin. The facade keeps CLI orchestration small while the workflow Store,
// state machine, Session, delivery, and Task adapters remain replaceable
// seams inside the workflow packages.
func main() {
	facade := workflowcmd.New(
		func(args []string) error {
			return workflowcli.RunWorkflowCommand(taskadapter.New(taskadapter.DefaultOperations{}), args)
		},
		workflowcli.PrintWorkflowHelp,
	)
	if err := facade.Dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "wf:", err)
		os.Exit(1)
	}
}
