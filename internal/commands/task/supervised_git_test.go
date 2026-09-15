package task

import (
	"os"
	"strings"
	"testing"
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

func TestScopedGitEnvironmentReplacesInheritedGitConfig(t *testing.T) {
	environment := scopedGitEnvironment([]string{
		"PATH=fixture",
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=user.name",
		"GIT_CONFIG_VALUE_0=Someone",
		"GIT_CONFIG_KEY_1=safe.directory",
		"GIT_CONFIG_VALUE_1=C:/unrelated",
	}, "C:/registered-worktree")

	got := map[string]string{}
	for _, entry := range environment {
		for _, key := range []string{"GIT_CONFIG_COUNT", "GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0", "GIT_CONFIG_KEY_1", "GIT_CONFIG_VALUE_1"} {
			entryKey, _, found := strings.Cut(entry, "=")
			if found && entryKey == key {
				got[key] = entry
			}
		}
	}
	if got["GIT_CONFIG_COUNT"] != "GIT_CONFIG_COUNT=1" ||
		got["GIT_CONFIG_KEY_0"] != "GIT_CONFIG_KEY_0=safe.directory" ||
		got["GIT_CONFIG_VALUE_0"] != "GIT_CONFIG_VALUE_0=C:/registered-worktree" {
		t.Fatalf("scoped Git environment = %#v", got)
	}
	if got["GIT_CONFIG_KEY_1"] != "" || got["GIT_CONFIG_VALUE_1"] != "" {
		t.Fatalf("inherited Git configuration leaked into scoped environment: %#v", got)
	}
}

func TestSupervisedGitTrustDirectoryReadsPreflightScope(t *testing.T) {
	got := supervisedGitTrustDirectory([]string{
		"PATH=fixture",
		"GIT_CONFIG_VALUE_0=C:/worktree",
	})
	if got != "C:/worktree" {
		t.Fatalf("trust directory = %q", got)
	}
}
