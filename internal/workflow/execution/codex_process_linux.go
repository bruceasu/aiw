package execution

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func managedProcessToken(pid int, _ string) (string, error) {
	start, group, bootID, err := linuxProcessIdentity(pid)
	if err != nil { return "", err }
	if group != pid { return "", errors.New("Codex process is not in its dedicated process group") }
	return fmt.Sprintf("%s:%s:%d", bootID, start, group), nil
}

func managedProcessState(pid int, token, _ string) (running, conclusive bool, err error) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 { return false, false, errors.New("invalid Linux Codex process identity") }
	start, group, bootID, statErr := linuxProcessIdentity(pid)
	if statErr == nil && start == parts[1] && strconv.Itoa(group) == parts[2] && bootID == parts[0] { return true, true, nil }
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) { return false, false, statErr }
	groupID, err := strconv.Atoi(parts[2])
	if err != nil { return false, false, err }
	if killErr := syscall.Kill(-groupID, 0); killErr == nil || errors.Is(killErr, syscall.EPERM) { return true, true, nil }
	if errors.Is(killErr, syscall.ESRCH) { return false, true, nil }
	return false, false, killErr
}

func linuxProcessIdentity(pid int) (start string, group int, bootID string, err error) {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil { return "", 0, "", err }
	closeName := strings.LastIndexByte(string(stat), ')')
	if closeName < 0 { return "", 0, "", errors.New("malformed Linux process stat") }
	fields := strings.Fields(string(stat[closeName+1:]))
	if len(fields) <= 19 { return "", 0, "", errors.New("incomplete Linux process stat") }
	group, err = strconv.Atoi(fields[2])
	if err != nil { return "", 0, "", err }
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil { return "", 0, "", err }
	return fields[19], group, strings.TrimSpace(string(boot)), nil
}
