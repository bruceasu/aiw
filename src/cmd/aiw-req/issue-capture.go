package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
	"os"
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
	warnCapturedDraft(artifact)
	fmt.Printf("captured %s: %s\n", artifact.Kind, artifact.Path)
	return nil
}

func warnCapturedDraft(artifact issue.Artifact) {
	if artifact.CleanupWarning != "" {
		fmt.Fprintln(os.Stderr, "warning:", artifact.CleanupWarning)
	}
}
