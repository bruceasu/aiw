//go:build windows

package say

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

const (
	clipboardUnicodeText = 13
	globalMoveable       = 0x0002
)

var (
	clipboardUser32      = syscall.NewLazyDLL("user32.dll")
	clipboardKernel32    = syscall.NewLazyDLL("kernel32.dll")
	openClipboardProc    = clipboardUser32.NewProc("OpenClipboard")
	closeClipboardProc   = clipboardUser32.NewProc("CloseClipboard")
	emptyClipboardProc   = clipboardUser32.NewProc("EmptyClipboard")
	getClipboardDataProc = clipboardUser32.NewProc("GetClipboardData")
	setClipboardDataProc = clipboardUser32.NewProc("SetClipboardData")
	globalAllocProc      = clipboardKernel32.NewProc("GlobalAlloc")
	globalFreeProc       = clipboardKernel32.NewProc("GlobalFree")
	globalLockProc       = clipboardKernel32.NewProc("GlobalLock")
	globalUnlockProc     = clipboardKernel32.NewProc("GlobalUnlock")
	globalSizeProc       = clipboardKernel32.NewProc("GlobalSize")
)

type systemClipboard struct{}

func NewClipboard() Clipboard { return systemClipboard{} }

func (systemClipboard) Read(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if result, _, callErr := openClipboardProc.Call(0); result == 0 {
		return "", windowsCallError("open clipboard", callErr)
	}
	defer closeClipboardProc.Call()

	handle, _, callErr := getClipboardDataProc.Call(clipboardUnicodeText)
	if handle == 0 {
		return "", windowsCallError("read Unicode clipboard data", callErr)
	}
	size, _, callErr := globalSizeProc.Call(handle)
	if size < 2 {
		return "", windowsCallError("inspect clipboard text", callErr)
	}
	locked, _, callErr := globalLockProc.Call(handle)
	if locked == 0 {
		return "", windowsCallError("lock clipboard text", callErr)
	}
	defer globalUnlockProc.Call(handle)
	units := unsafe.Slice((*uint16)(unsafe.Pointer(locked)), int(size/2))
	text := syscall.UTF16ToString(units)
	if !utf8.ValidString(text) {
		return "", errors.New("clipboard text is not valid Unicode")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return text, nil
}

func (systemClipboard) Write(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !utf8.ValidString(text) {
		return errors.New("clipboard text is not valid UTF-8")
	}
	units := utf16.Encode([]rune(text))
	units = append(units, 0)
	handle, _, callErr := globalAllocProc.Call(globalMoveable, uintptr(len(units)*2))
	if handle == 0 {
		return windowsCallError("allocate clipboard text", callErr)
	}
	owned := true
	defer func() {
		if owned {
			globalFreeProc.Call(handle)
		}
	}()
	locked, _, callErr := globalLockProc.Call(handle)
	if locked == 0 {
		return windowsCallError("lock clipboard text", callErr)
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(locked)), len(units)), units)
	globalUnlockProc.Call(handle)
	if err := ctx.Err(); err != nil {
		return err
	}
	if result, _, callErr := openClipboardProc.Call(0); result == 0 {
		return windowsCallError("open clipboard", callErr)
	}
	defer closeClipboardProc.Call()
	if result, _, callErr := emptyClipboardProc.Call(); result == 0 {
		return windowsCallError("clear clipboard", callErr)
	}
	if result, _, callErr := setClipboardDataProc.Call(clipboardUnicodeText, handle); result == 0 {
		return windowsCallError("write Unicode clipboard data", callErr)
	}
	owned = false
	return nil
}

func windowsCallError(operation string, callErr error) error {
	if callErr == nil || callErr == syscall.Errno(0) {
		return fmt.Errorf("%s failed", operation)
	}
	return fmt.Errorf("%s: %w", operation, callErr)
}
