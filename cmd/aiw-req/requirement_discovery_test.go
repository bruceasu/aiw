package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aiw/internal/req/domain"
	"aiw/internal/session"
)

func TestRequirementCaptureFactCheckpoint(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = os.Chdir(previous) })
	store := session.NewStore(filepath.Join(dir, ".ai"))
	if _, err := store.Create("conversation", "Conversation", dir, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	if _, err := requirement.Create("notify", "Notify"); err != nil { t.Fatal(err) }
	if err := os.WriteFile("draft.md", []byte("Only Task completion"), 0o644); err != nil { t.Fatal(err) }
	action := requirementPendingAction{Kind: "capture", RequirementID: "notify", Artifact: "problem-brief", Source: "draft.md", Facts: []string{"Only Task completion"}}
	b, _ := json.Marshal(action)
	if err := store.WriteArtifact("conversation", "pending-requirement-action.json", b); err != nil { t.Fatal(err) }
	plan := requirementChatPlan{SessionID: "conversation", RequirementID: "notify"}
	if _, err := displayRequirementCheckpoint(store, plan); err != nil { t.Fatal(err) }
	if _, err := store.ReadArtifact("conversation", "confirmed-requirement-facts.json"); !os.IsNotExist(err) { t.Fatal("facts confirmed before user checkpoint") }
	if err := os.WriteFile("draft.md", []byte("Changed scope"), 0o644); err != nil { t.Fatal(err) }
	if _, err := confirmRequirementAction(store, "conversation"); err == nil { t.Fatal("changed draft was confirmed") }
	if err := os.WriteFile("draft.md", []byte("Only Task completion"), 0o644); err != nil { t.Fatal(err) }
	if _, err := displayRequirementCheckpoint(store, plan); err != nil { t.Fatal(err) }
	if _, err := confirmRequirementAction(store, "conversation"); err != nil { t.Fatal(err) }
	content, err := store.ReadArtifact("conversation", "confirmed-requirement-facts.json")
	if err != nil { t.Fatal(err) }
	var facts requirement.ConfirmedFacts
	if err := json.Unmarshal([]byte(content), &facts); err != nil { t.Fatal(err) }
	snapshot, err := requirement.LoadConversationContext("notify", "Continue", nil)
	if err != nil || len(facts.Current(snapshot)) != 1 { t.Fatalf("confirmed fragments unavailable: %v", err) }
}

func TestRequirementConversationHistoryLatest(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(filepath.Join(dir, ".ai"))
	if _, err := store.Create("history", "History", dir, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	if raw, err := readRequirementHistory(store, "history"); err != nil || raw != "" { t.Fatal("legacy session failed") }
	record := requirement.ConversationEvidence{Version: 1, State: "running"}
	if err := writeRequirementEvidence(store, "history", "round.json", record); err != nil { t.Fatal(err) }
	if err := store.WriteArtifact("history", "requirement-discussion-latest.json", []byte(`"round.json"`)); err != nil { t.Fatal(err) }
	raw, err := readRequirementHistory(store, "history")
	if err != nil { t.Fatal(err) }
	if candidate, _ := requirement.RestoreConversationCandidate(raw, "", requirement.ConfirmedFacts{}); candidate != nil { t.Fatal("running record restored") }
	if err := store.WriteArtifact("history", "requirement-discussion-latest.json", []byte(`"missing.json"`)); err != nil { t.Fatal(err) }
	if raw, err := readRequirementHistory(store, "history"); err != nil || raw != "invalid" { t.Fatal("missing latest record masked") }
}
