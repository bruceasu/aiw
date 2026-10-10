package ai

import (
	"encoding/hex"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var managedJobs = struct { sync.Mutex; handles map[*exec.Cmd]windows.Handle }{handles: make(map[*exec.Cmd]windows.Handle)}

func configureManagedProcess(cmd *exec.Cmd, key string) error {
	if len(key) != 32 { return fmt.Errorf("invalid process-tree key") }
	if _, err := hex.DecodeString(key); err != nil { return fmt.Errorf("invalid process-tree key: %w", err) }
	name, err := windows.UTF16PtrFromString("Local\\aiw-codex-"+key)
	if err != nil { return err }
	job, err := windows.CreateJobObject(nil, name)
	if err != nil { return fmt.Errorf("create process-tree Job Object: %w", err) }
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		_ = windows.CloseHandle(job); return fmt.Errorf("configure process-tree Job Object: %w", err)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
	managedJobs.Lock(); managedJobs.handles[cmd] = job; managedJobs.Unlock()
	return nil
}

func startManagedProcess(cmd *exec.Cmd) error {
	managedJobs.Lock(); job := managedJobs.handles[cmd]; managedJobs.Unlock()
	if job == 0 { return fmt.Errorf("managed process Job Object is missing") }
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_INFORMATION, false, uint32(cmd.Process.Pid))
	if err != nil { return fmt.Errorf("open suspended Codex process: %w", err) }
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil { return fmt.Errorf("assign Codex process to Job Object: %w", err) }
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil { return fmt.Errorf("snapshot suspended Codex thread: %w", err) }
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	if err := windows.Thread32First(snapshot, &entry); err != nil { return fmt.Errorf("enumerate suspended Codex thread: %w", err) }
	for {
		if entry.OwnerProcessID == uint32(cmd.Process.Pid) {
			return nil
		}
		if err := windows.Thread32Next(snapshot, &entry); err != nil { return fmt.Errorf("find suspended Codex thread: %w", err) }
	}
}

func resumeManagedProcess(cmd *exec.Cmd) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil { return fmt.Errorf("snapshot suspended Codex thread: %w", err) }
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	if err := windows.Thread32First(snapshot, &entry); err != nil { return fmt.Errorf("enumerate suspended Codex thread: %w", err) }
	for {
		if entry.OwnerProcessID == uint32(cmd.Process.Pid) {
			thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
			if err != nil { return fmt.Errorf("open suspended Codex thread: %w", err) }
			_, resumeErr := windows.ResumeThread(thread); _ = windows.CloseHandle(thread)
			if resumeErr != nil { return fmt.Errorf("resume Codex process: %w", resumeErr) }
			return nil
		}
		if err := windows.Thread32Next(snapshot, &entry); err != nil { return fmt.Errorf("find suspended Codex thread: %w", err) }
	}
}

func finishManagedProcess(cmd *exec.Cmd) error {
	managedJobs.Lock(); job := managedJobs.handles[cmd]; delete(managedJobs.handles, cmd); managedJobs.Unlock()
	if job == 0 { return nil }
	if err := windows.CloseHandle(job); err != nil { return fmt.Errorf("close Codex process-tree Job Object: %w", err) }
	return nil
}
