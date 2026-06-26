package taskx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTaskMetaPath_PrefersTaskToml(t *testing.T) {
	id := "resolve-prefer-primary"
	dir := TaskDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	primary := filepath.Join(dir, TaskMetaFile)
	legacy := filepath.Join(dir, LegacyTaskMetaFile)
	if err := os.WriteFile(primary, []byte("id = \"a\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("id = \"b\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ResolveTaskMetaPath(id)
	if got != primary {
		t.Fatalf("got %q, want %q", got, primary)
	}
}

func TestResolveTaskMetaPath_FallsBackToLegacy(t *testing.T) {
	id := "resolve-fallback-legacy"
	dir := TaskDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	legacy := filepath.Join(dir, LegacyTaskMetaFile)
	if err := os.WriteFile(legacy, []byte("id = \"legacy\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ResolveTaskMetaPath(id)
	if got != legacy {
		t.Fatalf("got %q, want %q", got, legacy)
	}
}

func TestReadTaskMeta_ParsesSpecsAndTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.toml")
	content := "id = \"sample\"\nstatus = \"TODO\"\nspecs = [\"billing\", \"risk\"]\ntags = [\"backend\", \"critical\"]\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	meta, err := ReadTaskMeta(path)
	if err != nil {
		t.Fatalf("read task meta: %v", err)
	}
	if len(meta.Specs) != 2 || meta.Specs[0] != "billing" || meta.Specs[1] != "risk" {
		t.Fatalf("unexpected specs: %+v", meta.Specs)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "backend" || meta.Tags[1] != "critical" {
		t.Fatalf("unexpected tags: %+v", meta.Tags)
	}
}
