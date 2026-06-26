package help

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestListPluginsUsesExecutableDir(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	pluginsDir := filepath.Join(exeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "aiw-right.py"), []byte("print('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd := filepath.Join(td, "cwd")
	if err := os.MkdirAll(filepath.Join(cwd, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "plugins", "aiw-wrong.py"), []byte("print('wrong')"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}

	oldExecutablePathFn := executablePathFn
	executablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "aiw.exe"), nil
	}
	defer func() {
		executablePathFn = oldExecutablePathFn
	}()

	got, err := listPlugins()
	if err != nil {
		t.Fatalf("listPlugins returned error: %v", err)
	}
	if len(got) != 1 || got[0] != "right" {
		t.Fatalf("listPlugins() = %v, want [right]", got)
	}
}

func TestPluginScriptPathUsesExecutableDir(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	pluginsDir := filepath.Join(exeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(pluginsDir, "aiw-sample.py")
	if err := os.WriteFile(want, []byte("print('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldExecutablePathFn := executablePathFn
	executablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "aiw.exe"), nil
	}
	defer func() {
		executablePathFn = oldExecutablePathFn
	}()

	if got := pluginScriptPath("sample"); got != want {
		t.Fatalf("pluginScriptPath() = %q, want %q", got, want)
	}
}

func TestStaticBuiltinCommandsIncludesAI(t *testing.T) {
	builtins := staticBuiltinCommands()
	if !slices.Contains(builtins, "ai") {
		t.Fatalf("staticBuiltinCommands() = %v, expected ai to be present", builtins)
	}
}

func TestBuiltinExistsRecognizesTopLevelAI(t *testing.T) {
	if !builtinExists("ai") {
		t.Fatal("expected builtinExists(ai) to be true")
	}
}

func TestBuiltinUsageTextForNew(t *testing.T) {
	usage, ok := builtinUsageText("new")
	if !ok {
		t.Fatal("expected usage text for new")
	}
	if usage == "" {
		t.Fatal("expected non-empty usage text for new")
	}
}

func TestShowBuiltinHelpUsesInlineUsageWithoutExecution(t *testing.T) {
	oldExecPathFn := executablePathFn
	oldExecCmdFn := execCommandFn
	defer func() {
		executablePathFn = oldExecPathFn
		execCommandFn = oldExecCmdFn
	}()

	executablePathFn = func() (string, error) {
		return "", nil
	}

	called := false
	execCommandFn = func(name string, arg ...string) *exec.Cmd {
		called = true
		return exec.Command("cmd", "/C", "echo should-not-run")
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	err = showBuiltinHelp("new")
	_ = w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatalf("showBuiltinHelp(new): %v", err)
	}
	if called {
		t.Fatal("expected showBuiltinHelp(new) to avoid external command execution")
	}

	var out bytes.Buffer
	_, _ = out.ReadFrom(r)
	_ = r.Close()
	if out.String() == "" {
		t.Fatal("expected usage output")
	}
}
