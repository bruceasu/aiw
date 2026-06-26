package llm

import (
	"fmt"
	"os/exec"
	"testing"

	czdata "aiw-cz/internal/cz"
)

func TestNormalizeProviderNameAliases(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"auto":        "",
		"codex-cli":   "codex",
		"codex_cli":   "codex",
		"copilot-cli": "copilot",
		"copilot_cli": "copilot",
	}
	for in, want := range cases {
		if got := normalizeProviderName(in); got != want {
			t.Fatalf("normalizeProviderName(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestRunLLMUnknownProviderFails(t *testing.T) {
	_, err := RunLLM("x", czdata.Config{LLMProvider: "unknown-provider"})
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestRunCodexCLINotFound(t *testing.T) {
	oldLookPath := lookPathFn
	defer func() { lookPathFn = oldLookPath }()
	lookPathFn = func(file string) (string, error) {
		return "", fmt.Errorf("not found")
	}
	_, err := RunCodexCLI("test", czdata.Config{})
	if err == nil {
		t.Fatal("expected codex cli not found error")
	}
}

func TestRunCopilotCLINotFound(t *testing.T) {
	oldLookPath := lookPathFn
	defer func() { lookPathFn = oldLookPath }()
	lookPathFn = func(file string) (string, error) {
		return "", fmt.Errorf("not found")
	}
	_, err := RunCopilotCLI("test", czdata.Config{})
	if err == nil {
		t.Fatal("expected copilot cli not found error")
	}
}

func TestCodexCLIAvailable(t *testing.T) {
	oldLookPath := lookPathFn
	defer func() { lookPathFn = oldLookPath }()
	lookPathFn = func(file string) (string, error) {
		if file == "codex" {
			return "codex", nil
		}
		return "", fmt.Errorf("missing")
	}
	if !codexCLIAvailable(czdata.Config{}) {
		t.Fatal("expected codex CLI to be available")
	}
}

func TestCopilotCLIAvailableFalseWhenHelpFails(t *testing.T) {
	oldLookPath := lookPathFn
	oldExec := execCommandContextFn
	defer func() {
		lookPathFn = oldLookPath
		execCommandContextFn = oldExec
	}()
	lookPathFn = func(file string) (string, error) {
		if file == "gh" {
			return "gh", nil
		}
		return "", fmt.Errorf("missing")
	}
	execCommandContextFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("cmd", "/C", "exit", "1")
	}
	if copilotCLIAvailable(czdata.Config{}) {
		t.Fatal("expected copilot CLI to be unavailable when probe fails")
	}
}
