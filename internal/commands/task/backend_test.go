package task

import "testing"

func TestSelectBackendNativeDoesNotProbe(t *testing.T) {
	t.Setenv("AIW_OPENSPEC_BIN", "definitely-not-an-executable")
	mode, args, err := selectBackend("new", []string{"TASK-1", "--backend", "native"})
	if err != nil {
		t.Fatal(err)
	}
	if mode != backendNative || len(args) != 1 || args[0] != "TASK-1" {
		t.Fatalf("unexpected result: %s %#v", mode, args)
	}
}

func TestSelectBackendRejectsInvalidMode(t *testing.T) {
	if _, _, err := selectBackend("new", []string{"TASK-1", "--backend", "invalid"}); err == nil {
		t.Fatal("expected invalid backend error")
	}
}

func TestExplicitOpenSpecRequiresVerifiedExecutable(t *testing.T) {
	t.Setenv("AIW_OPENSPEC_BIN", "definitely-not-an-executable")
	if _, _, err := selectBackend("new", []string{"TASK-1", "--backend", "openspec"}); err == nil {
		t.Fatal("expected missing OpenSpec error")
	}
}

func TestOpenSpecModeRejectsUnsupportedOperation(t *testing.T) {
	if _, _, err := selectBackend("decision", []string{"TASK-1", "--backend", "openspec"}); err == nil {
		t.Fatal("expected unsupported operation error")
	}
}

func TestNewTaskMetaRecordsParentBranch(t *testing.T) {
	meta := taskMetaFor("TASK-1", "feature/base")
	if meta.ParentBranch != "feature/base" {
		t.Fatalf("parent branch = %q, want %q", meta.ParentBranch, "feature/base")
	}
	if meta.Session != "TASK-1" {
		t.Fatalf("session = %q, want %q", meta.Session, "TASK-1")
	}
}
