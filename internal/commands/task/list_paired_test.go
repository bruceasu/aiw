package task

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/ui"
	"aiw/internal/workflow"
)

func TestListReconstructsOnlyMissingRuntimeAndIsIdempotent(t *testing.T) {
	for _, missing := range []string{"directory", "metadata", "state", "legacy-state", "legacy-directory-files"} {
		t.Run(missing, func(t *testing.T) {
			listTestWorkspace(t)
			id := "reconstruct"
			change := "openspec/changes/" + id
			checklist := "## TODO\n\n- [x] 1.1 Historical completion is not runtime evidence.\n"
			archiveTestWrite(t, change+"/tasks.md", checklist)
			dir := ".ai/" + id
			if strings.HasPrefix(missing, "legacy-") { dir = ".ai/tasks/" + id }
			if missing == "metadata" || missing == "state" || missing == "legacy-state" {
				filename := "task.toml"
				if missing == "legacy-state" { filename = "tasks.toml" }
				meta := writeListTestMeta(t, dir+"/"+filename, id, "TODO")
				meta.Branch, meta.ParentBranch, meta.Session = "original-branch", "original-parent", "original-session"
				if err := task.WriteTaskMeta(dir+"/"+filename, meta); err != nil { t.Fatal(err) }
				if missing == "metadata" {
					state := taskworkflow.WorkflowRuntimeFromMeta(meta)
					state.Planning = workflow.PlanningReady
					state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemCompleted}}
					if _, err := workflow.NewStore(".ai").Create(state); err != nil { t.Fatal(err) }
					if err := os.Remove(dir+"/task.toml"); err != nil { t.Fatal(err) }
				}
			}
			if missing != "directory" { archiveTestWrite(t, dir+"/artifacts/preserved.txt", "existing evidence") }
			before := listRuntimeSnapshot(t)
			stdout, stderr, err := captureListCommand(t)
			if err != nil || !strings.Contains(stderr, "最小记录") || !strings.Contains(stderr, "历史状态可能丢失") { t.Fatalf("repair: %q / %q / %v", stdout, stderr, err) }
			wantStatus := "DRAFT"
			if missing == "metadata" { wantStatus = "DONE" }
			fields := strings.Fields(stdout)
			if !reflect.DeepEqual(fields, []string{id, wantStatus, change}) { t.Fatalf("row = %v", fields) }
			after := listRuntimeSnapshot(t)
			for path, content := range before { if after[path] != content { t.Fatalf("repair overwrote %s", path) } }
			state, err := workflow.LoadFromDirectory(dir, workflow.TaskID(id))
			if err != nil { t.Fatal(err) }
			if len(state.Attempts) != 0 || state.WriteLease != nil || len(state.Evidence) != 0 { t.Fatalf("repair fabricated execution: %+v", state) }
			metaPath, err := task.MetadataPathInDirectory(dir); if err != nil { t.Fatal(err) }
			meta, err := task.ReadTaskMeta(metaPath); if err != nil { t.Fatal(err) }
			if missing == "directory" || missing == "legacy-directory-files" {
				if meta.Branch != "" || meta.ParentBranch != "" || meta.Session != "" || meta.WorkspaceKind != "unassigned" { t.Fatalf("guessed bindings: %+v", meta) }
			}
			if strings.HasPrefix(missing, "legacy-") { archiveTestAbsent(t, ".ai/"+id) }
			if missing == "legacy-state" { archiveTestAbsent(t, dir+"/task.toml") }
			// Pin the timestamp as well as bytes to detect an identical rewrite.
			stamp := time.Unix(946684800, 0)
			if err := os.Chtimes(dir+"/state.json", stamp, stamp); err != nil { t.Fatal(err) }
			statBefore, err := os.Stat(dir+"/state.json"); if err != nil { t.Fatal(err) }
			second, diagnostics, err := captureListCommand(t)
			if err != nil || diagnostics != "" || second != stdout || !reflect.DeepEqual(after, listRuntimeSnapshot(t)) { t.Fatalf("second list rewrote records: %q / %q / %v", second, diagnostics, err) }
			statAfter, err := os.Stat(dir+"/state.json"); if err != nil { t.Fatal(err) }
			if !statBefore.ModTime().Equal(statAfter.ModTime()) { t.Fatal("second list rewrote state timestamp") }
			data, err := os.ReadFile(change+"/tasks.md"); if err != nil || string(data) != checklist { t.Fatalf("checklist changed: %q / %v", data, err) }
		})
	}
}

