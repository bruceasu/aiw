//go:build !windows

package say

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type systemClipboard struct{}

func NewClipboard() Clipboard { return systemClipboard{} }

type commandClipboard struct {
	readName  string
	readArgs  []string
	writeName string
	writeArgs []string
}

func (systemClipboard) Read(ctx context.Context) (string, error) {
	clipboard, err := platformClipboard()
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, clipboard.readName, clipboard.readArgs...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read clipboard with %s: %w", clipboard.readName, err)
	}
	return string(output), nil
}

func (systemClipboard) Write(ctx context.Context, text string) error {
	clipboard, err := platformClipboard()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, clipboard.writeName, clipboard.writeArgs...)
	cmd.Stdin = bytes.NewBufferString(text)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write clipboard with %s: %w: %s", clipboard.writeName, err, string(output))
	}
	return nil
}

func platformClipboard() (commandClipboard, error) {
	if runtime.GOOS == "linux" && isWSL() {
		return wslClipboard()
	}
	if runtime.GOOS != "linux" {
		return commandClipboard{}, ErrClipboardUnavailable
	}
	return linuxClipboard()
}

func isWSL() bool {
	if os.Getenv("WSL_INTEROP") != "" || os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	return err == nil && strings.Contains(strings.ToLower(string(data)), "microsoft")
}

func wslClipboard() (commandClipboard, error) {
	for _, name := range []string{"powershell.exe", "pwsh.exe"} {
		if _, err := exec.LookPath(name); err == nil {
			return commandClipboard{
				readName: name,
				readArgs: []string{
					"-NoLogo", "-NoProfile", "-NonInteractive", "-STA", "-Command",
					`[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new(); [Console]::Write((Get-Clipboard -Raw -ErrorAction Stop))`,
				},
				writeName: name,
				writeArgs: []string{
					"-NoLogo", "-NoProfile", "-NonInteractive", "-STA", "-Command",
					`$stream = [Console]::OpenStandardInput(); $memory = [IO.MemoryStream]::new(); $stream.CopyTo($memory); $text = [Text.Encoding]::UTF8.GetString($memory.ToArray()); Set-Clipboard -Value $text -ErrorAction Stop`,
				},
			}, nil
		}
	}
	return commandClipboard{}, errors.New("WSL clipboard access requires Windows PowerShell or PowerShell on PATH and enabled Windows interop")
}

func linuxClipboard() (commandClipboard, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return pairedClipboard("wl-paste", []string{"--no-newline"}, "wl-copy", nil)
	}
	if os.Getenv("DISPLAY") != "" {
		if clipboard, err := pairedClipboard("xclip", []string{"-selection", "clipboard", "-o"}, "xclip", []string{"-selection", "clipboard", "-i"}); err == nil {
			return clipboard, nil
		}
		if clipboard, err := pairedClipboard("xsel", []string{"--clipboard", "--output"}, "xsel", []string{"--clipboard", "--input"}); err == nil {
			return clipboard, nil
		}
		return commandClipboard{}, errors.New("X11 clipboard requires xclip or xsel with both read and write commands")
	}
	return commandClipboard{}, errors.New("clipboard access requires an active Wayland or X11 display session")
}

func pairedClipboard(readName string, readArgs []string, writeName string, writeArgs []string) (commandClipboard, error) {
	if _, err := exec.LookPath(readName); err != nil {
		return commandClipboard{}, fmt.Errorf("clipboard read command %s is unavailable", readName)
	}
	if _, err := exec.LookPath(writeName); err != nil {
		return commandClipboard{}, fmt.Errorf("clipboard write command %s is unavailable", writeName)
	}
	return commandClipboard{
		readName: readName, readArgs: readArgs,
		writeName: writeName, writeArgs: writeArgs,
	}, nil
}

