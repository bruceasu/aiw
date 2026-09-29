package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	sessioncmd "aiw/internal/commands/session"
	"aiw/internal/session"
	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow"
)

// The test executable doubles as a local OpenSpec process. The sentinel and
// exact argument shape prevent a normal test invocation from taking this path.
// This keeps backend coverage portable without a shell, downloads or a build.
func init() {
	mode := os.Getenv("AIW_TEST_PAIRED_ARCHIVE_HELPER")
	if mode == "" { return }
	if len(os.Args) == 2 && os.Args[1] == "--version" { os.Exit(0) }
	if len(os.Args) != 4 || os.Args[1] != "archive" || os.Args[2] != "--yes" { return }
	id := os.Args[3]
	if id != "paired" { os.Exit(91) }
	name := task.Today() + "-" + id
	if mode == "wrong-date" { name = "2000-01-01-" + id }
	if mode == "no-move" { os.Exit(0) }
	target := filepath.Join("openspec", "changes", "archive", name)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { os.Exit(92) }
	if err := os.Rename(filepath.Join("openspec", "changes", id), target); err != nil { os.Exit(93) }
	if mode == "move-then-fail" { os.Exit(94) }
	os.Exit(0)
}

func archiveTestWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { t.Fatal(err) }
}

func archiveTestTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil { return err }
		rel, err := filepath.Rel(root, path)
		if err != nil { return err }
		if entry.IsDir() { result[rel] = "dir"; return nil }
		data, err := os.ReadFile(path)
		if err != nil { return err }
		result[rel] = "file:" + string(data)
		return nil
	})
	if err != nil { t.Fatal(err) }
	return result
}

func archiveTestAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) { t.Fatalf("expected absent %s, got %v", path, err) }
}

func archiveTestFixture(t *testing.T) *session.Store {
	t.Helper()
	listTestWorkspace(t)
	root, err := filepath.Abs(".ai")
	if err != nil { t.Fatal(err) }
	t.Setenv("AIW_SESSION_ROOT", root)
	meta := writeListTestMeta(t, ".ai/paired/task.toml", "paired", "CANCELLED")
	meta.Session, meta.Delivery = "different-session", "discarded"
	if err := task.WriteTaskMeta(".ai/paired/task.toml", meta); err != nil { t.Fatal(err) }
	if _, err := workflow.NewStore(".ai").Create(taskworkflow.WorkflowRuntimeFromMeta(meta)); err != nil { t.Fatal(err) }
	archiveTestWrite(t, "openspec/changes/paired/tasks.md", "## TODO\n\n- [x] 1.1 Preserve the records.\n")
	archiveTestWrite(t, ".ai/paired/artifacts/evidence.txt", "original evidence\n")
	store := session.NewStore(root)
	status, err := store.Create("different-session", "Original title", ".", "codex", "", "Original instructions")
	if err != nil { t.Fatal(err) }
	status.Session.State = session.StatePaused
	status.Backend.ThreadID = "original-thread"
	status.Task = &session.ManagedExecutionRef{TaskID: "paired", WorkItemID: "wi-0001", AttemptID: "original-attempt"}
	status.Result.FinalOutputFile = filepath.Join(root, "sessions", "different-session", "outputs", "final.txt")
	if err := store.Save(status); err != nil { t.Fatal(err) }
	archiveTestWrite(t, status.Result.FinalOutputFile, "original output\n")
	if err := store.WriteArtifact("different-session", "handoff.md", []byte("original handoff\n")); err != nil { t.Fatal(err) }
	// A same-name Session is unrelated and must remain active.
	if _, err := store.Create("paired", "Unrelated", ".", "codex", "", ""); err != nil { t.Fatal(err) }
	return store
}

func archiveTestBackend(t *testing.T, mode string) {
	t.Helper()
	bin, err := os.Executable()
	if err != nil { t.Fatal(err) }
	t.Setenv("AIW_TEST_PAIRED_ARCHIVE_HELPER", mode)
	t.Setenv("AIW_OPENSPEC_BIN", bin)
}

