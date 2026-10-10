package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
)

func linkIssueParent(args []string) error {
	if len(args) != 3 {
		usage, _ := issueSubcommandUsage("link-parent")
		return errors.New(usage)
	}
	meta, err := issue.LinkParent(args[1], args[2])
	if err != nil {
		return err
	}
	fmt.Printf("issue %s parent=%s\n", meta.ID, meta.ParentID)
	return nil
}
