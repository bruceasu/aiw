package help

import (
	"os"
	"path/filepath"
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
