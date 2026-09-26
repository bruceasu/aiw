package requirement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeContextDocument(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestConversationContextBudgetKeepsRequiredBeforeOptional(t *testing.T) {
	inTempDir(t)
	if _, err := Create("notify", "Notify"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Capture("notify", "problem-brief", writeSource(t, "required")); err != nil {
		t.Fatal(err)
	}
	writeContextDocument(t, "docs/background.md", strings.Repeat("x", 4096))
	snapshot, err := LoadConversationContextWithOptions("notify", "Continue", nil, ConversationContextOptions{
		MaxBytes: 2048, BackgroundPaths: []string{"docs/background.md"},
	})
	if err != nil || !snapshot.SourcesLoaded {
		t.Fatalf("optional context blocked required inputs: %#v, %v", snapshot, err)
	}
	if snapshot.Sources[0].Content != "required" {
		t.Fatal("required source was lost")
	}
	last := snapshot.Sources[len(snapshot.Sources)-1]
	if last.Status != "omitted" || last.Content != "" || last.Reason == "" || snapshot.UsedBytes > snapshot.BudgetBytes {
		t.Fatalf("optional budget omission not recorded: %#v", snapshot)
	}
	if _, err := snapshot.Prompt(); err != nil {
		t.Fatalf("optional omission prevented rendering: %v", err)
	}
}

func TestConversationContextBudgetBlocksRequiredSources(t *testing.T) {
	inTempDir(t)
	if _, err := Create("notify", "Notify"); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "brief.md")
	writeContextDocument(t, source, strings.Repeat("x", 4096))
	if _, _, err := Capture("notify", "problem-brief", source); err != nil {
		t.Fatal(err)
	}
	snapshot, err := LoadConversationContextWithOptions("notify", "Continue", nil, ConversationContextOptions{MaxBytes: 2048})
	if err == nil || snapshot.SourcesLoaded || snapshot.Sources[0].Status != "omitted" {
		t.Fatalf("required over-budget source accepted: %#v, %v", snapshot, err)
	}
	if _, err := snapshot.Prompt(); err == nil {
		t.Fatal("required omission permitted rendering")
	}
}

func TestConversationContextSelectsExplicitAndModuleDocumentsOnly(t *testing.T) {
	inTempDir(t)
	writeContextDocument(t, "docs/explicit.md", "explicit")
	writeContextDocument(t, "module/README.md", "module readme")
	writeContextDocument(t, "module/CONTEXT.md", "module context")
	writeContextDocument(t, "module/unrelated.md", "must not be scanned")
	snapshot, err := LoadConversationContextWithOptions("", "Start", nil, ConversationContextOptions{
		BackgroundPaths: []string{"module/README.md", "docs/explicit.md", "./docs/explicit.md"},
		ModulePaths: []string{"module"},
	})
	if err != nil || len(snapshot.Sources) != 3 {
		t.Fatalf("wrong source selection: %#v, %v", snapshot, err)
	}
	for i, expected := range []string{"docs/explicit.md", "module/CONTEXT.md", "module/README.md"} {
		if snapshot.Sources[i].Path != expected || snapshot.Sources[i].Status != "loaded" {
			t.Fatalf("unexpected source order: %#v", snapshot.Sources)
		}
	}
}

func TestConversationContextRejectsUnsafeOptionalPaths(t *testing.T) {
	inTempDir(t)
	writeContextDocument(t, "docs/credentials.txt", "must not read")
	writeContextDocument(t, ".env", "must not read")
	paths := []string{"../outside.md", "/outside.md", "C:\\outside.md", "docs/../safe.md", "docs/a.md:stream", "docs/credentials.txt", ".env", "secrets/notes.md"}
	snapshot, err := LoadConversationContextWithOptions("", "Start", nil, ConversationContextOptions{BackgroundPaths: paths})
	if err != nil || !snapshot.SourcesLoaded || len(snapshot.Sources) != len(paths) {
		t.Fatalf("unsafe path diagnostics missing: %#v, %v", snapshot, err)
	}
	for _, source := range snapshot.Sources {
		if source.Status != "rejected" || source.Content != "" || source.Digest != "" {
			t.Fatalf("unsafe source read: %#v", source)
		}
	}
}

func TestConversationContextRejectsSymbolicLinks(t *testing.T) {
	inTempDir(t)
	target := filepath.Join(t.TempDir(), "outside.md")
	writeContextDocument(t, target, "outside content")
	if err := os.Symlink(target, "linked.md"); err != nil {
		t.Skipf("symlink creation unavailable on this host: %v", err)
	}
	snapshot, err := LoadConversationContextWithOptions("", "Start", nil, ConversationContextOptions{BackgroundPaths: []string{"linked.md"}})
	if err != nil || snapshot.Sources[0].Status != "rejected" || snapshot.Sources[0].Content != "" {
		t.Fatalf("symbolic link was read: %#v, %v", snapshot, err)
	}
}

func TestConversationContextBudgetAndMissingBackground(t *testing.T) {
	inTempDir(t)
	snapshot, err := LoadConversationContextWithOptions("", "Start", &ConversationCandidate{Content: strings.Repeat("x", 40)}, ConversationContextOptions{
		MaxBytes: len(discoveryBaseline) + 16, BackgroundPaths: []string{"missing.md"},
	})
	if err != nil || snapshot.Candidate != nil || snapshot.CandidateNotice == "" || snapshot.Sources[0].Status != "unavailable" {
		t.Fatalf("optional diagnostics missing: %#v, %v", snapshot, err)
	}
	if _, err := LoadConversationContextWithOptions("", strings.Repeat("x", 17), nil, ConversationContextOptions{MaxBytes: 16}); err == nil {
		t.Fatal("oversized current input accepted")
	}
	if _, err := LoadConversationContextWithOptions("", "Start", nil, ConversationContextOptions{BackgroundPaths: make([]string, 33)}); err == nil {
		t.Fatal("unbounded reference list accepted")
	}
}
