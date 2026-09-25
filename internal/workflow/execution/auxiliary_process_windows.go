//go:build windows

package execution

import (
	"os/exec"
	"syscall"
)

func detachAuxiliaryProcess(command *exec.Cmd) error {
	// DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP. HideWindow also prevents
	// a console window when launched by a windowed parent application.
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200, HideWindow: true}
	return nil
}
