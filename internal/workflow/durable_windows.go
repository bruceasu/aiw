//go:build windows

package workflow

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func durableReplace(source, target string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil { return err }
	to, err := windows.UTF16PtrFromString(target)
	if err != nil { return err }
	if err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH); err != nil { return err }
	file, err := os.OpenFile(target, os.O_RDWR, 0)
	if err != nil { return err }
	return errors.Join(file.Sync(), file.Close())
}

func systemTaskLock(file *os.File) error {
	return windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
}

func systemTaskUnlock(file *os.File) error {
	return errors.Join(windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &windows.Overlapped{}), file.Close())
}

func durablePlatformSupported() bool { return true }