func TestListFiltersBeforeArchivedReconstruction(t *testing.T) {
	for _, root := range []string{"openspec/changes/archive", "openspec/archive"} {
		t.Run(root, func(t *testing.T) {
			listTestWorkspace(t)
			name := "2000-01-01-spec-only"
			archiveTestWrite(t, root+"/"+name+"/proposal.md", "Archived specification")
			before := listRuntimeSnapshot(t)
			stdout, stderr, err := captureListCommand(t)
			if err != nil || stdout != "" || stderr != "" || !reflect.DeepEqual(before, listRuntimeSnapshot(t)) { t.Fatalf("default repaired hidden Task: %q / %q / %v", stdout, stderr, err) }
			stdout, stderr, err = captureListCommand(t, "--all")
			if err != nil || !strings.Contains(stderr, "最小记录") { t.Fatalf("archive repair: %q / %q / %v", stdout, stderr, err) }
			if fields := strings.Fields(stdout); !reflect.DeepEqual(fields, []string{"spec-only", "DRAFT", "ARCHIVED", root+"/"+name}) { t.Fatalf("row = %v", fields) }
			archiveTestAbsent(t, ".ai/spec-only")
			state, err := workflow.LoadFromDirectory(".ai/archive/"+name, "spec-only")
			if err != nil || len(state.Attempts) != 0 || state.WriteLease != nil { t.Fatalf("reconstructed archive: %+v / %v", state, err) }
			before = listRuntimeSnapshot(t)
			_, stderr, err = captureListCommand(t, "--all")
			if err != nil || stderr != "" || !reflect.DeepEqual(before, listRuntimeSnapshot(t)) { t.Fatalf("archive repair not idempotent: %q / %v", stderr, err) }
		})
	}
}

func TestListArchivedWorkflowStatusAndSortedColumns(t *testing.T) {
	listTestWorkspace(t)
	for _, id := range []string{"z-active", "a-archived"} {
		meta := writeListTestMeta(t, ".ai/"+id+"/task.toml", id, "TODO")
		state := taskworkflow.WorkflowRuntimeFromMeta(meta)
		state.Planning = workflow.PlanningReady
		state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", State: workflow.WorkItemCompleted}}
		if _, err := workflow.NewStore(".ai").Create(state); err != nil { t.Fatal(err) }
		archiveTestWrite(t, "openspec/changes/"+id+"/proposal.md", "One missing Markdown file is not a deleted specification.")
	}
	name := "2000-01-01-a-archived"
	for _, root := range []string{".ai", "openspec/changes"} {
		if err := os.MkdirAll(root+"/archive", 0o755); err != nil { t.Fatal(err) }
		if err := os.Rename(root+"/a-archived", root+"/archive/"+name); err != nil { t.Fatal(err) }
	}
	before := listRuntimeSnapshot(t)
	stdout, stderr, err := captureListCommand(t, "--all")
	if err != nil || stderr != "" { t.Fatalf("all: %q / %v", stderr, err) }
	want := []string{"a-archived", "DONE", "ARCHIVED", "openspec/changes/archive/"+name, "z-active", "DONE", "ACTIVE", "openspec/changes/z-active"}
	if !reflect.DeepEqual(strings.Fields(stdout), want) { t.Fatalf("all rows = %q", stdout) }
	stdout, stderr, err = captureListCommand(t)
	if err != nil || stderr != "" || !reflect.DeepEqual(strings.Fields(stdout), []string{"z-active", "DONE", "openspec/changes/z-active"}) { t.Fatalf("default = %q / %q / %v", stdout, stderr, err) }
	if !reflect.DeepEqual(before, listRuntimeSnapshot(t)) { t.Fatal("complete records should remain read-only") }
	archiveTestAbsent(t, ".ai/a-archived")
}

