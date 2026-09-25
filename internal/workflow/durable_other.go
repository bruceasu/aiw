//go:build !windows

package workflow

import (
	"errors"
	"os"
)

// The first protocol is deliberately limited to the validated Windows host.
// Legacy Store operations retain their original platform behavior.
func durableReplace(source, target string) error { return errors.New("durable workflow replacement requires a validated Windows local volume") }
func systemTaskLock(file *os.File) error { return errors.New("durable Task locks are unavailable on this platform") }
func systemTaskUnlock(file *os.File) error { return file.Close() }
func durablePlatformSupported() bool { return false }
