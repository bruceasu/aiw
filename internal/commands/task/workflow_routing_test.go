package task

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"aiw/internal/session"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func routingTestTask(t *testing.T) (taskx.TaskMeta, *workflow.Store) {
	t.Helper()
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(root); err != nil { t.Fatal(err) }
	t.Setenv("AIW_ROOT", root)
	t.Setenv("AIW_LLM_PROVIDER", "routing-test-unsupported")
	t.Setenv("AIW_LLM_MODEL", "original-model")
	if err := os.WriteFile("aiw.toml", []byte("[ai]\nprovider = \"routing-test-unsupported\"\n"), 0600); err != nil { t.Fatal(err) }
	meta := taskx.TaskMeta{ID: "task-1", Status: "TODO", Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged", Session: "session-1"}
	if _, err := session.NewStore("").Create(meta.Session, "fixture", root, "codex", "", "fixture"); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(taskx.TaskDir(meta.ID), 0700); err != nil { t.Fatal(err) }
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath(meta.ID), meta); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(taskx.TaskDir(meta.ID), "tasks.md"), []byte("- [ ] 1.1 Implement routing\n"), 0600); err != nil { t.Fatal(err) }
	_, store, err := compatibleWorkflow(meta.ID)
	if err != nil { t.Fatal(err) }
	return meta, store
}

func TestRecommendationFailurePersistsDefaults(t *testing.T) {
	meta, store := routingTestTask(t)
	if _, err := recommendRoutingProfiles("."); err == nil { t.Fatal("unsupported provider must fail recommendation") }
	if err := runWorkflowCommand([]string{"recommend-routing", meta.ID}); err != nil { t.Fatal(err) }
	plan, err := store.LoadRoutingPlan(workflow.TaskID(meta.ID))
	if err != nil { t.Fatal(err) }
	if plan.Source != "defaults" || !reflect.DeepEqual(plan.Profiles, workflow.DefaultRoutingProfiles()) { t.Fatalf("fallback plan = %+v", plan) }
	if !reflect.DeepEqual(plan.Compile, workflow.CompilePlanForWorkspace(".")) { t.Fatalf("fallback compile plan = %+v", plan.Compile) }
}

func TestSupervisedAdvanceReusesSnapshotAfterConfigChange(t *testing.T) {
	meta, store := routingTestTask(t)
	state, err := advanceWorkflow(meta.ID, meta, store, true, "", "")
	if err != nil { t.Fatal(err) }
	request := state.Automation.PreparedRequest
	if request == nil || request.AISelection == nil { t.Fatalf("missing prepared selection: %+v", request) }
	want := *request.AISelection
	if want.Profile != "balanced" || want.Model != "original-model" || want.Digest == "" { t.Fatalf("initial snapshot = %+v", want) }
	t.Setenv("AIW_LLM_PROVIDER", "changed-provider")
	t.Setenv("AIW_LLM_MODEL", "changed-model")
	for i := 0; i < 2; i++ {
		recovered, err := advanceWorkflow(meta.ID, meta, workflow.NewStore(""), true, "override-provider", "override-model")
		if err != nil { t.Fatal(err) }
		got := recovered.Automation.PreparedRequest
		if got == nil || got.AISelection == nil || *got.AISelection != want || got.AttemptID != request.AttemptID || len(recovered.Attempts) != 1 { t.Fatalf("retry replaced snapshot or attempt: %+v", got) }
	}
}
