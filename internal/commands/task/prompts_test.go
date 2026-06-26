package task

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLanguageCopilotTemplate_PrefersLanguageLocal(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "dot.github"), 0o755); err != nil {
		t.Fatal(err)
	}

	local := "local-copilot"
	fallback := "fallback-copilot"
	if err := os.WriteFile(filepath.Join(root, "go", filepath.Base(copilotFile)), []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dot.github", "copilot-instructions-for-go.md"), []byte(fallback), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readLanguageCopilotTemplate(root, "go")
	if err != nil {
		t.Fatalf("read language copilot template: %v", err)
	}
	if got != local {
		t.Fatalf("got %q, want %q", got, local)
	}
}

func TestReadLanguageCopilotTemplate_FallsBackToDotGithub(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dot.github"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := "dot-github-copilot"
	if err := os.WriteFile(filepath.Join(root, "dot.github", "copilot-instructions-for-python.md"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readLanguageCopilotTemplate(root, "python")
	if err != nil {
		t.Fatalf("read language copilot template: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReadLanguageCopilotTemplate_FallsBackToHiddenGithub(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := "hidden-github-copilot"
	if err := os.WriteFile(filepath.Join(root, ".github", "copilot-instructions-for-java.md"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readLanguageCopilotTemplate(root, "java")
	if err != nil {
		t.Fatalf("read language copilot template: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestIsTemplateDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "dot.github"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go", agentsFile), []byte("# AGENTS"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !isTemplateDirectory(root, "go") {
		t.Fatal("expected go to be recognized as a template directory")
	}
	if isTemplateDirectory(root, "dot.github") {
		t.Fatal("expected dot.github to be excluded from template directories")
	}
}

func TestPromptConflictAction_Merge(t *testing.T) {
	in := strings.NewReader("m\n")
	var out bytes.Buffer
	action, err := promptConflictAction("AGENTS.md", true, in, &out)
	if err != nil {
		t.Fatalf("prompt conflict action: %v", err)
	}
	if action != conflictActionMerge {
		t.Fatalf("got %q, want %q", action, conflictActionMerge)
	}
}

func TestPromptConflictAction_Overwrite(t *testing.T) {
	in := strings.NewReader("o\n")
	var out bytes.Buffer
	action, err := promptConflictAction("CODEX.md", false, in, &out)
	if err != nil {
		t.Fatalf("prompt conflict action: %v", err)
	}
	if action != conflictActionOverwrite {
		t.Fatalf("got %q, want %q", action, conflictActionOverwrite)
	}
}

func TestPromptConflictAction_RejectsMergeForNonMergeableFile(t *testing.T) {
	in := strings.NewReader("m\no\n")
	var out bytes.Buffer
	action, err := promptConflictAction("prompts/core/validation.md", false, in, &out)
	if err != nil {
		t.Fatalf("prompt conflict action: %v", err)
	}
	if action != conflictActionOverwrite {
		t.Fatalf("got %q, want %q", action, conflictActionOverwrite)
	}
	if !strings.Contains(out.String(), "invalid choice") {
		t.Fatalf("expected invalid choice hint, got %q", out.String())
	}
}
