package help

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDispatchJSON(t *testing.T) {
	oldExecutable := executablePathFn
	defer func() { executablePathFn = oldExecutable }()

	tmp := t.TempDir()
	pluginsDir := filepath.Join(tmp, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pluginPath := filepath.Join(pluginsDir, "aiw-sample.py")
	if err := os.WriteFile(pluginPath, []byte("META = {'short': 'sample plugin'}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return filepath.Join(tmp, "aiw.exe"), nil }

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	dispatchErr := Dispatch([]string{"--json"})
	w.Close()
	os.Stdout = oldStdout
	if dispatchErr != nil {
		t.Fatal(dispatchErr)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Command  string `json:"command"`
		Builtins []struct {
			Name        string `json:"name"`
			Short       string `json:"short"`
			Description string `json:"description"`
			Source      string `json:"source"`
		} `json:"builtins"`
		Plugins []struct {
			Name        string `json:"name"`
			Short       string `json:"short"`
			Description string `json:"description"`
			Source      string `json:"source"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, buf.String())
	}
	if doc.Command != "help" {
		t.Fatalf("unexpected command: %s", doc.Command)
	}
	if len(doc.Builtins) == 0 {
		t.Fatalf("expected builtins in json: %s", buf.String())
	}
	if len(doc.Plugins) == 0 {
		t.Fatalf("expected plugins in json: %s", buf.String())
	}
	for _, b := range doc.Builtins {
		if b.Name == "" || b.Source != "builtin" {
			t.Fatalf("unexpected builtin entry: %+v", b)
		}
	}
	foundPlugin := false
	for _, p := range doc.Plugins {
		if p.Name == "sample" || p.Name == "aiw-sample" || p.Short == "sample plugin" {
			foundPlugin = true
		}
		if p.Source != "plugin" {
			t.Fatalf("unexpected plugin entry: %+v", p)
		}
	}
	if !foundPlugin {
		t.Fatalf("expected sample plugin in json: %s", buf.String())
	}
}
