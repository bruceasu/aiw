package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
)

func captureIssue(args []string) error {
	if len(args) != 4 || args[2] != "--file" {
		usage, _ := issueSubcommandUsage("capture")
		return errors.New(usage)
	}
	_, artifact, err := issue.Capture(args[0], args[1], args[3])
	if err != nil {
		return err
	}
	fmt.Printf("captured %s: %s\n", artifact.Kind, artifact.Path)
	return nil
}
