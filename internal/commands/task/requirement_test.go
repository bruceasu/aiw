package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/requirement"
	"aiw/internal/session"
	"aiw/internal/taskx"
)

func TestPromoteRequirementRefusesUnapprovedWithoutTask(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := requirement.Create("unapproved", "Unapproved"); err != nil {
		t.Fatal(err)
	}
	if err := promoteRequirement([]string{"unapproved", "--task", "target"}); err == nil {
		t.Fatal("expected unapproved promotion refusal")
	}
	if _, err := os.Stat(taskx.TaskDir("target")); !os.IsNotExist(err) {
		t.Fatalf("task was created: %v", err)
	}
}

func TestParseTerminalArgsAllowsUnorderedOptionalActor(t *testing.T) {
	id, by, reason, err := parseTerminalArgs([]string{"req", "--reason", "done"}, "archive")
	if err != nil || id != "req" || by == "" || reason != "done" {
		t.Fatalf("unexpected defaults: %q %q %q %v", id, by, reason, err)
	}
	id, by, reason, err = parseTerminalArgs([]string{"req", "--reason", "done", "--by", "owner"}, "archive")
	if err != nil || id != "req" || by != "owner" || reason != "done" {
		t.Fatalf("unexpected unordered args: %q %q %q %v", id, by, reason, err)
	}
}

func TestRequirementSubcommandHelpDoesNotCreateRequirement(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			if err := DispatchRequirement([]string{"new", flag}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join("requirements", flag)); !os.IsNotExist(err) {
				t.Fatalf("help flag created a Requirement: %v", err)
			}
		})
	}
}

func TestRequirementHandoffIsIdempotentAndLeavesRecoverableStateOnFailure(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	meta := requirement.Meta{
		ID: "approved", Revision: 3,
		Approval:  requirement.Approval{Status: "APPROVED", By: "owner", At: "2026-09-08T00:00:00Z"},
		Promotion: requirement.Promotion{Status: "TASK_CREATED", TaskID: "target"},
	}
	artifacts := []requirement.Artifact{{Kind: "requirement-plan", Path: "requirements/approved/requirement-plan.md", Digest: "digest"}}
	if err := writeRequirementHandoff(meta, artifacts); err != nil {
		t.Fatal(err)
	}
	handoff := filepath.Join(taskx.RuntimeTaskDir("target"), "artifacts", "requirement-handoff.md")
	content, err := os.ReadFile(handoff)
	if err != nil || !strings.Contains(string(content), "## Approved Scope") || !strings.Contains(string(content), "## Accepted Risks") {
		t.Fatalf("unexpected handoff: %s (%v)", content, err)
	}
	if err := os.WriteFile(handoff, []byte("user-authored"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeRequirementHandoff(meta, artifacts); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(handoff)
	if err != nil || string(content) != "user-authored" {
		t.Fatalf("handoff was overwritten: %s (%v)", content, err)
	}
	if err := os.RemoveAll(taskx.TaskDir("blocked")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(taskx.TaskDir("blocked")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(taskx.TaskDir("blocked"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	meta.Promotion.TaskID = "blocked"
	if err := writeRequirementHandoff(meta, artifacts); err == nil {
		t.Fatal("expected interrupted handoff failure")
	}
	if meta.Promotion.Status != "TASK_CREATED" {
		t.Fatalf("handoff failure changed recoverable state: %s", meta.Promotion.Status)
	}
}

func TestPrepareRequirementChatCreatesUnboundConversation(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	plan, err := prepareRequirementChat(nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Phase != "intake" || !strings.HasPrefix(plan.SessionID, "requirement-chat-") {
		t.Fatalf("unexpected new chat plan: %#v", plan)
	}
	if _, err := os.Stat(filepath.Join(".ai", "sessions", plan.SessionID, "status.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("requirements"); !os.IsNotExist(err) {
		t.Fatalf("new chat created a requirement directory: %v", err)
	}
}

func TestPrepareRequirementChatBindsAndRoutesExistingRequirement(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := requirement.Create("existing", "Existing"); err != nil {
		t.Fatal(err)
	}
	plan, err := prepareRequirementChat([]string{"existing"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Phase != "intake" || plan.SessionID != "requirement-existing" {
		t.Fatalf("unexpected existing chat plan: %#v", plan)
	}
	meta, err := requirement.Read("existing")
	if err != nil || meta.Conversation.SessionID != plan.SessionID {
		t.Fatalf("conversation was not bound: %#v %v", meta, err)
	}
	source := filepath.Join(t.TempDir(), "problem.md")
	if err := os.WriteFile(source, []byte("%% NEEDS_INPUT: owner"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := requirement.Capture("existing", "problem-brief", source); err != nil {
		t.Fatal(err)
	}
	meta, err = requirement.Read("existing")
	if err != nil {
		t.Fatal(err)
	}
	phase, err := requirementConversationPhase(meta)
	if err != nil || phase != "deep-discovery" {
		t.Fatalf("unexpected deep-discovery route: %s %v", phase, err)
	}
}

func TestRequirementConversationInstructionsRequireConfirmation(t *testing.T) {
	instructions := requirementConversationInstructions()
	if !strings.Contains(instructions, "type confirm or 确认") || !strings.Contains(instructions, "artifact type, file path, or CLI parameter") {
		t.Fatalf("missing conversation contract: %s", instructions)
	}
}

func TestRequirementConversationConfirmationUsesPlainLanguage(t *testing.T) {
	for _, line := range []string{"confirm", "确认"} {
		if !isRequirementConversationConfirmation(line) {
			t.Fatalf("confirmation was not accepted: %q", line)
		}
	}
	if isRequirementConversationConfirmation("/confirm") {
		t.Fatal("CLI-like confirmation must not be accepted")
	}
}

func TestPreparedRequirementActionNeedsExplicitConfirmation(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	store := session.NewStore("")
	if _, err := store.Create("conversation", "Conversation", dir, "codex", "", "instructions"); err != nil {
		t.Fatal(err)
	}
	previousSession, hadSession := os.LookupEnv("AIW_REQUIREMENT_SESSION")
	t.Cleanup(func() {
		if hadSession {
			_ = os.Setenv("AIW_REQUIREMENT_SESSION", previousSession)
		} else {
			_ = os.Unsetenv("AIW_REQUIREMENT_SESSION")
		}
	})
	if err := os.Setenv("AIW_REQUIREMENT_SESSION", "conversation"); err != nil {
		t.Fatal(err)
	}
	if err := prepareRequirementAction([]string{"new", "chat-created", "Chat created"}); err != nil {
		t.Fatal(err)
	}
	if _, err := requirement.Read("chat-created"); !os.IsNotExist(err) {
		t.Fatalf("prepared action wrote Requirement before confirmation: %v", err)
	}
	if err := confirmRequirementAction(store, "conversation"); err != nil {
		t.Fatal(err)
	}
	meta, err := requirement.Read("chat-created")
	if err != nil || meta.Conversation.SessionID != "conversation" {
		t.Fatalf("confirmation did not create and bind Requirement: %#v %v", meta, err)
	}
	if err := confirmRequirementAction(store, "conversation"); err == nil {
		t.Fatal("expected second confirmation refusal")
	}
}
