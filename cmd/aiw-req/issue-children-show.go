package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
)

func listIssueChildren(args []string) error {
	if len(args) != 2 {
		usage, _ := issueSubcommandUsage("children")
		return errors.New(usage)
	}
	if _, err := issue.Read(args[1]); err != nil {
		return err
	}
	issues, err := issue.List(issue.ListAll)
	if err != nil {
		return err
	}
	for _, issue := range issues {
		if issue.ParentID == args[1] {
			fmt.Printf("%s\t%s\t%s\n", issue.ID, issue.Status, issue.Title)
		}
	}
	return nil
}
