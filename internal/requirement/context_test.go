package requirement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConversationContextNewDoesNotCreateRequirement(t *testing.T) {
	inTempDir(t)
	candidate := &ConversationCandidate{Content: "Maybe send a card"}
	snapshot, err := LoadConversationContext("", "Notify me when a task is done", candidate)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Requirement != nil || len(snapshot.Sources) != 0 || !snapshot.SourcesLoaded {
		t.Fatalf("unexpected new context: %#v", snapshot)
	}
	candidate.Content = "changed by caller"
	if snapshot.Candidate.Content != "Maybe send a card" {
		t.Fatal("candidate was not copied")
	}
	prompt, err := snapshot.Prompt()
	if err != nil || !strings.Contains(prompt, "unconfirmed_candidate") || !strings.Contains(prompt, "Notify me") {
		t.Fatalf("missing new conversation input: %s, %v", prompt, err)
	}
	if _, err := os.Stat(Root); !os.IsNotExist(err) {
		t.Fatalf("context created a requirement: %v", err)
	}
}

func TestConversationContextLoadsCapturedTextWithoutHistory(t *testing.T) {
	inTempDir(t)
	if _, err := Create("notify", "Task notifications"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"problem-brief", "business-case"} {
		source := filepath.Join(t.TempDir(), "source.md")
		if err := os.WriteFile(source, []byte("# Facts\nTask completion only\n# Assumptions\nCard format\n%% NEEDS_INPUT: delivery failures"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := Capture("notify", kind, source); err != nil {
			t.Fatal(err)
		}
	}
	meta, err := Read("notify")
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(Dir("notify"), "requirement.toml"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := LoadConversationContext("notify", "Continue", &ConversationCandidate{
		RequirementID: "notify", Revision: meta.Revision, Content: "Proposed retry",
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Requirement.Revision != meta.Revision || snapshot.Requirement.ID != "notify" {
		t.Fatalf("wrong identity: %#v", snapshot.Requirement)
	}
	if len(snapshot.Sources) != 3 || snapshot.Sources[0].Kind != "business-case" || snapshot.Sources[2].Status != "absent" {
		t.Fatalf("non-deterministic or missing sources: %#v", snapshot.Sources)
	}
	for _, source := range snapshot.Sources[:2] {
		if source.Status != "loaded" || source.Digest == "" || source.Digest != source.RecordedDigest {
			t.Fatalf("unverified captured source: %#v", source)
		}
	}
	prompt, err := snapshot.Prompt()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Task completion only", "Card format", "NEEDS_INPUT: delivery failures", "Proposed retry", "recorded_digest"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q", expected)
		}
	}
	after, err := os.ReadFile(filepath.Join(Dir("notify"), "requirement.toml"))
	if err != nil || string(before) != string(after) {
		t.Fatalf("context mutated metadata: %v", err)
	}
}

func TestConversationContextRejectsMissingOrChangedCapture(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed", true: "missing"}[missing], func(t *testing.T) {
			inTempDir(t)
			if _, err := Create("notify", "Notify"); err != nil {
				t.Fatal(err)
			}
			if _, _, err := Capture("notify", "problem-brief", writeSource(t, "original")); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(Dir("notify"), "problem-brief.md")
			var err error
			if missing {
				err = os.Remove(path)
			} else {
				err = os.WriteFile(path, []byte("unconfirmed edit"), 0o644)
			}
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := LoadConversationContext("notify", "Continue", nil)
			if err == nil || snapshot.SourcesLoaded || snapshot.Sources[0].Content != "" || snapshot.Sources[0].Reason == "" {
				t.Fatalf("unsafe partial context: %#v, %v", snapshot, err)
			}
			if prompt, err := snapshot.Prompt(); err == nil || prompt != "" {
				t.Fatal("failed source was rendered as usable context")
			}
		})
	}
}

func TestConversationContextOmitsStaleCandidates(t *testing.T) {
	inTempDir(t)
	for _, candidate := range []*ConversationCandidate{
		{RequirementID: "another", Content: "wrong requirement"},
		{Revision: 1, Content: "wrong revision"},
	} {
		snapshot, err := LoadConversationContext("", "Start", candidate)
		if err != nil || snapshot.Candidate != nil || snapshot.CandidateNotice == "" {
			t.Fatalf("stale candidate retained: %#v, %v", snapshot, err)
		}
	}
}

func TestConversationContextLoadsArchivedSourcesAndDecisions(t *testing.T) {
	inTempDir(t)
	if _, err := Create("notify", "Notify"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Capture("notify", "requirement-plan", writeSource(t, "plan")); err != nil {
		t.Fatal(err)
	}
	if _, err := Archive("notify", "owner", "kept for reference"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := LoadConversationContext("notify", "Explain this decision", nil)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Requirement.Status != "ARCHIVED" || !strings.Contains(snapshot.Sources[0].Path, "/archive/") {
		t.Fatalf("incorrect archived source: %#v", snapshot)
	}
	if !strings.Contains(snapshot.Sources[1].Content, "kept for reference") {
		t.Fatal("decision context was omitted")
	}
}

func TestConversationContextRejectsInvalidInput(t *testing.T) {
	inTempDir(t)
	for _, id := range []string{".", "..", "../outside"} {
		if _, err := LoadConversationContext(id, "Start", nil); err == nil {
			t.Fatalf("accepted invalid id %q", id)
		}
	}
	if _, err := LoadConversationContext("", " ", nil); err == nil {
		t.Fatal("accepted blank user input")
	}
}
