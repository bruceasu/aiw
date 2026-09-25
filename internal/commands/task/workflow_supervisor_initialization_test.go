package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestWorkflowSupervisorInitializesOnlyStartWithValidMetadata(t *testing.T) {
	for _, action := range []string{"start", "status", "stop", "unknown"} {
		t.Run(action, func(t *testing.T) {
			t.Chdir(t.TempDir())
			meta := taskx.TaskMeta{ID: "task-1", WorkspaceKind: "unassigned", Delivery: "merged"}
			if err := taskx.CreateTaskMeta(taskx.TaskMetaPath(meta.ID), meta); err != nil {
				t.Fatal(err)
			}
			err := runWorkflowSupervisor([]string{"supervise", meta.ID, action})
			state, loadErr := workflow.NewStore("").Load(workflow.TaskID(meta.ID))
			if action != "start" {
				if err == nil || !os.IsNotExist(loadErr) {
					t.Fatalf("read-only/invalid action initialized state: %v, %v", err, loadErr)
				}
				return
			}
			// Merged delivery makes the real start entry point return without
			// launching an agent or performing Git delivery.
			if err != nil || loadErr != nil {
				t.Fatalf("start = %v, load = %v", err, loadErr)
			}
			if state.WriteLease != nil || len(state.Attempts) != 0 || state.Automation.Supervisor.LeaseID != "" {
				t.Fatal("initialization invented execution ownership")
			}
		})
	}
}

func TestWorkflowSupervisorPreservesMissingAndCorruptRecords(t *testing.T) {
	for _, scenario := range []string{"marker-only", "wrong-id", "corrupt-state"} {
		t.Run(scenario, func(t *testing.T) {
			t.Chdir(t.TempDir())
			const id = "task-1"
			if scenario == "marker-only" {
				dir := filepath.Join(".ai", "tasks", id)
				if err := os.MkdirAll(dir, 0o755); err != nil { t.Fatal(err) }
				if err := os.WriteFile(filepath.Join(dir, "migrated-to"), []byte("old target"), 0o644); err != nil { t.Fatal(err) }
				if err := os.MkdirAll(taskx.TaskDir(id), 0o755); err != nil { t.Fatal(err) }
			} else {
				meta := taskx.TaskMeta{ID: id, WorkspaceKind: "unassigned", Delivery: "merged"}
				if scenario == "wrong-id" { meta.ID = "another-task" }
				if err := taskx.CreateTaskMeta(taskx.TaskMetaPath(id), meta); err != nil { t.Fatal(err) }
			}
			statePath := filepath.Join(".ai", id, "state.json")
			if scenario == "corrupt-state" {
				if err := os.WriteFile(statePath, []byte("broken"), 0o644); err != nil { t.Fatal(err) }
			}
			err := runWorkflowSupervisor([]string{"supervise", id, "start"})
			if err == nil { t.Fatal("expected refusal") }
			if scenario == "marker-only" && !strings.Contains(err.Error(), "restore valid task metadata") {
				t.Fatalf("missing recovery guidance: %v", err)
			}
			data, readErr := os.ReadFile(statePath)
			if scenario == "corrupt-state" {
				if readErr != nil || string(data) != "broken" { t.Fatal("corrupt state was replaced") }
			} else if !os.IsNotExist(readErr) {
				t.Fatal("state was invented without valid metadata")
			}
		})
	}
}
