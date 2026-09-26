package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aiw/internal/req/domain"
	"aiw/internal/session"
)

func TestRequirementReadinessRechecksApprovalEvidence(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = os.Chdir(previous) })
	store := session.NewStore(filepath.Join(dir, ".ai"))
	if _, err := store.Create("approval", "Approval", dir, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	if _, err := requirement.Create("ready", "Ready"); err != nil { t.Fatal(err) }
	body := "Notify once on Task completion. No Work Item notices. The owner confirms the scope. No remaining design decisions."
	if err := os.WriteFile("plan.md", []byte(body), 0o644); err != nil { t.Fatal(err) }
	refs, _, err := requirement.CaptureFactReferences("ready", "requirement-plan", "plan.md", []string{body})
	if err != nil { t.Fatal(err) }
	meta, _, err := requirement.Capture("ready", "requirement-plan", "plan.md")
	if err != nil { t.Fatal(err) }
	facts := requirement.ConfirmedFacts{RequirementID: "ready", Revision: meta.Revision, References: refs}
	// Controlled model output tests the checkpoint contract, not writing quality.
	turn, err := requirement.RunConversationTurn(context.Background(), "ready", "Continue", nil, facts, func(_ context.Context, phase, _ string) (string, error) {
		if phase == "method-selection" { return "{}", nil }
		assessment := requirement.CoverageAssessment{Version: 1, RequirementID: "ready", Revision: meta.Revision, PlanReview: &requirement.PlanReview{}}
		for _, dimension := range []string{"roles", "problem", "current_workflow", "goals", "scope", "rules", "exceptions", "data", "permissions", "dependencies", "acceptance"} {
			assessment.Items = append(assessment.Items, requirement.CoverageItem{Dimension: dimension, Status: "resolved", Conclusion: "Confirmed", Sources: refs, Impact: "Scope", NextStep: "Review"})
		}
		for _, section := range []string{"facts", "assumptions", "goals", "scope", "non_goals", "rules", "acceptance", "sources", "remaining_decisions"} {
			assessment.PlanReview.Sections = append(assessment.PlanReview.Sections, requirement.PlanSection{Section: section, Explanation: "Fixture evidence", Sources: refs})
		}
		b, err := json.Marshal(assessment)
		return string(b), err
	})
	if err != nil || !turn.Readiness.Ready { t.Fatalf("fixture not ready: %v %#v", err, turn.Readiness) }
	record := requirement.ConversationEvidence{Version: 1, RequirementID: "ready", State: "completed", Turns: []int{1, 2}, Turn: turn, Confirmed: facts}
	if err := writeRequirementEvidence(store, "approval", "record.json", record); err != nil { t.Fatal(err) }
	b, _ := json.Marshal(facts)
	if err := store.WriteArtifact("approval", "confirmed-requirement-facts.json", b); err != nil { t.Fatal(err) }
	if err := store.WriteArtifact("approval", "requirement-discussion-latest.json", []byte(`"record.json"`)); err != nil { t.Fatal(err) }
	action := requirementPendingAction{Kind: "approve", RequirementID: "ready", Decision: "APPROVED", By: "owner", Reason: "reviewed"}
	b, _ = json.Marshal(action)
	if err := store.WriteArtifact("approval", "pending-requirement-action.json", b); err != nil { t.Fatal(err) }
	plan := requirementChatPlan{SessionID: "approval", RequirementID: "ready"}
	if shown, err := displayRequirementCheckpoint(store, plan); err != nil || shown == "" { t.Fatalf("ready checkpoint withheld: %v", err) }
	path := filepath.Join(requirement.Dir("ready"), "requirement-plan.md")
	if err := os.WriteFile(path, []byte("Changed after display"), 0o644); err != nil { t.Fatal(err) }
	if _, err := confirmRequirementAction(store, "approval"); err == nil { t.Fatal("changed Plan approved") }
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil { t.Fatal(err) }
	if _, err := displayRequirementCheckpoint(store, plan); err != nil { t.Fatal(err) }
	if _, err := confirmRequirementAction(store, "approval"); err != nil { t.Fatalf("unchanged displayed approval failed: %v", err) }
}

func TestRequirementReadinessPreservesDirectCLI(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = os.Chdir(previous) })
	store := session.NewStore(filepath.Join(dir, ".ai"))
	if _, err := store.Create("readiness", "Readiness", dir, "codex", "", "instructions"); err != nil { t.Fatal(err) }
	if _, err := requirement.Create("draft", "Draft"); err != nil { t.Fatal(err) }
	if err := os.WriteFile("plan.md", []byte("# Plan\n%% NEEDS_INPUT: scope"), 0o644); err != nil { t.Fatal(err) }
	capture := requirementPendingAction{Kind: "capture", RequirementID: "draft", Artifact: "requirement-plan", Source: "plan.md"}
	save := func(action requirementPendingAction) {
		t.Helper()
		b, err := json.Marshal(action)
		if err != nil { t.Fatal(err) }
		if err := store.WriteArtifact("readiness", "pending-requirement-action.json", b); err != nil { t.Fatal(err) }
	}
	save(capture)
	plan := requirementChatPlan{SessionID: "readiness", RequirementID: "draft"}
	if shown, err := displayRequirementCheckpoint(store, plan); err != nil || shown == "" { t.Fatalf("incomplete draft blocked: %v", err) }
	if _, err := confirmRequirementAction(store, "readiness"); err != nil { t.Fatal(err) }
	approve := requirementPendingAction{Kind: "approve", RequirementID: "draft", Decision: "APPROVED", By: "owner", Reason: "reviewed"}
	save(approve)
	if shown, err := displayRequirementCheckpoint(store, plan); err != nil || shown != "" { t.Fatal("unready approval was displayed") }
	save(approve)
	if _, err := confirmRequirementAction(store, "readiness"); err == nil { t.Fatal("chat approval bypassed readiness") }
	if err := approveRequirement([]string{"draft", "APPROVED", "--by", "owner", "--reason", "explicit direct CLI decision"}); err != nil { t.Fatalf("direct CLI contract changed: %v", err) }
	meta, err := requirement.Read("draft")
	if err != nil || meta.Status != "APPROVED" { t.Fatal("direct CLI approval missing") }
}
