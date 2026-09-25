//go:build !windows

package execution

import (
	"errors"
	"os/exec"
)

func detachAuxiliaryProcess(command *exec.Cmd) error {
	return errors.New("verified durable auxiliary host is only available on Windows")
}
