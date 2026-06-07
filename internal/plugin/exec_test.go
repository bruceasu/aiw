package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInterpreterCommandPrefersProgramDir(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	pythonDir := filepath.Join(exeDir, "python")
	if err := os.MkdirAll(pythonDir, 0o755); err != nil {
		t.Fatal(err)
	}

	localPython := filepath.Join(pythonDir, executableName("python"))
	if err := os.WriteFile(localPython, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	oldExecutablePathFn := pluginExecutablePathFn
	oldLookPathFn := lookPathFn
	pluginExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "aiw.exe"), nil
	}
	lookPathFn = func(file string) (string, error) {
		t.Fatalf("lookPathFn should not be called when local interpreter exists, got %q", file)
		return "", nil
	}
	defer func() {
		pluginExecutablePathFn = oldExecutablePathFn
		lookPathFn = oldLookPathFn
	}()

	got, err := resolveInterpreterCommand(".py", "")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 1 || got[0] != localPython {
		t.Fatalf("resolveInterpreterCommand() = %v, want [%q]", got, localPython)
	}
}

func TestResolveInterpreterCommandFallsBackToSystem(t *testing.T) {
	oldExecutablePathFn := pluginExecutablePathFn
	oldLookPathFn := lookPathFn
	pluginExecutablePathFn = func() (string, error) {
		return filepath.Join(t.TempDir(), "aiw.exe"), nil
	}
	lookPathFn = func(file string) (string, error) {
		if file == "python" {
			return "/usr/bin/python", nil
		}
		t.Fatalf("unexpected lookPathFn input: %q", file)
		return "", nil
	}
	defer func() {
		pluginExecutablePathFn = oldExecutablePathFn
		lookPathFn = oldLookPathFn
	}()

	got, err := resolveInterpreterCommand(".py", "")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 1 || got[0] != "/usr/bin/python" {
		t.Fatalf("resolveInterpreterCommand() = %v, want [/usr/bin/python]", got)
	}
}

func TestResolveInterpreterCommandForJarUsesJavaBinAndJarFlag(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	javaBinDir := filepath.Join(exeDir, "java", "bin")
	if err := os.MkdirAll(javaBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	localJava := filepath.Join(javaBinDir, executableName("java"))
	if err := os.WriteFile(localJava, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	oldExecutablePathFn := pluginExecutablePathFn
	oldLookPathFn := lookPathFn
	pluginExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "aiw.exe"), nil
	}
	lookPathFn = func(file string) (string, error) {
		t.Fatalf("lookPathFn should not be called when local java exists, got %q", file)
		return "", nil
	}
	defer func() {
		pluginExecutablePathFn = oldExecutablePathFn
		lookPathFn = oldLookPathFn
	}()

	got, err := resolveInterpreterCommand(".jar", "")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 2 || got[0] != localJava || got[1] != "-jar" {
		t.Fatalf("resolveInterpreterCommand() = %v, want [%q -jar]", got, localJava)
	}
}

func TestResolveInterpreterCommandShebangUsesProgramDirFirst(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	bashDir := filepath.Join(exeDir, "bash")
	if err := os.MkdirAll(bashDir, 0o755); err != nil {
		t.Fatal(err)
	}

	localBash := filepath.Join(bashDir, executableName("bash"))
	if err := os.WriteFile(localBash, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	oldExecutablePathFn := pluginExecutablePathFn
	oldLookPathFn := lookPathFn
	pluginExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "aiw.exe"), nil
	}
	lookPathFn = func(file string) (string, error) {
		t.Fatalf("lookPathFn should not be called when local bash exists, got %q", file)
		return "", nil
	}
	defer func() {
		pluginExecutablePathFn = oldExecutablePathFn
		lookPathFn = oldLookPathFn
	}()

	got, err := resolveInterpreterCommand("", "bash")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 1 || got[0] != localBash {
		t.Fatalf("resolveInterpreterCommand() = %v, want [%q]", got, localBash)
	}
}
