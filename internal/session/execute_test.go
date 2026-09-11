package session

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecuteInteractiveAppliesOverridesWithoutPersistingThem(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	command := writeSessionRecordingCommand(t)
	if err := os.WriteFile("aiw.toml", []byte("[ai]\ncopilot_command = \""+strings.ReplaceAll(command, "\\", "\\\\")+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore("")
	if _, err := store.Create("S-1", "S-1", root, "codex", "session-model", "instructions"); err != nil {
		t.Fatal(err)
	}

	result, err := ExecuteInteractiveWithOverrides(context.Background(), store, "S-1", "handoff", "prompt", "copilot", "cli-model", true)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("interactive exit code = %d", result.ExitCode)
	}
	status, err := store.Load("S-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Backend.Name != "codex" || status.Backend.Model != "session-model" {
		t.Fatalf("CLI overrides were persisted: %+v", status.Backend)
	}
	if status.Session.State != StateActive || status.Session.LastTurn != 1 || status.Execution.LastExitCode == nil || *status.Execution.LastExitCode != 0 {
		t.Fatalf("unexpected interactive Session state: %+v", status)
	}
}

func writeSessionRecordingCommand(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(t.TempDir(), "provider.bat")
		if err := os.WriteFile(path, []byte("@exit /b 0\r\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	path := filepath.Join(t.TempDir(), "provider.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