func TestListConflictsAreDiagnosedWithoutRepair(t *testing.T) {
	for _, conflict := range []string{"duplicate-runtime", "active-and-archive", "two-archives", "different-dates", "invalid-date-suffix", "blocked-runtime-path"} {
		t.Run(conflict, func(t *testing.T) {
			listTestWorkspace(t)
			meta := writeListTestMeta(t, ".ai/conflict/task.toml", "conflict", "TODO")
			if _, err := workflow.NewStore(".ai").Create(taskworkflow.WorkflowRuntimeFromMeta(meta)); err != nil { t.Fatal(err) }
			switch conflict {
			case "duplicate-runtime": writeListTestMeta(t, ".ai/tasks/conflict/tasks.toml", "conflict", "TODO")
			case "active-and-archive":
				archiveTestWrite(t, "openspec/changes/conflict/proposal.md", "active")
				archiveTestWrite(t, "openspec/changes/archive/2000-01-01-conflict/proposal.md", "archived")
			case "two-archives":
				archiveTestWrite(t, "openspec/changes/archive/2000-01-01-conflict/proposal.md", "one")
				archiveTestWrite(t, "openspec/archive/2000-01-01-conflict/proposal.md", "two")
			case "different-dates":
				archiveTestWrite(t, "openspec/changes/archive/2000-01-01-conflict/proposal.md", "one")
				archiveTestWrite(t, "openspec/changes/archive/2000-01-02-conflict/proposal.md", "two")
			case "invalid-date-suffix":
				archiveTestWrite(t, "openspec/changes/archive/not-a-date-conflict/proposal.md", "not a match")
				archiveTestWrite(t, "openspec/changes/archive/2000-01-01-prefix-conflict/proposal.md", "different Task")
			case "blocked-runtime-path":
				archiveTestWrite(t, "openspec/changes/blocked/proposal.md", "identity")
				archiveTestWrite(t, ".ai/blocked", "not a directory")
			}
			before := listRuntimeSnapshot(t)
			stdout, stderr, err := captureListCommand(t)
			if conflict == "invalid-date-suffix" {
				if err != nil || stderr != "" || !strings.Contains(stdout, "conflict") || strings.Contains(stdout, "ARCHIVED") { t.Fatalf("suffix matched incorrectly: %q / %q / %v", stdout, stderr, err) }
			} else if err == nil || stderr == "" { t.Fatalf("conflict not reported: %q / %q / %v", stdout, stderr, err) }
			if !reflect.DeepEqual(before, listRuntimeSnapshot(t)) { t.Fatal("conflict handling mutated runtime") }
		})
	}
}

func TestListPreservesCancelledArchivedWorkflow(t *testing.T) {
	listTestWorkspace(t)
	meta := writeListTestMeta(t, ".ai/cancelled/task.toml", "cancelled", "TODO")
	state := taskworkflow.WorkflowRuntimeFromMeta(meta)
	state.Cancellation = &workflow.Cancellation{Reason: "Discarded by owner", Delivery: workflow.DeliveryDiscarded}
	state.Delivery = workflow.DeliveryDiscarded
	if _, err := workflow.NewStore(".ai").Create(state); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(".ai/archive", 0o755); err != nil { t.Fatal(err) }
	if err := os.Rename(".ai/cancelled", ".ai/archive/2000-01-01-cancelled"); err != nil { t.Fatal(err) }
	before := listRuntimeSnapshot(t)
	stdout, stderr, err := captureListCommand(t, "--all")
	if err != nil || stderr != "" || !reflect.DeepEqual(strings.Fields(stdout), []string{"cancelled", "CANCELLED", "ARCHIVED", "规格已删除"}) { t.Fatalf("cancelled archive: %q / %q / %v", stdout, stderr, err) }
	if !reflect.DeepEqual(before, listRuntimeSnapshot(t)) { t.Fatal("cancelled archive was rewritten") }
	archiveTestAbsent(t, ".ai/cancelled")
}

