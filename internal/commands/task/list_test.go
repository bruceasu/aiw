package task

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow"
)

// These tests change process-wide output and cwd, so they must not run in parallel.
func listTestWorkspace(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	// Keep repository discovery and command dispatch independent of installed tools.
	t.Setenv("PATH", "")
	t.Setenv("AIW_SESSION_ROOT", "")
	t.Setenv("AIW_OPENSPEC_BIN", "")
	if err := os.MkdirAll(".ai", 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeListTestMeta(t *testing.T, path, id, status string) task.TaskMeta {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := task.TaskMeta{ID: id, Type: "task", Status: status, Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged"}
	if err := task.WriteTaskMeta(path, meta); err != nil {
		t.Fatal(err)
	}
	return meta
}

func captureListCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return captureTaskOutput(t, func() error { return DispatchTopLevel("list", args) })
}

func captureTaskOutput(t *testing.T, command func() error) (string, string, error) {
	t.Helper()
	dir := t.TempDir()
	stdout, err := os.Create(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()
	stderr, err := os.Create(filepath.Join(dir, "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()
	previousOut, previousErr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = previousOut, previousErr }()
	os.Stdout, os.Stderr = stdout, stderr
	commandErr := command()
	os.Stdout, os.Stderr = previousOut, previousErr
	read := func(file *os.File) string {
		content, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}
	return read(stdout), read(stderr), commandErr
}

func assertListRows(t *testing.T, rows ...[2]string) {
	t.Helper()
	stdout, stderr, err := captureListCommand(t)
	if err != nil || stderr != "" {
		t.Fatalf("list returned %v, stderr = %q", err, stderr)
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if stdout == "" { lines = nil }
	if len(lines) != len(rows) { t.Fatalf("list stdout = %q, want %d rows", stdout, len(rows)) }
	for i, row := range rows {
		if got := strings.Fields(lines[i]); !reflect.DeepEqual(got, []string{row[0], row[1], "规格已删除"}) {
			t.Fatalf("row %d = %q", i, lines[i])
		}
	}
}

func TestListCommandDiscoversMetadataAndWorkflowStatus(t *testing.T) {
	listTestWorkspace(t)
	for _, name := range []string{"compile-cache", "issue", "locks", "requirements", "sessions", "tasks", "tmp", "future-runtime-directory"} {
		if err := os.MkdirAll(filepath.Join(".ai", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(".ai", "ordinary-file"), []byte("not a task"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Metadata deeper than the supported legacy root must not be discovered.
	writeListTestMeta(t, ".ai/sessions/nested/task.toml", "nested", "TODO")
	writeListTestMeta(t, ".ai/tasks/container/deeper/task.toml", "deeper", "TODO")
	// Create in reverse output order and omit OpenSpec artifacts entirely.
	done := writeListTestMeta(t, ".ai/z-done/task.toml", "z-done", "TODO")
	draft := writeListTestMeta(t, ".ai/a-draft/task.toml", "a-draft", "READY")
	store := workflow.NewStore(".ai")
	doneState := taskworkflow.WorkflowRuntimeFromMeta(done)
	doneState.Planning = workflow.PlanningReady
	doneState.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemCompleted}}
	if _, err := store.Create(doneState); err != nil {
		t.Fatal(err)
	}
	draftState := taskworkflow.WorkflowRuntimeFromMeta(draft)
	draftState.Planning = workflow.PlanningDraft
	if _, err := store.Create(draftState); err != nil {
		t.Fatal(err)
	}
	assertListRows(t, [2]string{"a-draft", "DRAFT"}, [2]string{"z-done", "DONE"})
}

func TestListCommandEmpty(t *testing.T) {
	for _, populated := range []bool{false, true} {
		t.Run(fmt.Sprintf("internal-directories-%t", populated), func(t *testing.T) {
			listTestWorkspace(t)
			if populated {
				for _, name := range []string{"compile-cache", "issue", "locks", "requirements", "sessions", "tasks", "tmp", "arbitrary-directory"} {
					if err := os.MkdirAll(filepath.Join(".ai", name), 0o755); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(".ai/ordinary-file", nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			assertListRows(t)
		})
	}
}

func TestListCommandCompatibilityAndPrecedence(t *testing.T) {
	listTestWorkspace(t)
	writeListTestMeta(t, ".ai/tasks/z-legacy/tasks.toml", "z-legacy", "TODO")
	writeListTestMeta(t, ".ai/tasks/y-legacy/task.toml", "y-legacy", "TODO")
	writeListTestMeta(t, ".ai/c-filename/tasks.toml", "c-filename", "TODO")
	writeListTestMeta(t, ".ai/a-duplicate/task.toml", "a-duplicate", "TODO")
	writeListTestMeta(t, ".ai/b-filename/tasks.toml", "b-filename", "READY")
	writeListTestMeta(t, ".ai/b-filename/task.toml", "b-filename", "TODO")
	// A familiar internal directory name is a Task when it has valid metadata.
	writeListTestMeta(t, ".ai/locks/task.toml", "locks", "TODO")
	// Only the first read reconstructs absent state; legacy paths stay in place.
	if _, stderr, err := captureListCommand(t); err != nil || !strings.Contains(stderr, "最小记录") {
		t.Fatalf("initial repair: %v, %q", err, stderr)
	}
	assertListRows(t,
		[2]string{"a-duplicate", "DRAFT"},
		[2]string{"b-filename", "DRAFT"},
		[2]string{"c-filename", "DRAFT"},
		[2]string{"locks", "DRAFT"},
		[2]string{"y-legacy", "DRAFT"},
		[2]string{"z-legacy", "DRAFT"},
	)
}

// Include directories as well as file bytes to catch newly created execution
// artifacts, state changes, and event-log writes during a read-only list.
func listRuntimeSnapshot(t *testing.T) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	err := filepath.WalkDir(".ai", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			snapshot[path] = "directory"
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[path] = "file:" + string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestListCommandMetadataErrorsContinueAndDoNotExecute(t *testing.T) {
	listTestWorkspace(t)
	cases := []struct {
		id      string
		content string
		reason  string
	}{
		{"b-empty", "", "missing task id"},
		{"c-missing", "status = \"TODO\"\n", "missing task id"},
		{"d-blank", "id = \"   \"\n", "missing task id"},
		{"e-invalid", "id = \"bad/id\"\n", "task id \"bad/id\" does not match"},
		{"f-dot", "id = \".\"\n", "task id \".\" does not match"},
		{"g-parent", "id = \"..\"\n", "task id \"..\" does not match"},
		{"h-mismatch", "id = \"another-task\"\n", "task id \"another-task\" does not match directory id \"h-mismatch\""},
		// Scanner failure exercises the real reader after a successful file stat,
		// without platform-specific permissions or a production test hook.
		{"i-read-failure", "id = \"i-read-failure\"\n#" + strings.Repeat("x", bufio.MaxScanTokenSize), "bufio.Scanner: token too long"},
		{"j-not-file", "", "not a regular file"},
	}
	for _, tc := range cases {
		// The compatible filename must not mask a broken canonical task.toml.
		writeListTestMeta(t, filepath.Join(".ai", tc.id, "tasks.toml"), tc.id, "READY")
		path := filepath.Join(".ai", tc.id, "task.toml")
		if tc.id == "j-not-file" {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Valid rows on both sides of the failures prove enumeration continues.
	ready := writeListTestMeta(t, ".ai/a-valid/task.toml", "a-valid", "READY")
	state := taskworkflow.WorkflowRuntimeFromMeta(ready)
	state.Planning = workflow.PlanningReady
	state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemReady}}
	if _, err := workflow.NewStore(".ai").Create(state); err != nil {
		t.Fatal(err)
	}
	z := writeListTestMeta(t, ".ai/z-valid/task.toml", "z-valid", "TODO")
	if _, err := workflow.NewStore(".ai").Create(taskworkflow.WorkflowRuntimeFromMeta(z)); err != nil { t.Fatal(err) }
	writeListTestMeta(t, ".ai/k-runtime/task.toml", "k-runtime", "TODO")
	if err := os.WriteFile(".ai/k-runtime/state.json", []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := listRuntimeSnapshot(t)
	stdout, stderr, err := captureListCommand(t)
	if err == nil {
		t.Fatal("list must report corrupt metadata and Workflow state")
	}
	for _, tc := range cases {
		if !strings.Contains(stderr, tc.id) || !strings.Contains(stderr, tc.reason) { t.Fatalf("missing diagnostic for %s (%s): %q", tc.id, tc.reason, stderr) }
	}
	if !strings.Contains(stdout, "a-valid") || !strings.Contains(stdout, "z-valid") || !strings.Contains(stdout, "RUNTIME_ERROR") || strings.Contains(stderr, "最小记录") {
		t.Fatalf("list must retain valid rows without repairing corrupt records: %q / %q", stdout, stderr)
	}
	if after := listRuntimeSnapshot(t); !reflect.DeepEqual(before, after) {
		t.Fatal("list changed runtime files or directories; it must not start execution or update Task state")
	}
}

func TestListCommandWorkflowErrorReportsFailureWithoutOverwriting(t *testing.T) {
	listTestWorkspace(t)
	writeListTestMeta(t, ".ai/a-runtime/task.toml", "a-runtime", "TODO")
	if err := os.WriteFile(".ai/a-runtime/state.json", []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	z := writeListTestMeta(t, ".ai/z-valid/task.toml", "z-valid", "TODO")
	if _, err := workflow.NewStore(".ai").Create(taskworkflow.WorkflowRuntimeFromMeta(z)); err != nil { t.Fatal(err) }
	before := listRuntimeSnapshot(t)
	stdout, stderr, err := captureListCommand(t)
	if err == nil || !strings.Contains(stderr, "a-runtime") || !strings.Contains(stdout, "RUNTIME_ERROR") || !strings.Contains(stdout, "z-valid") {
		t.Fatalf("corrupt runtime: %q / %q / %v", stdout, stderr, err)
	}
	if after := listRuntimeSnapshot(t); !reflect.DeepEqual(before, after) {
		t.Fatal("list changed runtime files or directories")
	}
}
