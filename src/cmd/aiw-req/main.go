package main

import (
	"fmt"
	"os"
)

// The req plugin owns the public Requirement CLI and its domain model.
func main() {
	if err := Dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "req:", err)
		os.Exit(1)
	}
}

// Dispatch is the public entry point for the standalone req program.
func Dispatch(args []string) error { return DispatchIssue(args) }

func DispatchIssue(args []string) error {
	if len(args) == 0 || isIssueHelpFlag(args[0]) {
		fmt.Print(requirementUsage)
		return nil
	}
	if len(args) == 2 && isIssueHelpFlag(args[1]) {
		if usage, ok := issueSubcommandUsage(args[0]); ok {
			fmt.Print(usage)
			return nil
		}
	}
	switch args[0] {
	case "chat":
		return dispatchIssueChat(args[1:])
	case "new":
		return dispatchIssueNew(args)
	case "show":
		return showIssue(args)
	case "link-parent":
		return linkIssueParent(args)
	case "children":
		return listIssueChildren(args)
	case "list":
		return listIssues(args[1:])
	case "capture":
		return captureIssue(args[1:])
	case "approve":
		return approveIssue(args[1:])
	case "promote":
		return promoteIssue(args[1:])
	case "archive":
		return archiveIssue(args[1:])
	case "cancel":
		return cancelIssue(args[1:])
	default:
		return fmt.Errorf("unknown requirement command: %s", args[0])
	}
}
