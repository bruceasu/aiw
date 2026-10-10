package say

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrDialogCanceled = errors.New("dialog canceled")

// DialogInput opens an editable multiline Zenity input window.
func DialogInput(ctx context.Context, kind string) (string, error) {
	if kind != "zenity" {
		return "", fmt.Errorf("unsupported dialog %q", kind)
	}
	output, err := runZenity(ctx, []string{"--text-info", "--editable", "--title=AIW Say", "--width=720", "--height=480", "--no-markup"}, "")
	if err != nil {
		return "", fmt.Errorf("open translation input: %w", err)
	}
	return string(output), nil
}

// ShowDialogResult presents a translation. Closing the result window is reported
// as cancellation because the translation has already been copied.
func ShowDialogResult(ctx context.Context, kind, text string) error {
	if kind != "zenity" {
		return fmt.Errorf("unsupported dialog %q", kind)
	}
	_, err := runZenity(ctx, []string{"--text-info", "--title=AIW Say - Translation", "--width=720", "--height=480", "--no-markup"}, text)
	if errors.Is(err, ErrDialogCanceled) {
		return fmt.Errorf("translation was copied, then the result window was closed: %w", ErrDialogCanceled)
	}
	if err != nil {
		return fmt.Errorf("show translation result: %w", err)
	}
	return nil
}

func runZenity(ctx context.Context, args []string, input string) ([]byte, error) {
	if err := ensureZenitySession(); err != nil {
		return nil, err
	}
	name, err := zenityExecutable()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		if strings.TrimSpace(string(exitErr.Stderr)) == "" {
			return nil, ErrDialogCanceled
		}
		return nil, fmt.Errorf("Zenity failed: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	if message := strings.TrimSpace(string(exitOutput(err))); message != "" {
		return nil, fmt.Errorf("Zenity failed: %w: %s", err, message)
	}
	return nil, fmt.Errorf("Zenity failed: %w", err)
}

func zenityExecutable() (string, error) {
	for _, name := range zenityNames() {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", zenityMissingError()
}

func exitOutput(err error) []byte {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Stderr
	}
	return nil
}
