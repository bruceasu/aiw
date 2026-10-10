package ui

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// PromptLine reads a single line from stdin with the given prompt.
func PromptLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

// EnsureTrailingNewline makes sure the string ends with a newline.
func EnsureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

// GitOutput runs a git command and returns its trimmed stdout.
func GitOutput(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// IsInteractiveTerminal reports whether stdin is a terminal.
func IsInteractiveTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// RunEditorCommand launches the given editor command with the supplied path.
func RunEditorCommand(editor, path string) error {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return errors.New("empty editor command")
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsEditorShortcut determines whether the input should trigger the external editor.
func IsEditorShortcut(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "/edit", "^e", "ctrl+e", "control+e":
		return true
	}
	return strings.ContainsRune(s, rune(5))
}

// ResolveEditor determines the editor command to use based on env vars and defaults.
func ResolveEditor(editorConfig string) string {
	if editorConfig != "" {
		return editorConfig
	}
	for _, key := range []string{"GIT_EDITOR", "VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	if out, err := GitOutput("git", "config", "--get", "core.editor"); err == nil && out != "" {
		return out
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vim"
}
