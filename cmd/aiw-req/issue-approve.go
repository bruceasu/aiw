package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
	"os/user"
	"strings"
)

func approveIssue(args []string) error {
	usage, _ := issueSubcommandUsage("approve")
	id, decision, by, reason, err := parseApprovalArgs(args, usage)
	if err != nil {
		return err
	}
	meta, err := issue.Approve(id, decision, by, reason)
	if err != nil {
		return err
	}
	fmt.Printf("requirement %s: %s\n", meta.ID, meta.Status)
	return nil
}

func parseApprovalArgs(args []string, usage string) (string, string, string, string, error) {
	if len(args) < 4 || !issue.ValidID(args[0]) {
		return "", "", "", "", errors.New(usage)
	}
	id, decision, by, reason := args[0], args[1], "", ""
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--by":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", "", "", errors.New(usage)
			}
			by, i = args[i+1], i+1
		case "--reason":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", "", "", errors.New(usage)
			}
			reason, i = args[i+1], i+1
		default:
			return "", "", "", "", errors.New(usage)
		}
	}
	if reason == "" {
		return "", "", "", "", errors.New(usage)
	}
	if by == "" {
		current, err := user.Current()
		if err != nil || current.Username == "" {
			return "", "", "", "", errors.New("cannot determine current user; provide --by <actor>")
		}
		by = current.Username
	}
	return id, decision, by, reason, nil
}
