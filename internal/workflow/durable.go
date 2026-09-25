package workflow

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const durableLockMarker = "aiw-task-lock:v2\n"

// durableWrite never removes the old target or retries an uncertain replace.
// A failure after publication requires reconciliation, not replay of a caller.
func durableWrite(path string, data []byte) error {
	if !durablePlatformSupported() { return errors.New("durable workflow storage is unavailable on this platform") }
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
	file, err := os.CreateTemp(filepath.Dir(path), ".aiw-durable-*")
	if err != nil { return err }
	name := file.Name()
	defer os.Remove(name)
	n, writeErr := file.Write(data)
	if writeErr == nil && n != len(data) { writeErr = io.ErrShortWrite }
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil { return err }
	if err := durableReplace(name, path); err != nil { return fmt.Errorf("durable replacement requires reconciliation: %w", err) }
	return nil
}

func isSystemLock(file *os.File) bool {
	data := make([]byte, len(durableLockMarker))
	n, err := file.ReadAt(data, 0)
	return err == nil && n == len(data) && bytes.Equal(data, []byte(durableLockMarker))
}

func releaseTaskLock(file *os.File) error {
	if file == nil { return nil }
	if isSystemLock(file) { return systemTaskUnlock(file) }
	return errors.Join(file.Close(), os.Remove(file.Name()))
}

// PrepareDurableTaskLock is a maintenance operation, never automatic stale-lock
// recovery. The caller must have fenced old binaries and verified writer exit.
// An existing legacy lock is preserved even when its PID or age looks stale.
func (s *Store) PrepareDurableTaskLock(id TaskID, verifyMaintenance func() error) error {
	if verifyMaintenance == nil || !durablePlatformSupported() { return errors.New("verified maintenance and Windows storage are required") }
	file, err := s.lock(id)
	if err != nil { return err }
	if isSystemLock(file) { return releaseTaskLock(file) }
	if err := verifyMaintenance(); err != nil { _ = releaseTaskLock(file); return err }
	state, err := s.Load(id)
	if err != nil { _ = releaseTaskLock(file); return err }
	if state.PendingEvent != nil || state.WriteLease != nil || state.Automation.PreparedRequest != nil || activeAttempt(state.Attempts) != "" || state.Automation.Supervisor.LeaseID != "" {
		_ = releaseTaskLock(file)
		return errors.New("reconcile pending commits, attempts, supervisors and workspace writers before lock migration")
	}
	if err := systemTaskLock(file); err != nil { _ = releaseTaskLock(file); return err }
	// Once marker publication starts, preserve the file even on failure. An
	// incomplete marker must fail closed rather than invite an old O_EXCL writer.
	n, writeErr := file.WriteAt([]byte(durableLockMarker), 0)
	if writeErr == nil && n != len(durableLockMarker) { writeErr = io.ErrShortWrite }
	return errors.Join(writeErr, file.Sync(), systemTaskUnlock(file))
}
