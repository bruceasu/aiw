package main

import (
	"os"
	"testing"

	"aiw/internal/req/domain"
	"aiw/internal/session"
)

func requirementNumberingWorkspace(t *testing.T) string {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(dir); err != nil { t.Fatal(err) }
	t.Setenv("AIW_REQUIREMENT_SESSION", "")
	return dir
}

func TestRequirementNumberingCLI(t *testing.T) {
	requirementNumberingWorkspace(t)
	for _, args := range [][]string{
		{"new", "add-chat-support", "Chat support"},
		{"new", "add-chat-support"},
		{"new", "--id", "legacy_report", "Legacy"},
	} {
		if err := DispatchRequirement(args); err != nil { t.Fatal(err) }
	}
	for id, title := range map[string]string{"REQ00001-add-chat-support": "Chat support", "REQ00002-add-chat-support": "add-chat-support", "legacy_report": "Legacy"} {
		if _, err := os.Stat("docs/requirements/" + id + "/requirement.toml"); err != nil { t.Fatal(err) }
		meta, err := requirement.Read(id)
		if err != nil || meta.ID != id || meta.Title != title { t.Fatalf("%s: %#v %v", id, meta, err) }
	}
	if err := DispatchRequirement([]string{"show", "REQ00001-add-chat-support"}); err != nil { t.Fatal(err) }
	if err := DispatchRequirement([]string{"new", "--id", "legacy_report"}); err == nil { t.Fatal("explicit duplicate accepted") }
}

func TestRequirementNumberingLegacyPendingAction(t *testing.T) {
	dir := requirementNumberingWorkspace(t)
	store := session.NewStore("")
	if _, err := store.Create("legacy-chat", "Legacy", dir, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	// A previously saved action has no auto_number field.
	if err := store.WriteArtifact("legacy-chat", "pending-requirement-action.json", []byte(`{"kind":"new","requirement_id":"legacy-chat-created","title":"Legacy"}`)); err != nil { t.Fatal(err) }
	id, err := confirmRequirementAction(store, "legacy-chat")
	if err != nil || id != "legacy-chat-created" { t.Fatalf("legacy action changed identity: %s %v", id, err) }
	meta, err := requirement.Read(id)
	if err != nil || meta.Conversation.SessionID != "legacy-chat" { t.Fatalf("legacy binding: %#v %v", meta, err) }
	if _, err := os.Stat(".ai/requirements/sequence"); !os.IsNotExist(err) { t.Fatalf("legacy action consumed a number: %v", err) }
}

func TestRequirementNumberingConfirmedIDCanCapture(t *testing.T) {
	dir := requirementNumberingWorkspace(t)
	store := session.NewStore("")
	if _, err := store.Create("numbered-chat", "Numbered", dir, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	t.Setenv("AIW_REQUIREMENT_SESSION", "numbered-chat")
	if err := prepareRequirementAction([]string{"new", "chat-support"}); err != nil { t.Fatal(err) }
	if _, err := displayRequirementCheckpoint(store, requirementChatPlan{SessionID: "numbered-chat"}); err != nil { t.Fatal(err) }
	id, err := confirmRequirementAction(store, "numbered-chat")
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile("draft.md", []byte("Add chat support."), 0o644); err != nil { t.Fatal(err) }
	if err := prepareRequirementAction([]string{"capture", id, "problem-brief", "--file", "draft.md"}); err != nil { t.Fatal(err) }
	if _, err := displayRequirementCheckpoint(store, requirementChatPlan{SessionID: "numbered-chat", RequirementID: id}); err != nil { t.Fatal(err) }
	capturedID, err := confirmRequirementAction(store, "numbered-chat")
	if err != nil || capturedID != id { t.Fatalf("capture lost identity: %s %v", capturedID, err) }
	meta, err := requirement.Read(id)
	if err != nil || meta.Status != "DISCOVERED" || meta.Artifacts["problem-brief"].Digest == "" { t.Fatalf("capture: %#v %v", meta, err) }
}
