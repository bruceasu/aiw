package gitx

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseDirtyPathsIncludesRenameAndDeletePaths(t *testing.T) {
	paths, err := parseDirtyPaths([]byte(" M docs/guide.md\x00D  old.txt\x00R  new.txt\x00old-name.txt\x00"))
	if err != nil {
		t.Fatalf("parse dirty paths: %v", err)
	}
	want := []string{
		filepath.Join("docs", "guide.md"),
		"old.txt",
		"new.txt",
		"old-name.txt",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %#v, want %#v", paths, want)
	}
}

func TestDirtyPathsReturnsStatusReadError(t *testing.T) {
	_, err := dirtyPaths(func() ([]byte, error) { return nil, errors.New("git unavailable") })
	if err == nil || err.Error() != "read worktree status: git unavailable" {
		t.Fatalf("unexpected error: %v", err)
	}
}
