package task

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"aiw/internal/session"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestWriteLineageIsReadable(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.MkdirAll(filepath.Join("openspec", "changes", "T-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := agentLineage{TaskID: "T-1", SessionID: "S-1", ParentThread: "p", ChildThread: "c", ParentState: "handed-off", ChildState: "completed", Status: "completed"}
	if err := writeLineage("T-1", want); err != nil {
		t.Fatal(err)
	}
	b, err := readLineage("T-1")
	if err != nil {
		t.Fatal(err)
	}
	var got agentLineage
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.TaskID != want.TaskID || got.ChildThread != want.ChildThread || got.ParentState != want.ParentState || got.ChildState != want.ChildState || got.Status != want.Status {
		t.Fatalf("unexpected lineage: %+v", got)
	}
}

func TestParseAgentOptionsAllowsExplicitIsolation(t *testing.T) {
	opts, err := parseAgentOptions([]string{"--isolated"})
	if err != nil || !opts.Isolated {
		t.Fatalf("expected isolated option, got %+v err=%v", opts, err)
	}
}

func TestParseAgentOptions(t *testing.T) {
	opts, err := parseAgentOptions([]string{"--handoff", "handoff.md", "--takeover", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Handoff != "handoff.md" || !opts.Takeover || !opts.Yes {
		t.Fatalf("unexpected options: %+v", opts)
	}
}

func TestParseAgentOptionsAcceptsProviderAndModelOverrides(t *testing.T) {
	opts, err := parseAgentOptions([]string{"--provider", "copilot", "--model", "gpt-test"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Provider != "copilot" || opts.Model != "gpt-test" {
		t.Fatalf("unexpected provider/model overrides: %+v", opts)
	}
}

func TestNormalizeIDIsDeterministic(t *testing.T) {
	if got := normalizeID("Feature/Task 42"); got != "feature-task-42" {
		t.Fatalf("unexpected normalized ID: %q", got)
	}
}

func TestRunTaskTurnRejectsInvalidIDWithoutConfirmation(t *testing.T) {
	if err := runTaskAgent([]string{"turn", "invalid/id"}); err == nil {
		t.Fatal("expected invalid Task ID refusal")
	}
}

func TestResolveHandoffPrefersExplicitSource(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.MkdirAll(filepath.Join("openspec", "changes", "T-1", "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("openspec", "changes", "T-1", "artifacts", "handoff.md"), []byte("task"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("explicit.md", []byte("explicit"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := resolveHandoff("explicit.md", "", "T-1")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs("explicit.md")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidateTaskBindingsRejectsUnrelatedSession(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.MkdirAll(taskx.TaskDir("other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("other"), taskx.TaskMeta{ID: "other", Session: "S-1", Worktree: ".wt/other"}); err != nil {
		t.Fatal(err)
	}
	err = validateTaskBindings("current", taskx.TaskMeta{ID: "current", Session: "S-1", Worktree: "."})
	if err == nil {
		t.Fatal("expected unrelated session conflict")
	}
}

func TestStartManagedAttemptPersistsAttemptLease(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.MkdirAll(taskx.TaskDir("T-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := taskx.TaskMeta{ID: "T-1", Status: "TODO", Created: "2026-01-01", Updated: "2026-01-01", Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged"}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("T-1"), meta); err != nil {
		t.Fatal(err)
	}
	attemptID, err := startManagedAttempt("T-1", meta)
	if err != nil {
		t.Fatal(err)
	}
	state, err := workflow.NewStore("").Load("T-1")
	if err != nil {
		t.Fatal(err)
	}
	if state.WriteLease == nil || state.WriteLease.AttemptID != attemptID {
		t.Fatalf("expected Attempt write lease, got %#v", state.WriteLease)
	}
	if state.LastEventSequence == 0 {
		t.Fatalf("expected compatibility fallback to record workflow history: %#v", state)
	}
}

func TestRecordSessionHandoffPersistsConsumptionAudit(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	store := session.NewStore("")
	if _, err := store.Create("S-1", "S-1", tmp, "codex", "", "instructions"); err != nil {
		t.Fatal(err)
	}
	lineage := agentLineage{TaskID: "T-1", WorkItemID: "wi-0001", SessionID: "S-1", Handoff: "handoff.md", HandoffHash: "hash", HandoffStatus: "consumed", ParentThread: "parent", ChildThread: "child", ConsumedAt: "2026-09-08T00:00:00Z", ConsumerThread: "child"}
	if err := recordSessionHandoff(store, "S-1", lineage); err != nil {
		t.Fatal(err)
	}
	status, err := store.Load("S-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Task == nil || status.Task.WorkItemID != "wi-0001" || status.Task.HandoffStatus != "consumed" || status.Task.ConsumerThread != "child" {
		t.Fatalf("unexpected handoff audit: %#v", status.Task)
	}
}

func TestRunTaskAgentReusesAndCreatesTasks(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	for _, args := range [][]string{{"init"}, {"config", "user.email", "aiw@example.test"}, {"config", "user.name", "AIW Test"}} {
		if output, err := runTestCommand("git", args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if err := os.WriteFile("README.md", []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if output, err := runTestCommand("git", "add", "README.md"); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}
	if output, err := runTestCommand("git", "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, output)
	}

	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	codexName := "codex"
	contents := "#!/bin/sh\nprintf '%s\\n' '{\"type\":\"thread.started\",\"thread_id\":\"child-thread\"}'\n"
	if runtime.GOOS == "windows" {
		codexName = "codex.bat"
		contents = "@echo {\"type\":\"thread.started\",\"thread_id\":\"child-thread\"}\r\n"
	}
	codexPath := filepath.Join(binDir, codexName)
	if err := os.WriteFile(codexPath, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatal(err)
	}
	defer os.Setenv("PATH", oldPath)

	store := session.NewStore("")
	if _, err := store.Create("S-existing", "S-existing", tmp, "codex", "", "instructions"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(taskx.RuntimeTaskDir("existing"), "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := taskx.TaskMeta{ID: "existing", Status: "TODO", Created: "2026-09-08", Updated: "2026-09-08", Branch: "main", ParentBranch: "main", Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged", Session: "S-existing"}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("existing"), meta); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskx.RuntimeTaskDir("existing"), "artifacts", "handoff.md"), []byte("existing handoff"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runTaskAgent([]string{"turn", "existing", "--allow-dirty"}); err != nil {
		t.Fatal(err)
	}
	lineage, err := readLineage("existing")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lineage), "child-thread") {
		t.Fatalf("lineage did not record child thread: %s", lineage)
	}

	if err := os.WriteFile("source-handoff.md", []byte("new handoff"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runTaskAgent([]string{"turn", "created", "--handoff", "source-handoff.md", "--allow-dirty"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(taskx.RuntimeTaskDir("created"), "artifacts", "handoff.md")); err != nil {
		t.Fatal(err)
	}
	createdMeta, err := taskx.ReadTaskMeta(taskx.TaskMetaPath("created"))
	if err != nil {
		t.Fatal(err)
	}
	if createdMeta.Status != "DRAFT" {
		t.Fatalf("created Task status = %q, want DRAFT", createdMeta.Status)
	}

	if err := os.MkdirAll(filepath.Join(taskx.RuntimeTaskDir("partial"), "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("partial"), taskx.TaskMeta{ID: "partial"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskx.RuntimeTaskDir("partial"), "artifacts", "handoff.md"), []byte("partial handoff"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runTaskAgent([]string{"turn", "partial", "--allow-dirty"}); err != nil {
		t.Fatal(err)
	}
	partialMeta, err := taskx.ReadTaskMeta(taskx.TaskMetaPath("partial"))
	if err != nil {
		t.Fatal(err)
	}
	if partialMeta.Session == "" || partialMeta.Worktree == "" || partialMeta.Branch == "" {
		t.Fatalf("partial Task was not repaired: %+v", partialMeta)
	}

	if err := os.MkdirAll(taskx.TaskDir("owner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("owner"), taskx.TaskMeta{ID: "owner", Session: "S-conflict", Worktree: ".wt/owner"}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(taskx.RuntimeTaskDir("conflict"), "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("conflict"), taskx.TaskMeta{ID: "conflict", Session: "S-conflict", Worktree: "."}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskx.RuntimeTaskDir("conflict"), "artifacts", "handoff.md"), []byte("conflict handoff"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runTaskAgent([]string{"turn", "conflict", "--allow-dirty"}); err == nil {
		t.Fatal("expected unrelated Session conflict")
	}

	if _, err := store.Create("S-running", "S-running", tmp, "codex", "", "instructions"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition("S-running", session.StateRunning); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(taskx.RuntimeTaskDir("running"), "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath("running"), taskx.TaskMeta{ID: "running", Session: "S-running", Worktree: ".", WorkspaceKind: "primary"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskx.RuntimeTaskDir("running"), "artifacts", "handoff.md"), []byte("running handoff"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runTaskAgent([]string{"turn", "running", "--allow-dirty"}); err == nil {
		t.Fatal("expected running Session refusal")
	}
	failing := "#!/bin/sh\nexit 1\n"
	if runtime.GOOS == "windows" {
		failing = "@exit /b 1\r\n"
	}
	if err := os.WriteFile(codexPath, []byte(failing), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runTaskAgent([]string{"turn", "failed-create", "--handoff", "source-handoff.md", "--allow-dirty"}); err == nil {
		t.Fatal("expected failed Thread startup")
	}
	if _, err := os.Stat(taskx.TaskDir("failed-create")); !os.IsNotExist(err) {
		t.Fatalf("failed creation left Task resources: %v", err)
	}
}

func runTestCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}
