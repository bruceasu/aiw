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
	parent, err := issue.Read(args[1])
	if err != nil {
		return err
	}
	issues, err := issue.List(issue.ListAll)
	if err != nil {
		return err
	}
	for _, issue := range issues {
		if issue.ParentID == parent.ID {
			fmt.Printf("%s\t%s\t%s\n", issue.ID, issue.Status, issue.Title)
		}
	}
	return nil
}