func archiveTestRun(t *testing.T, backend string) (string, error) {
	t.Helper()
	_, stderr, err := captureTaskOutput(t, func() error {
		return DispatchTopLevel("archive", []string{"paired", "--backend", backend, "--force"})
	})
	return stderr, err
}

func TestArchivePrefersFDAndMovesItWithNativeTask(t *testing.T) {
	listTestWorkspace(t)
	id := "native"
	meta := writeListTestMeta(t, task.TaskMetaPath(id), id, "TODO")
	archiveTestWrite(t, task.FeatureDesignPath(id), "## Design Readiness\n\nFD_APPLIED\n\n## Work Items\n- [ ] 1.1 FD work\n")
	archiveTestWrite(t, filepath.Join(task.TaskDir(id), "tasks.md"), "- [ ] 9.9 Stale change work\n")
	if err := syncArchiveWorkflow(id, meta, task.TaskDir(id)); err != nil { t.Fatal(err) }
	state, err := workflow.NewStore("").Load(workflow.TaskID(id))
	if err != nil || len(state.WorkItems) != 1 || state.WorkItems[0].Checklist.Item != "1.1" { t.Fatalf("archive plan source: %#v %v", state.WorkItems, err) }
	plan, err := prepareTaskArchive(id)
	if err != nil { t.Fatal(err) }
	want := task.FeatureDesignArchivePath(plan.name)
	found := false
	for _, move := range plan.moves {
		if move.source == task.FeatureDesignPath(id) && move.target == want { found = true }
	}
	if !found { t.Fatalf("FD archive move missing: %#v", plan.moves) }
}

func TestPairedArchiveBackendsPreserveThreeTreesAndRetry(t *testing.T) {
	for _, backend := range []string{"native", "auto", "openspec", "auto-fallback"} {
		t.Run(backend, func(t *testing.T) {
			store := archiveTestFixture(t)
			selected := backend
			if backend == "auto-fallback" { selected = "auto" } else if backend != "native" { archiveTestBackend(t, "success") }
			name := task.Today() + "-paired"
			sources := []string{"openspec/changes/paired", ".ai/paired", ".ai/sessions/different-session"}
			targets := []string{"openspec/changes/archive/" + name, ".ai/archive/" + name, ".ai/sessions/archive/" + name + "/different-session"}
			before := make([]map[string]string, len(sources))
			for i, source := range sources { before[i] = archiveTestTree(t, source) }
			unrelated := archiveTestTree(t, ".ai/sessions/paired")
			stderr, err := archiveTestRun(t, selected)
			if err != nil { t.Fatalf("archive: %v / %s", err, stderr) }
			if backend == "auto-fallback" && !strings.Contains(stderr, "native fallback") { t.Fatalf("missing fallback: %s", stderr) }
			for i, target := range targets {
				archiveTestAbsent(t, sources[i])
				if !reflect.DeepEqual(before[i], archiveTestTree(t, target)) { t.Fatalf("contents changed at %s", target) }
			}
			if !reflect.DeepEqual(unrelated, archiveTestTree(t, ".ai/sessions/paired")) { t.Fatal("unrelated Session changed") }
			status, err := store.Load("different-session")
			if err != nil || status.Session.State != session.StatePaused || status.Backend.ThreadID != "original-thread" { t.Fatalf("archived Session: %+v / %v", status, err) }
			if got, err := store.ReadText("different-session", status.Result.FinalOutputFile); err != nil || got != "original output\n" { t.Fatalf("absolute output mapping: %q / %v", got, err) }
			beforeRetry := listRuntimeSnapshot(t)
			if _, err := archiveTestRun(t, selected); err != nil { t.Fatalf("retry: %v", err) }
			if !reflect.DeepEqual(beforeRetry, listRuntimeSnapshot(t)) { t.Fatal("retry changed archived runtime") }
		})
	}
}

