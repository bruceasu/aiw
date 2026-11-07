//go:build windows

package codex

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const dpapiPrefix = "dpapi:v1:"

func encodeStoredOpenAICodexCredential(data []byte) ([]byte, error) {
	if len(data) == 0 { return nil, errors.New("cannot encrypt empty OpenAI Codex credentials") }
	input := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var output windows.DataBlob
	if err := windows.CryptProtectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, errors.New("cannot protect OpenAI Codex credentials with Windows DPAPI")
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(output.Data))))
	protected := append([]byte(nil), unsafe.Slice(output.Data, int(output.Size))...)
	encoded := make([]byte, len(dpapiPrefix)+base64.RawStdEncoding.EncodedLen(len(protected)))
	copy(encoded, dpapiPrefix)
	base64.RawStdEncoding.Encode(encoded[len(dpapiPrefix):], protected)
	return encoded, nil
}

func decodeStoredOpenAICodexCredential(data []byte) ([]byte, error) {
	if len(data) <= len(dpapiPrefix) || string(data[:len(dpapiPrefix)]) != dpapiPrefix {
		return nil, errors.New("stored OpenAI Codex credentials are not DPAPI-protected")
	}
	protected := make([]byte, base64.RawStdEncoding.DecodedLen(len(data)-len(dpapiPrefix)))
	n, err := base64.RawStdEncoding.Decode(protected, data[len(dpapiPrefix):])
	if err != nil { return nil, errors.New("stored OpenAI Codex credentials are invalid") }
	protected = protected[:n]
	input := windows.DataBlob{Size: uint32(len(protected)), Data: &protected[0]}
	var output windows.DataBlob
	if err := windows.CryptUnprotectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, errors.New("cannot unprotect OpenAI Codex credentials for this Windows user")
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(output.Data))))
	return append([]byte(nil), unsafe.Slice(output.Data, int(output.Size))...), nil
}

func lockOpenAICodexCredential(ctx context.Context) (func(), error) {
	path, err := openAICodexCredentialPath()
	if err != nil { return nil, err }
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { return nil, errors.New("cannot create AIW credential directory") }
	file, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil { return nil, errors.New("cannot open AIW credential lock") }
	var overlapped windows.Overlapped
	for {
		err = windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlapped)
		if err == nil {
			return func() { _ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlapped); _ = file.Close() }, nil
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop(); _ = file.Close(); return nil, errors.New("waiting for AIW credential lock was cancelled")
		case <-timer.C:
		}
	}
}

