package execution

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func managedProcessToken(pid int, _ string) (string, error) {
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil { return "", err }
	defer windows.CloseHandle(process)
	creation := windows.Filetime{}
	if err := windows.GetProcessTimes(process, &creation, &windows.Filetime{}, &windows.Filetime{}, &windows.Filetime{}); err != nil { return "", err }
	return fmt.Sprintf("%08x%08x", creation.HighDateTime, creation.LowDateTime), nil
}

func managedProcessState(pid int, token, key string) (running, conclusive bool, err error) {
	if len(key) != 32 { return false, false, errors.New("invalid Windows Codex process-tree key") }
	name, err := windows.UTF16PtrFromString("Local\\aiw-codex-"+key)
	if err != nil { return false, false, err }
	job, err := windows.CreateJobObject(nil, name)
	if err != nil { return false, false, fmt.Errorf("reopen Codex process-tree Job Object: %w", err) }
	defer windows.CloseHandle(job)
	var accounting struct {
		TotalUserTime, TotalKernelTime, PeriodUserTime, PeriodKernelTime int64
		TotalPageFaults, TotalProcesses, ActiveProcesses, TerminatedProcesses uint32
	}
	if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
		return false, false, fmt.Errorf("inspect Codex process-tree Job Object: %w", err)
	}
	if accounting.ActiveProcesses > 0 { return true, true, nil }
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) { return false, true, nil }
		return false, false, err
	}
	defer windows.CloseHandle(process)
	creation := windows.Filetime{}
	if err := windows.GetProcessTimes(process, &creation, &windows.Filetime{}, &windows.Filetime{}, &windows.Filetime{}); err != nil { return false, false, err }
	if actual := fmt.Sprintf("%08x%08x", creation.HighDateTime, creation.LowDateTime); actual != token { return false, true, nil }
	state, err := windows.WaitForSingleObject(process, 0)
	if err != nil { return false, false, err }
	if state == uint32(windows.WAIT_TIMEOUT) { return true, true, nil }
	if state == uint32(windows.WAIT_OBJECT_0) { return false, true, nil }
	return false, false, fmt.Errorf("unexpected Codex process wait status 0x%x", state)
}
