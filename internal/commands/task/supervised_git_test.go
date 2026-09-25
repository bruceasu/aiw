package task

import (
	"os"
	"strings"
	"testing"

	"aiw/internal/workflow"
)

func TestSupervisedGitCommandPrefixPreservesShellPath(t *testing.T) {
	directory := "/work/task with $literal's name"
	want := `git -c 'safe.directory=/work/task with $literal'"'"'s name' -C '/work/task with $literal'"'"'s name'`
	if os.PathSeparator == '\\' {
		directory = `C:\Users\task with $literal's name\worktree`
		want = `git -c 'safe.directory=C:/Users/task with $literal''s name/worktree' -C 'C:/Users/task with $literal''s name/worktree'`
	}
	if got := supervisedGitCommandPrefix(directory); got != want {
		t.Fatalf("Git command prefix = %q, want %q", got, want)
	}
}



func TestSupervisedGitTrustDirectoryReadsPreflightScope(t *testing.T) {
	got, err := supervisedGitTrustDirectory([]string{
		"PATH=fixture",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=safe.directory",
		"GIT_CONFIG_VALUE_0=C:/worktree",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "C:/worktree" {
		t.Fatalf("trust directory = %q", got)
	}
}

func TestSupervisedWorkItemInstructionBuildsScopedProviderRequest(t *testing.T) {
	const currentDirectory = "C:/current worktree with $literal's name"
	const oldHandoffDirectory = "C:/old handoff worktree"

	tests := []struct {
		name          string
		title         string
		wantCheckbox  bool
	}{
		{
			name:         "ordinary implementation with a Chinese title",
			title:        "为普通实现项补充离线回归测试",
			wantCheckbox: false,
		},
		{
			name:         "scope review keeps its checkbox-only restriction",
			title:        "Review scope: no unrelated changes",
			wantCheckbox: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepareSupervisedInstructionState(t, test.title)
			instruction, err := supervisedWorkItemInstruction("task-1", "wi-0001", []string{
				"GIT_CONFIG_COUNT=1",
				"GIT_CONFIG_KEY_0=safe.directory",
				"GIT_CONFIG_VALUE_0=" + currentDirectory,
			})
			if err != nil {
				t.Fatal(err)
			}

			wantPrefix := supervisedGitCommandPrefix(currentDirectory)
			if !strings.Contains(instruction, wantPrefix) {
				t.Fatalf("provider instruction does not contain current Git prefix %q: %s", wantPrefix, instruction)
			}
			if strings.Contains(instruction, oldHandoffDirectory) {
				t.Fatalf("provider instruction reused an old handoff directory: %s", instruction)
			}
			if got := strings.Contains(instruction, "only the selected checkbox"); got != test.wantCheckbox {
				t.Fatalf("checkbox-only restriction = %t, want %t: %s", got, test.wantCheckbox, instruction)
			}
		})
	}
}

func TestSupervisedWorkItemInstructionRejectsMissingOrInvalidPreflightTrust(t *testing.T) {
	tests := []struct {
		name        string
		environment []string
	}{
		{name: "missing trust evidence", environment: nil},
		{name: "wrong configuration count", environment: []string{
			"GIT_CONFIG_COUNT=2",
			"GIT_CONFIG_KEY_0=safe.directory",
			"GIT_CONFIG_VALUE_0=C:/current-worktree",
		}},
		{name: "unexpected additional configuration", environment: []string{
			"GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=safe.directory",
			"GIT_CONFIG_VALUE_0=C:/current-worktree",
			"GIT_CONFIG_KEY_1=user.name",
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepareSupervisedInstructionState(t, "ordinary implementation")
			instruction, err := supervisedWorkItemInstruction("task-1", "wi-0001", test.environment)
			if err == nil {
				t.Fatalf("instruction = %q, want preflight trust failure", instruction)
			}
			if instruction != "" {
				t.Fatalf("failed preflight returned a dispatchable instruction: %q", instruction)
			}
		})
	}
}

func TestSupervisedGitCommandPrefixPreservesWindowsAndPOSIXSpecialPaths(t *testing.T) {
	tests := []struct {
		name      string
		directory string
	}{
		{name: "Windows", directory: `C:\work tree\$literal's name`},
		{name: "POSIX", directory: `/work tree/$literal's name`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prefix := supervisedGitCommandPrefix(test.directory)
			canonical := strings.ReplaceAll(test.directory, "\\", "/")
			escaped := strings.ReplaceAll(canonical, "'", "'\"'\"'")
			if os.PathSeparator == '\\' {
				escaped = strings.ReplaceAll(canonical, "'", "''")
			}
			if !strings.Contains(prefix, "safe.directory="+escaped) {
				t.Fatalf("safe.directory lost the literal path: %q", prefix)
			}
			if !strings.Contains(prefix, " -C ") || !strings.Contains(prefix, "$literal") {
				t.Fatalf("Git command prefix lost required arguments: %q", prefix)
			}
		})
	}
}

func prepareSupervisedInstructionState(t *testing.T, title string) {
	t.Helper()
	t.Setenv("AIW_ROOT", t.TempDir())
	state := workflow.RuntimeState{
		SchemaVersion: workflow.SchemaVersion,
		Task:          workflow.TaskReference{ID: "task-1", Workspace: ".", Kind: workflow.WorkspacePrimary},
		Planning:      workflow.PlanningReady,
		Delivery:      workflow.DeliveryUnmanaged,
		WorkItems: []workflow.WorkItem{{
			ID:    "wi-0001",
			Title: title,
			State: workflow.WorkItemReady,
		}},
	}
	if _, err := workflow.NewStore("").Create(state); err != nil {
		t.Fatal(err)
	}
}