func TestPairedArchiveRepairsCleanedMergedTaskAcrossBackends(t *testing.T) {
	for _, backend := range []string{"native", "openspec"} {
		t.Run(backend, func(t *testing.T) {
			archiveTestFixture(t)
			meta, err := task.ReadTaskMeta(".ai/paired/task.toml")
			if err != nil {
				t.Fatal(err)
			}
			meta.Status = string(workflow.TaskDone)
			meta.Delivery = string(workflow.DeliveryMerged)
			meta.Worktree = ""
			meta.WorkspaceKind = "unassigned"
			if err := task.WriteTaskMeta(".ai/paired/task.toml", meta); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(".ai/paired/task.toml", append(mustReadArchiveTestFile(t, ".ai/paired/task.toml"), []byte("operator_tag = \"keep-me\"\n")...), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := workflow.NewStore(".ai").SetDelivery("paired", workflow.DeliveryMerged); err != nil {
				t.Fatal(err)
			}
			if backend == "openspec" {
				archiveTestBackend(t, "success")
			}
			if stderr, err := archiveTestRun(t, backend); err != nil {
				t.Fatalf("archive: %v / %s", err, stderr)
			}
			name := task.Today() + "-paired"
			archivedMeta := filepath.Join(".ai", "archive", name, "task.toml")
			content := string(mustReadArchiveTestFile(t, archivedMeta))
			for _, field := range []string{"status = \"DONE\"", "delivery = \"merged\"", "workspace_kind = \"unassigned\"", "operator_tag = \"keep-me\""} {
				if !strings.Contains(content, field) {
					t.Fatalf("archived metadata missing %q: %s", field, content)
				}
			}
			beforeRetry := listRuntimeSnapshot(t)
			if _, err := archiveTestRun(t, backend); err != nil {
				t.Fatalf("retry: %v", err)
			}
			if !reflect.DeepEqual(beforeRetry, listRuntimeSnapshot(t)) {
				t.Fatal("retry changed repaired archive")
			}
		})
	}
}

func mustReadArchiveTestFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestPairedArchiveMoveFailureRestoresInReverseOrder(t *testing.T) {
	for _, backend := range []string{"native", "openspec"} {
		for failAt := 1; failAt <= 3; failAt++ {
			if backend == "openspec" && failAt == 1 { continue } // Delegated failures have their own cases below.
			t.Run(fmt.Sprintf("%s-step-%d", backend, failAt), func(t *testing.T) {
				archiveTestFixture(t)
				if backend == "openspec" { archiveTestBackend(t, "success") }
				sources := []string{"openspec/changes/paired", ".ai/paired", ".ai/sessions/different-session"}
				before := make([]map[string]string, len(sources))
				for i, source := range sources { before[i] = archiveTestTree(t, source) }
				oldRename := archiveRename
				t.Cleanup(func() { archiveRename = oldRename })
				var restored []string
				archiveRename = func(source, target string) error {
					if sameArchivePath(source, sources[failAt-1]) { return errors.New("injected move failure") }
					for _, active := range sources { if sameArchivePath(target, active) { restored = append(restored, active) } }
					return os.Rename(source, target)
				}
				if _, err := archiveTestRun(t, backend); err == nil || !strings.Contains(err.Error(), "injected move failure") { t.Fatalf("failure: %v", err) }
				var want []string
				for i := failAt-2; i >= 0; i-- { want = append(want, sources[i]) }
				if !reflect.DeepEqual(restored, want) { t.Fatalf("restore order %v, want %v", restored, want) }
				for i, source := range sources { if !reflect.DeepEqual(before[i], archiveTestTree(t, source)) { t.Fatalf("source changed: %s", source) } }
				archiveRename = oldRename
				if _, err := archiveTestRun(t, backend); err != nil { t.Fatalf("retry after repair: %v", err) }
			})
		}
	}
}

func TestPairedArchiveCompensationFailureReportsSurvivingPaths(t *testing.T) {
	archiveTestFixture(t)
	oldRename := archiveRename
	t.Cleanup(func() { archiveRename = oldRename })
	archiveRename = func(source, target string) error {
		if sameArchivePath(source, ".ai/sessions/different-session") { return errors.New("session move failed") }
		if sameArchivePath(target, ".ai/paired") { return errors.New("runtime restore failed") }
		return os.Rename(source, target)
	}
	_, err := archiveTestRun(t, "native")
	if err == nil { t.Fatal("expected compensation failure") }
	for _, part := range []string{"session move failed", "runtime restore failed", "inspect both paths", "retry archive", task.Today()+"-paired"} {
		if !strings.Contains(err.Error(), part) { t.Fatalf("missing %q: %v", part, err) }
	}
	archiveTestAbsent(t, ".ai/paired")
	for _, path := range []string{"openspec/changes/paired/tasks.md", ".ai/archive/"+task.Today()+"-paired/artifacts/evidence.txt", ".ai/sessions/different-session/status.json"} {
		if _, err := os.Stat(path); err != nil { t.Fatalf("lost %s: %v", path, err) }
	}
}

func TestPairedArchiveVerificationFailureAlsoRestoresSession(t *testing.T) {
	archiveTestFixture(t)
	oldRename := archiveRename
	t.Cleanup(func() { archiveRename = oldRename })
	var restored []string
	archiveRename = func(source, target string) error {
		if err := os.Rename(source, target); err != nil { return err }
		if sameArchivePath(source, ".ai/sessions/different-session") {
			// Introduce occupancy only after all three moves, before verification.
			archiveTestWrite(t, ".ai/locks/paired.lock", "new writer")
		}
		for _, path := range []string{"openspec/changes/paired", ".ai/paired", ".ai/sessions/different-session"} {
			if sameArchivePath(target, path) { restored = append(restored, path) }
		}
		return nil
	}
	if _, err := archiveTestRun(t, "native"); err == nil { t.Fatal("verification must fail") }
	want := []string{".ai/sessions/different-session", ".ai/paired", "openspec/changes/paired"}
	if !reflect.DeepEqual(restored, want) { t.Fatalf("restore order %v, want %v", restored, want) }
	for _, path := range want { if _, err := os.Stat(path); err != nil { t.Fatal(err) } }
}

func TestPairedArchiveHistoricalFailureDoesNotRestorePriorArchive(t *testing.T) {
	archiveTestFixture(t)
	name := "2000-01-01-paired"
	if err := os.MkdirAll("openspec/archive", 0o755); err != nil { t.Fatal(err) }
	if err := os.Rename("openspec/changes/paired", "openspec/archive/"+name); err != nil { t.Fatal(err) }
	before := archiveTestTree(t, "openspec/archive/"+name)
	oldRename := archiveRename
	t.Cleanup(func() { archiveRename = oldRename })
	archiveRename = func(source, target string) error {
		if sameArchivePath(source, ".ai/sessions/different-session") { return errors.New("session repair failed") }
		return os.Rename(source, target)
	}
	if _, err := archiveTestRun(t, "native"); err == nil { t.Fatal("expected repair failure") }
	archiveTestAbsent(t, "openspec/changes/paired")
	if !reflect.DeepEqual(before, archiveTestTree(t, "openspec/archive/"+name)) { t.Fatal("previous archive changed") }
	if _, err := os.Stat(".ai/paired/task.toml"); err != nil { t.Fatal(err) }
}

func TestPairedArchiveEligibilityAndTargetsRemainProtected(t *testing.T) {
	for _, problem := range []string{"nonterminal", "unknown-workspace", "undiscarded", "finalize-primary", "runtime-target", "spec-target", "missing-meta"} {
		t.Run(problem, func(t *testing.T) {
			archiveTestFixture(t)
			meta, err := task.ReadTaskMeta(".ai/paired/task.toml"); if err != nil { t.Fatal(err) }
			switch problem {
			case "nonterminal": meta.Status = "TODO"
			case "unknown-workspace": meta.WorkspaceKind, meta.Worktree = "unknown", ""
			case "undiscarded": meta.Delivery = "unmanaged"
			case "runtime-target": archiveTestWrite(t, ".ai/archive/"+task.Today()+"-paired/keep", "existing target")
			case "spec-target": archiveTestWrite(t, "openspec/changes/archive/"+task.Today()+"-paired/keep", "existing target")
			}
			if err := task.WriteTaskMeta(".ai/paired/task.toml", meta); err != nil { t.Fatal(err) }
			if problem == "missing-meta" { if err := os.Remove(".ai/paired/task.toml"); err != nil { t.Fatal(err) } }
			before := listRuntimeSnapshot(t)
			change := archiveTestTree(t, "openspec/changes/paired")
			args := []string{"paired", "--backend", "native", "--force"}
			if problem == "finalize-primary" { args = append(args, "--finalize") }
			_, _, err = captureTaskOutput(t, func() error { return DispatchTopLevel("archive", args) })
			if err == nil { t.Fatal("unsafe archive was allowed") }
			if !reflect.DeepEqual(before, listRuntimeSnapshot(t)) || !reflect.DeepEqual(change, archiveTestTree(t, "openspec/changes/paired")) { t.Fatal("rejected archive changed files") }
		})
	}
}

func TestPairedArchiveDelegatedFailure(t *testing.T) {
	for _, mode := range []string{"move-then-fail", "wrong-date", "no-move"} {
		t.Run(mode, func(t *testing.T) {
			archiveTestFixture(t)
			archiveTestBackend(t, mode)
			before := archiveTestTree(t, "openspec/changes/paired")
			runtimeBefore := listRuntimeSnapshot(t)
			if _, err := archiveTestRun(t, "openspec"); err == nil { t.Fatal("delegate must not report paired success") }
			if !reflect.DeepEqual(before, archiveTestTree(t, "openspec/changes/paired")) || !reflect.DeepEqual(runtimeBefore, listRuntimeSnapshot(t)) { t.Fatal("failed delegate lost or changed original records") }
		})
	}
}

func TestPairedArchiveSessionPreflightRefusesBeforeMoving(t *testing.T) {
	for _, problem := range []string{"running", "unfinished-time", "task-mismatch", "corrupt", "missing-status", "duplicate-session", "task-lock", "session-lock", "target-conflict", "duplicate-active-binding", "duplicate-legacy-binding", "duplicate-archived-binding"} {
		t.Run(problem, func(t *testing.T) {
			store := archiveTestFixture(t)
			status, err := store.Load("different-session")
			if err != nil { t.Fatal(err) }
			switch problem {
			case "running": status.Session.State = session.StateRunning
			case "unfinished-time": status.Execution.LastStartedAt = "2026-01-01T00:00:00Z"
			case "task-mismatch": status.Task.TaskID = "other"
			}
			if err := store.Save(status); err != nil { t.Fatal(err) }
			switch problem {
			case "corrupt": archiveTestWrite(t, ".ai/sessions/different-session/status.json", "{")
			case "missing-status": if err := os.Remove(".ai/sessions/different-session/status.json"); err != nil { t.Fatal(err) }
			case "duplicate-session":
				data, err := os.ReadFile(".ai/sessions/different-session/status.json"); if err != nil { t.Fatal(err) }
				archiveTestWrite(t, ".ai/archive/different-session/status.json", string(data))
			case "task-lock": archiveTestWrite(t, ".ai/locks/paired.lock", "occupied")
			case "session-lock": archiveTestWrite(t, ".ai/locks/different-session.lock", "occupied")
			case "target-conflict": archiveTestWrite(t, ".ai/sessions/archive/"+task.Today()+"-paired/different-session/keep", "do not overwrite")
			case "duplicate-active-binding", "duplicate-legacy-binding", "duplicate-archived-binding":
				path := ".ai/other/task.toml"
				if problem == "duplicate-legacy-binding" { path = ".ai/tasks/other/tasks.toml" }
				if problem == "duplicate-archived-binding" { path = ".ai/archive/2000-01-01-other/task.toml" }
				meta := writeListTestMeta(t, path, "other", "TODO"); meta.Session = "different-session"
				if err := task.WriteTaskMeta(path, meta); err != nil { t.Fatal(err) }
			}
			before := listRuntimeSnapshot(t)
			change := archiveTestTree(t, "openspec/changes/paired")
			if _, err := archiveTestRun(t, "native"); err == nil { t.Fatal("preflight should fail") }
			if !reflect.DeepEqual(before, listRuntimeSnapshot(t)) || !reflect.DeepEqual(change, archiveTestTree(t, "openspec/changes/paired")) { t.Fatal("preflight failure mutated records") }
		})
	}
}

func TestPairedArchiveTaskOccupancyRefusesBeforeMoving(t *testing.T) {
	for _, problem := range []string{"lease", "created-attempt", "running-attempt", "paused-attempt", "running-item", "leased-item"} {
		t.Run(problem, func(t *testing.T) {
			archiveTestFixture(t)
			state, err := workflow.LoadFromDirectory(".ai/paired", "paired")
			if err != nil { t.Fatal(err) }
			state.WorkItems = []workflow.WorkItem{{ID: "busy-item", State: workflow.WorkItemReady}}
			wantError := "unfinished attempt"
			switch problem {
			case "lease":
				state.WorkItems[0].State = workflow.WorkItemRunning
				state.Attempts = []workflow.Attempt{{ID: "busy", WorkItemID: "busy-item", Workspace: ".", State: workflow.AttemptRunning}}
				state.WriteLease = &workflow.WriteLease{AttemptID: "busy", Workspace: "."}
				wantError = "unreleased write lease"
			case "created-attempt", "running-attempt", "paused-attempt":
				state.Attempts = []workflow.Attempt{{ID: "busy", WorkItemID: "busy-item", Workspace: ".", State: workflow.AttemptState(strings.TrimSuffix(problem, "-attempt"))}}
			default:
				state.WorkItems[0].State = workflow.WorkItemState(strings.TrimSuffix(problem, "-item"))
				wantError = "is occupied"
			}
			data, err := json.Marshal(state); if err != nil { t.Fatal(err) }
			archiveTestWrite(t, ".ai/paired/state.json", string(data))
			before := listRuntimeSnapshot(t)
			if _, err := archiveTestRun(t, "native"); err == nil || !strings.Contains(err.Error(), wantError) { t.Fatalf("occupancy check: %v, want %s", err, wantError) }
			if !reflect.DeepEqual(before, listRuntimeSnapshot(t)) { t.Fatal("occupied records changed") }
			if _, err := os.Stat("openspec/changes/paired/tasks.md"); err != nil { t.Fatal(err) }
		})
	}
}

func TestPairedArchiveMissingSessionAndSpecification(t *testing.T) {
	for _, missing := range []string{"session", "specification", "unbound"} {
		t.Run(missing, func(t *testing.T) {
			store := archiveTestFixture(t)
			switch missing {
			case "session": if err := store.Delete("different-session"); err != nil { t.Fatal(err) }
			case "specification": if err := os.RemoveAll("openspec/changes/paired"); err != nil { t.Fatal(err) }
			case "unbound":
				meta, err := task.ReadTaskMeta(".ai/paired/task.toml"); if err != nil { t.Fatal(err) }; meta.Session = ""
				if err := task.WriteTaskMeta(".ai/paired/task.toml", meta); err != nil { t.Fatal(err) }
			}
			stderr, err := archiveTestRun(t, "native")
			if err != nil { t.Fatal(err) }
			if missing == "session" && strings.Contains(stderr, "会话记录缺失") { t.Fatalf("optional Session produced a missing-record warning: %s", stderr) }
			if missing == "specification" && !strings.Contains(stderr, "规格已删除") { t.Fatalf("missing warning: %s", stderr) }
			archiveTestAbsent(t, ".ai/paired")
			if missing == "session" || missing == "unbound" { archiveTestAbsent(t, ".ai/sessions/archive/"+task.Today()+"-paired/different-session") }
		})
	}
}

func TestPairedArchiveHistoricalRepairAndSessionReadOnlyCLI(t *testing.T) {
	for _, root := range []string{"openspec/changes/archive", "openspec/archive"} {
		for _, legacySession := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/legacy-session-%t", root, legacySession), func(t *testing.T) {
				store := archiveTestFixture(t)
				name := "2000-01-01-paired"
				if err := os.MkdirAll(root, 0o755); err != nil { t.Fatal(err) }
				if err := os.Rename("openspec/changes/paired", filepath.Join(root, name)); err != nil { t.Fatal(err) }
				if legacySession {
					if err := os.MkdirAll(".ai/archive", 0o755); err != nil { t.Fatal(err) }
					if err := os.Rename(".ai/sessions/different-session", ".ai/archive/different-session"); err != nil { t.Fatal(err) }
				}
				before := listRuntimeSnapshot(t)
				stdout, stderr, err := captureListCommand(t)
				if err != nil || stdout != "" || stderr != "" { t.Fatalf("historical default: %q / %q / %v", stdout, stderr, err) }
				stdout, stderr, err = captureListCommand(t, "--all")
				if err != nil || stderr != "" || !strings.Contains(stdout, "ARCHIVED") || !strings.Contains(stdout, root+"/"+name) { t.Fatalf("historical all: %q / %q / %v", stdout, stderr, err) }
				if !reflect.DeepEqual(before, listRuntimeSnapshot(t)) { t.Fatal("list moved historical runtime") }
				if legacySession { assertArchivedSessionCLI(t, store) }
				if _, err := archiveTestRun(t, "native"); err != nil { t.Fatal(err) }
				archiveTestAbsent(t, ".ai/paired")
				if _, err := os.Stat(".ai/archive/"+name+"/task.toml"); err != nil { t.Fatal(err) }
				assertArchivedSessionCLI(t, store)
			})
		}
	}
}

