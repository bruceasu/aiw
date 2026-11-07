//go:build !windows

package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func encodeStoredOpenAICodexCredential(data []byte) ([]byte, error) { return data, nil }
func decodeStoredOpenAICodexCredential(data []byte) ([]byte, error) { return data, nil }

func lockOpenAICodexCredential(ctx context.Context) (func(), error) {
	path, err := openAICodexCredentialPath()
	if err != nil { return nil, err }
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { return nil, errors.New("cannot create AIW credential directory") }
	file, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil { return nil, errors.New("cannot open AIW credential lock") }
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil { return func() { _ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN); _ = file.Close() }, nil }
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN { _ = file.Close(); return nil, errors.New("cannot lock AIW credentials") }
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop(); _ = file.Close(); return nil, errors.New("waiting for AIW credential lock was cancelled")
		case <-timer.C:
		}
	}
}

