package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
)

func showIssue(args []string) error {
	if len(args) != 2 {
		usage, _ := issueSubcommandUsage("show")
		return errors.New(usage)
	}
	meta, err := issue.Read(args[1])
	if err != nil {
		return err
	}
	fmt.Printf("%s\t%s\tapproval=%s\tpromotion=%s\ttask=%s\tparent=%s\n", meta.ID, meta.Status, meta.Approval.Status, meta.Promotion.Status, meta.Promotion.TaskID, meta.ParentID)
	return nil
}