func assertArchivedSessionCLI(t *testing.T, store *session.Store) {
	t.Helper()
	before := listRuntimeSnapshot(t)
	for _, tc := range []struct { args []string; want string }{
		{[]string{"get", "paired"}, "original-thread"},
		{[]string{"status", "paired"}, "original-attempt"},
		{[]string{"memory", "show", "paired"}, "Session Memory"},
		{[]string{"handoff", "show", "paired"}, "original handoff"},
	} {
		stdout, _, err := captureTaskOutput(t, func() error { return sessioncmd.Dispatch(tc.args) })
		if err != nil || !strings.Contains(stdout, tc.want) { t.Fatalf("%v: %q / %v", tc.args, stdout, err) }
	}
	stdout, _, err := captureTaskOutput(t, func() error { return sessioncmd.Dispatch([]string{"list"}) })
	if err != nil || strings.Contains(stdout, "different-session") || strings.Contains(stdout, "archive") { t.Fatalf("session list: %q / %v", stdout, err) }
	for _, args := range [][]string{{"finish", "paired"}, {"archive", "paired"}, {"delete", "paired", "--yes"}, {"memory", "append", "paired", "forbidden"}, {"handoff", "paired"}} {
		if err := sessioncmd.Dispatch(args); err == nil || !strings.Contains(err.Error(), "read-only") { t.Fatalf("write %v: %v", args, err) }
	}
	status, err := store.Load("different-session")
	if err != nil { t.Fatal(err) }
	if err := session.RequireRunnable(status); err == nil { t.Fatal("archived paused Session is runnable") }
	for _, write := range []func() error{
		func() error { return store.Save(status) },
		func() error { return store.SavePrompt("different-session", 2, "test", "forbidden") },
		func() error { return store.AppendEvent("different-session", "forbidden") },
		func() error { return store.WriteArtifact("different-session", "new.txt", []byte("forbidden")) },
	} { if err := write(); err == nil || !strings.Contains(err.Error(), "read-only") { t.Fatalf("archived write: %v", err) } }
	if !reflect.DeepEqual(before, listRuntimeSnapshot(t)) { t.Fatal("read-only Session operations changed records") }
}
