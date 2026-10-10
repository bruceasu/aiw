package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
)

func listIssues(args []string) error {
	if len(args) > 1 {
		return errors.New("usage: aiw req list [--all|--archived|--cancelled]")
	}
	filter := issue.ListActive
	if len(args) == 1 {
		switch args[0] {
		case "--all":
			filter = issue.ListAll
		case "--archived":
			filter = issue.ListArchived
		case "--cancelled":
			filter = issue.ListCancelled
		default:
			return errors.New("usage: aiw req list [--all|--archived|--cancelled]")
		}
	}
	items, err := issue.List(filter)
	if err != nil {
		return err
	}
	for _, meta := range items {
		fmt.Printf("%s\t%s\t%s\n", meta.ID, meta.Status, meta.Title)
	}
	return nil
}
