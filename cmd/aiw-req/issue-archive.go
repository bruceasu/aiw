package main

import (
	"aiw/internal/issue"
	"fmt"
)

func archiveIssue(args []string) error {
	id, by, reason, err := parseTerminalArgs(args, "archive")
	if err != nil {
		return err
	}
	meta, err := issue.Archive(id, by, reason)
	if err != nil {
		return err
	}
	fmt.Printf("requirement %s: %s\n", meta.ID, meta.Status)
	return nil
}