func TestListUnreadableMetadataIsNotMissing(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip("POSIX permissions are not Windows ACLs") }
	listTestWorkspace(t)
	path := ".ai/unreadable/task.toml"
	writeListTestMeta(t, path, "unreadable", "TODO")
	if err := os.Chmod(path, 0); err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	if _, err := os.ReadFile(path); err == nil { t.Skip("current user bypasses file permissions") } else if !os.IsPermission(err) { t.Fatal(err) }
	stdout, stderr, err := captureListCommand(t)
	if err == nil || !strings.Contains(stderr, "unreadable") || strings.Contains(stderr, "最小记录") || !strings.Contains(stdout, "RUNTIME_ERROR") { t.Fatalf("permission error: %q / %q / %v", stdout, stderr, err) }
	archiveTestAbsent(t, ".ai/unreadable/state.json")
}

func TestListRenderLongColumnsHeadersAndColors(t *testing.T) {
	rows := []taskListRow{
		{ID: "fix-task-list-discovery", Status: "DONE", Path: "openspec/changes/one"},
		{ID: "improve-requirement-management", Status: "AWAITING_AUTHORIZATION", Path: "openspec/changes/two", Archived: true},
		{ID: "requirement-artifact-generation", Status: "RUNTIME_ERROR", Path: "规格已删除"},
		{ID: "workflow-automation", Status: "FUTURE_UNKNOWN_STATUS", Path: "path with spaces"},
	}
	ansi := regexp.MustCompile("\x1b\\[[0-9;]*m")
	for _, all := range []bool{false, true} {
		for _, interactive := range []bool{false, true} {
			for _, colored := range []bool{false, true} {
				t.Run(fmt.Sprintf("all-%t/tty-%t/color-%t", all, interactive, colored), func(t *testing.T) {
					var out bytes.Buffer
					if err := renderTaskList(&out, rows, all, interactive, colored); err != nil { t.Fatal(err) }
					raw := out.String()
					if !interactive || !colored { if strings.Contains(raw, "\x1b") { t.Fatalf("unexpected ANSI: %q", raw) } }
					plain := ansi.ReplaceAllString(raw, "")
					lines := strings.Split(strings.TrimSuffix(plain, "\n"), "\n")
					if interactive {
						want := []string{"TASK", "STATUS", "PATH"}; if all { want = []string{"TASK", "STATUS", "ARCHIVE", "PATH"} }
						if !reflect.DeepEqual(strings.Fields(lines[0]), want) { t.Fatalf("header: %q", lines[0]) }
						lines = lines[1:]
					}
					if len(lines) != len(rows) { t.Fatalf("rows: %q", plain) }
					statusStart, pathStart, archiveStart := -1, -1, -1
					for i, row := range rows {
						line := lines[i]
						if !strings.HasPrefix(line, row.ID+"  ") || !strings.HasSuffix(line, row.Path) { t.Fatalf("truncated row: %q", line) }
						s, p := strings.Index(line, row.Status), strings.Index(line, row.Path)
						if i == 0 { statusStart, pathStart = s, p }
						if s != statusStart || p != pathStart { t.Fatalf("unaligned row: %q", line) }
						if all {
							label := "ACTIVE"; if row.Archived { label = "ARCHIVED" }
							a := strings.Index(line, label)
							if i == 0 { archiveStart = a }; if a != archiveStart { t.Fatalf("archive column: %q", line) }
						}
					}
					if interactive && colored {
						for _, cell := range []string{"\x1b[32mDONE\x1b[0m  ", "\x1b[33mAWAITING_AUTHORIZATION\x1b[0m  ", "\x1b[31mRUNTIME_ERROR\x1b[0m  "} { if !strings.Contains(raw, cell) { t.Fatalf("missing reset before padding %q: %q", cell, raw) } }
						if all && !strings.Contains(raw, "\x1b[90mARCHIVED\x1b[0m  ") { t.Fatalf("archive color: %q", raw) }
						var uncolored bytes.Buffer
						if err := renderTaskList(&uncolored, rows, all, interactive, false); err != nil { t.Fatal(err) }
						if plain != uncolored.String() { t.Fatal("ANSI changed visible layout") }
					}
				})
			}
		}
	}
}

