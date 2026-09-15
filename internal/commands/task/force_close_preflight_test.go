package task

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func TestForceCloseDeliveryPreflightGitFixtures(t *testing.T) {
	primary, taskWorktree := forceCloseGitFixture(t)
	meta := taskx.TaskMeta{Branch: "feature/task-1", ParentBranch: "main", Worktree: taskWorktree, WorkspaceKind: "isolated"}

	withWorkingDirectory(t, primary)
	if err := preflightForceCloseDelivery(meta, workflow.DeliveryMerged); err != nil {
		t.Fatalf("merged delivery preflight failed: %v", err)
	}
	if err := preflightForceCloseDelivery(meta, workflow.DeliveryDiscarded); err != nil {
		t.Fatalf("discarded delivery preflight failed: %v", err)
	}
}

func TestMergedForceClosePreflightFailurePreservesGitFixture(t *testing.T) {
	primary, taskWorktree := forceCloseGitFixture(t)
	meta := taskx.TaskMeta{Branch: "feature/task-1", ParentBranch: "main", Worktree: taskWorktree, WorkspaceKind: "isolated"}

	withWorkingDirectory(t, primary)
	parentHead := gitFixtureOutput(t, primary, "rev-parse", "HEAD")
	taskHead := gitFixtureOutput(t, taskWorktree, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(primary, "uncommitted.txt"), []byte("preserve me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := preflightForceCloseDelivery(meta, workflow.DeliveryMerged)
	if err == nil || !strings.Contains(err.Error(), "parent worktree must be clean") {
		t.Fatalf("preflight error = %v, want clean-parent rejection", err)
	}
	if got := gitFixtureOutput(t, primary, "rev-parse", "HEAD"); got != parentHead {
		t.Fatalf("parent branch changed from %s to %s", parentHead, got)
	}
	if got := gitFixtureOutput(t, taskWorktree, "rev-parse", "HEAD"); got != taskHead {
		t.Fatalf("Task branch changed from %s to %s", taskHead, got)
	}
	if _, err := os.Stat(taskWorktree); err != nil {
		t.Fatalf("Task worktree was not preserved: %v", err)
	}
}

func TestSupervisorDeliveryCleansVerifiedTaskResources(t *testing.T) {
	primary, taskWorktree := forceCloseGitFixture(t)
	withWorkingDirectory(t, primary)

	const id = "task-1"
	meta := taskx.TaskMeta{ID: id, Branch: "feature/task-1", ParentBranch: "main", Worktree: taskWorktree, WorkspaceKind: "isolated"}
	if err := os.MkdirAll(taskx.RuntimeTaskDir(id), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath(id), meta); err != nil {
		t.Fatal(err)
	}
	store := workflow.NewStore("")
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: id, Workspace: taskWorktree, Kind: workflow.WorkspaceIsolated}, workflow.PlanningReady, workflow.DeliveryPending)
	state.WorkItems = []workflow.WorkItem{{ID: "wi-0001", Checklist: workflow.ChecklistReference{Item: "1.1"}, Title: "accepted", State: workflow.WorkItemCompleted}}
	state.Evidence = []workflow.Evidence{{ID: "evidence-1", WorkItemID: "wi-0001", Kind: workflow.EvidenceStaticReview, State: workflow.EvidencePassed}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}

	taskHead := gitFixtureOutput(t, taskWorktree, "rev-parse", "HEAD")
	if delivered, err := deliverCompletedSupervisorTask(id, store); err != nil || !delivered {
		t.Fatalf("supervised delivery = %t, error = %v", delivered, err)
	}
	if !isAncestorAt(primary, taskHead, meta.ParentBranch) {
		t.Fatal("Task commit is not an ancestor after local delivery")
	}
	if delivered, err := deliverCompletedSupervisorTask(id, store); err != nil || delivered {
		t.Fatalf("repeated delivery = %t, error = %v", delivered, err)
	}
	if err := runWorkflowSupervisor([]string{"supervise", id, "start"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(taskWorktree); !os.IsNotExist(err) {
		t.Fatalf("Task worktree was not cleaned: %v", err)
	}
	if err := gitRunAt(primary, "show-ref", "--verify", "--quiet", "refs/heads/"+meta.Branch); err == nil {
		t.Fatal("Task branch was not cleaned")
	}
	loaded, err := store.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Delivery != workflow.DeliveryMerged {
		t.Fatalf("delivery = %q, want merged", loaded.Delivery)
	}
}

func TestPreserveConflictCandidateRestoresParentAndKeepsTaskWorktree(t *testing.T) {
	primary, taskWorktree := forceCloseGitFixture(t)
	meta := taskx.TaskMeta{ID: "task-1", Branch: "feature/task-1", ParentBranch: "main", Worktree: taskWorktree, WorkspaceKind: "isolated"}
	if err := os.WriteFile(filepath.Join(primary, "README.md"), []byte("parent change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFixtureRun(t, primary, "add", "README.md")
	gitFixtureRun(t, primary, "commit", "-m", "parent change")
	parentHead := gitFixtureOutput(t, primary, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(taskWorktree, "README.md"), []byte("task change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFixtureRun(t, taskWorktree, "add", "README.md")
	gitFixtureRun(t, taskWorktree, "commit", "-m", "task conflict")

	mergeErr := gitRunAt(primary, "merge", "--no-ff", meta.Branch)
	if mergeErr == nil {
		t.Fatal("fixture did not create a parent merge conflict")
	}
	store := workflow.NewStore(t.TempDir())
	state := workflow.NewCompatibleRuntime(workflow.TaskReference{ID: workflow.TaskID(meta.ID), Workspace: taskWorktree, Kind: workflow.WorkspaceIsolated}, workflow.PlanningReady, workflow.DeliveryPending)
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	if err := preserveConflictCandidate(store, meta.ID, primary, taskWorktree, meta, mergeErr); err == nil || !strings.Contains(err.Error(), "preserved candidate branch") {
		t.Fatalf("preserve conflict result = %v", err)
	}
	if got := gitFixtureOutput(t, primary, "rev-parse", "HEAD"); got != parentHead {
		t.Fatalf("parent changed from %s to %s", parentHead, got)
	}
	if _, err := os.Stat(taskWorktree); err != nil {
		t.Fatalf("conflict candidate worktree was removed: %v", err)
	}
	status := gitFixtureOutput(t, taskWorktree, "status", "--porcelain")
	if !strings.Contains(status, "UU README.md") {
		t.Fatalf("candidate did not retain merge conflict: %q", status)
	}
}

func forceCloseGitFixture(t *testing.T) (string, string) {
	t.Helper()
	primary := t.TempDir()
	gitFixtureRun(t, primary, "init", "--initial-branch=main")
	gitFixtureRun(t, primary, "config", "user.email", "fixture@example.test")
	gitFixtureRun(t, primary, "config", "user.name", "Fixture")
	if err := os.WriteFile(filepath.Join(primary, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFixtureRun(t, primary, "add", "README.md")
	gitFixtureRun(t, primary, "commit", "-m", "base")

	taskWorktree := filepath.Join(t.TempDir(), "task-1")
	gitFixtureRun(t, primary, "worktree", "add", "-b", "feature/task-1", taskWorktree, "main")
	if err := os.WriteFile(filepath.Join(taskWorktree, "delivery.txt"), []byte("committed task content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFixtureRun(t, taskWorktree, "add", "delivery.txt")
	gitFixtureRun(t, taskWorktree, "commit", "-m", "task content")
	return primary, taskWorktree
}

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
}

func gitFixtureRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func gitFixtureOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output))
}
