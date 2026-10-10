//go:build !linux && !windows

package ai

import (
	"errors"
	"os/exec"
)

func configureManagedProcess(*exec.Cmd, string) error { return errors.New("managed Codex process-tree supervision is unsupported on this platform") }
func startManagedProcess(*exec.Cmd) error { return nil }
func resumeManagedProcess(*exec.Cmd) error { return nil }
func finishManagedProcess(*exec.Cmd) error { return nil }
