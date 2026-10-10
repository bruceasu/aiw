package ai

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureManagedProcess(cmd *exec.Cmd, _ string) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	cmd.Cancel = func() error {
		if cmd.Process == nil { return os.ErrProcessDone }
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) { return os.ErrProcessDone }
		return err
	}
	return nil
}

func startManagedProcess(*exec.Cmd) error  { return nil }
func resumeManagedProcess(*exec.Cmd) error { return nil }
func finishManagedProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil { return nil }
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) { return nil }
	return err
}
