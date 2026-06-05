package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverPluginIn_PluginsDir(t *testing.T) {
	td := t.TempDir()
	plugins := filepath.Join(td, "plugins")
	if err := os.Mkdir(plugins, 0755); err != nil {
		t.Fatal(err)
	}

	name := "hello"
	wantBase := "aiw-" + name

	// create candidates according to platform
	if runtime.GOOS == "windows" {
		f1 := filepath.Join(plugins, wantBase+".bat")
		if err := os.WriteFile(f1, []byte("@echo hello"), 0644); err != nil {
			t.Fatal(err)
		}
		// also create python variant
		f2 := filepath.Join(plugins, wantBase+".py")
		if err := os.WriteFile(f2, []byte("print('hi')"), 0644); err != nil {
			t.Fatal(err)
		}
	} else {
		f1 := filepath.Join(plugins, wantBase+".sh")
		if err := os.WriteFile(f1, []byte("#!/bin/sh\necho hello"), 0755); err != nil {
			t.Fatal(err)
		}
		f2 := filepath.Join(plugins, wantBase+".py")
		if err := os.WriteFile(f2, []byte("print('hi')"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	path, err := DiscoverPluginIn([]string{plugins}, name)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if path == "" {
		t.Fatalf("empty path returned")
	}
}
