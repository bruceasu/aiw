package main

import (
	"fmt"
	"os"

	workflowcmd "aiw/internal/workflow/facade"
	airuntime "aiw/internal/runtime"
	workflowcli "aiw/internal/workflow/cli"
	taskadapter "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow/execution"
)

// The standalone workflow program is the implementation behind the aiw-wf
// plugin. The facade keeps CLI orchestration small while the workflow Store,
// state machine, Session, delivery, and Task adapters remain replaceable
// seams inside the workflow packages.
func main() {
	if len(os.Args) > 1 && os.Args[1] == "--auxiliary-root" {
		if err := execution.RunAuxiliaryHelper(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "auxiliary:", err)
			os.Exit(1)
		}
		return
	}
	airuntime.ConfigureProduction()

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
