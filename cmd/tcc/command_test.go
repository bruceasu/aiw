package tcc

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArgsAddsIncludeAndLibDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "include"), 0o755); err != nil {
		t.Fatalf("create include dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatalf("create lib dir: %v", err)
	}

	args := argsFor("", []string{"hello.c", "-o", "hello.exe"}, root)
	want := []string{
		"hello.c",
		"-o",
		"hello.exe",
		"-I" + filepath.Join(root, "include"),
		"-L" + filepath.Join(root, "lib"),
	}
	for _, item := range want {
		if !containsString(args, item) {
			t.Fatalf("expected %q in args: %v", item, args)
		}
	}
}

func TestArgsAddsModeShortcuts(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "include"), 0o755); err != nil {
		t.Fatalf("create include dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatalf("create lib dir: %v", err)
	}

	args := argsFor("dll", []string{"hello.c", "-o", "hello.dll"}, root)
	if !containsString(args, "-shared") {
		t.Fatalf("expected -shared in args: %v", args)
	}

	args = argsFor("run", []string{"hello.c"}, root)
	if !containsString(args, "-run") {
		t.Fatalf("expected -run in args: %v", args)
	}

	args = argsFor("x86_64", []string{"hello.c", "-o", "hello.exe"}, root)
	if containsString(args, "-shared") || containsString(args, "-run") {
		t.Fatalf("x86_64 mode should not add shared/run flags: %v", args)
	}
}

func TestPrintHelpIncludesDefaultPaths(t *testing.T) {
	out := captureOutput(t, func() {
		printHelp()
	})
	checks := []string{"aiw tcc [args...]", "Auto paths:", "Examples:"}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q in:\n%s", want, out)
		}
	}
}

func TestExecutableDefaultsTo32BitBinary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tcc.exe"), []byte(""), 0o644); err != nil {
		t.Fatalf("write tcc.exe: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "x86_64-win32-tcc.exe"), []byte(""), 0o644); err != nil {
		t.Fatalf("write x86_64-win32-tcc.exe: %v", err)
	}

	if got := executable(root, false); !strings.HasSuffix(strings.ToLower(got), "tcc.exe") {
		t.Fatalf("expected tcc.exe as default, got %q", got)
	}
}

func TestExecutablePrefersX64WhenRequested(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tcc.exe"), []byte(""), 0o644); err != nil {
		t.Fatalf("write tcc.exe: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "x86_64-win32-tcc.exe"), []byte(""), 0o644); err != nil {
		t.Fatalf("write x86_64-win32-tcc.exe: %v", err)
	}

	if got := executable(root, true); !strings.HasSuffix(strings.ToLower(got), "x86_64-win32-tcc.exe") {
		t.Fatalf("expected x86_64-win32-tcc.exe when requested, got %q", got)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	os.Stdout = w
	os.Stderr = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close pipe reader: %v", err)
	}

	return string(out)
}
