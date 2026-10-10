package main

import (
	"aiw/internal/issue"
	"fmt"
)

func cancelIssue(args []string) error {
	id, by, reason, err := parseTerminalArgs(args, "cancel")
	if err != nil {
		return err
	}
	meta, err := issue.Cancel(id, by, reason)
	if err != nil {
		return err
	}
	fmt.Printf("requirement %s: %s\n", meta.ID, meta.Status)
	return nil
}
