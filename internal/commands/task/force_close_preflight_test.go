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
