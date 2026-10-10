package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func promoteIssue(args []string) error {
	const usage = "usage: aiw issue promote <id>"
	if len(args) != 1 || !issue.ValidID(args[0]) {
		return errors.New(usage)
	}

	meta, err := issue.Read(args[0])
	if err != nil {
		return err
	}
	if (meta.Status != "APPROVED" && meta.Status != "PROMOTED") || meta.Approval.Status != "APPROVED" {
		return fmt.Errorf("issue is not approved: %s", meta.ID)
	}
	if _, err := issue.ArtifactSnapshot(meta.ID); err != nil { return err }

	cli, err := aiwCLI()
	if err != nil {
		return err
	}
	cmd := exec.Command(cli, "fd", "new", meta.Title, "--issue", meta.ID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("create FD for Issue %s: %w", meta.ID, err)
	}
	return nil
}

func aiwCLI() (string, error) {
	name := "aiw"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if root := os.Getenv("AIW_ROOT"); root != "" {
		candidate := filepath.Join(root, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if candidate, err := exec.LookPath("aiw"); err == nil {
		return candidate, nil
	}
	return "", errors.New("cannot find the AIW CLI required to create an FD")
}