func TestListRenderStatePaletteAndEmptyOutput(t *testing.T) {
	for status, color := range map[string]string{"DONE":"32", "DRAFT":"90", "CANCELLED":"90", "READY":"36", "RUNNING":"36", "NEEDS_DECISION":"33", "AWAITING_VERIFICATION":"33", "BLOCKED":"31", "FAILED":"31", "RUNTIME_ERROR":"31", "UNKNOWN":""} {
		var out bytes.Buffer
		if err := renderTaskList(&out, []taskListRow{{ID:"task", Status:status, Path:"path"}}, false, true, true); err != nil { t.Fatal(err) }
		if color == "" { if strings.Contains(out.String(), "\x1b") { t.Fatalf("unknown status colored: %q", out.String()) } } else if !strings.Contains(out.String(), "\x1b["+color+"m"+status+"\x1b[0m") { t.Fatalf("palette: %q", out.String()) }
	}
	for _, all := range []bool{false, true} {
		var out bytes.Buffer
		if err := renderTaskList(&out, nil, all, true, true); err != nil || out.Len() != 0 { t.Fatalf("empty header: %q / %v", out.String(), err) }
	}
}

func TestListRedirectedAndDisabledColor(t *testing.T) {
	listTestWorkspace(t)
	for _, env := range []struct { name, noColor, term string }{{"redirect", "", "xterm-256color"}, {"no-color", "1", "xterm-256color"}, {"dumb", "", "dumb"}, {"unknown", "", "unknown-terminal"}} {
		t.Run(env.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", env.noColor); t.Setenv("TERM", env.term)
			archiveTestWrite(t, "openspec/changes/plain/proposal.md", "spec")
			stdout, _, err := captureListCommand(t, "--all")
			if err != nil || strings.Contains(stdout, "\x1b") || strings.Contains(stdout, "TASK") || !reflect.DeepEqual(strings.Fields(stdout), []string{"plain", "DRAFT", "ACTIVE", "openspec/changes/plain"}) { t.Fatalf("redirected: %q / %v", stdout, err) }
			reader, writer, err := os.Pipe(); if err != nil { t.Fatal(err) }; defer reader.Close(); defer writer.Close()
			terminal := ui.NewTerminal(writer)
			if terminal.Interactive() || terminal.ColorEnabled() { t.Fatal("pipe detected as color terminal") }
			if ui.NewTerminal(&bytes.Buffer{}).ColorEnabled() { t.Fatal("unknown writer detected as color terminal") }
			// A real TTY is optional: when available, verify environment suppression
			// independently of the redirected-output fallback above.
			if env.noColor != "" || env.term == "dumb" {
				if ui.NewTerminal(os.Stdout).ColorEnabled() { t.Fatal("explicit color suppression ignored") }
			}
		})
	}
}

func TestListArguments(t *testing.T) {
	listTestWorkspace(t)
	for _, args := range [][]string{{"--unknown"}, {"extra"}, {"--all", "extra"}, {"--all", "--all"}} {
		_, _, err := captureListCommand(t, args...)
		if err == nil || !strings.Contains(err.Error(), "usage: aiw list") { t.Fatalf("args %v: %v", args, err) }
	}
	for _, flag := range []string{"--help", "-h"} {
		stdout, stderr, err := captureListCommand(t, flag)
		if err != nil || stderr != "" || !strings.Contains(stdout, "--all") { t.Fatalf("help: %q / %q / %v", stdout, stderr, err) }
	}
}
