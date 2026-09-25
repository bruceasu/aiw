package workflow

import (
	"strings"
	"testing"
)

func grantTestEntry() GrantEntry {
	ref := ActorReference{Kind: "verification-plan", Path: "reports/plan.json", SHA256: strings.Repeat("a", 64)}
	return SealGrantEntry(GrantEntry{ID: "approve-1", Sequence: 1, Date: "2026-09-18T00:00:00Z", Permission: "test-run", Approved: true, Reason: "The fixed plan follows the policy.", AuthorizerType: "ai", AuthorizerID: "provider/model/request-1", ApplicantStage: "test-plan", DecisionReference: ref, PlanReference: GrantVersion{ActorReference: ref, SchemaVersion: 2, Version: "plan-1"}, PolicyReference: GrantVersion{ActorReference: ref, SchemaVersion: 1, Version: "policy-1"}, Scope: GrantScope{WorkspaceDigest: strings.Repeat("b", 64), Paths: []string{"internal/workflow"}, Actions: []string{"test-run"}, Targets: []string{"C:/workspace/internal/workflow"}}, Supersedes: []string{}})
}

func grantTestDocument(t *testing.T, entries ...GrantEntry) []byte {
	t.Helper()
	data, err := canonicalJSON(GrantLog{SchemaVersion: 1, TaskID: "task", Revision: len(entries), Entries: entries})
	if err != nil { t.Fatal(err) }
	return []byte("# Task decisions\n"+grantOpen+"\n```json\n"+string(data)+"\n```\n"+grantClose+"\n")
}

func TestGrantLogRejectsAmbiguousOrMissingDecisionFields(t *testing.T) {
	entry := grantTestEntry()
	valid := string(grantTestDocument(t, entry))
	if _, err := ParseGrantLog([]byte(valid), "task"); err != nil { t.Fatal(err) }
	cases := map[string]string{
		"duplicate-key": strings.Replace(valid, `"approved":true`, `"approved":true,"approved":false`, 1),
		"unknown-field": strings.Replace(valid, `"approved":true`, `"extra":true,"approved":true`, 1),
		"missing-field": strings.Replace(valid, `"approved":true,`, "", 1),
		"null-decision": strings.Replace(valid, `"approved":true`, `"approved":null`, 1),
		"trailing-value": strings.Replace(valid, "\n```\n", " {}\n```\n", 1),
		"duplicate-block": valid+valid,
		"future-schema": strings.Replace(valid, `"schema_version":1,"task_id"`, `"schema_version":9,"task_id"`, 1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) { if _, err := ParseGrantLog([]byte(input), "task"); err == nil { t.Fatal("ambiguous authorization was accepted") } })
	}
	if _, err := ParseGrantLog([]byte(valid), "another-task"); err == nil { t.Fatal("cross-Task grant was accepted") }
}

func TestGrantDenialInvalidatesOldApprovalAndRequiresExplicitReplacement(t *testing.T) {
	approved := grantTestEntry()
	denied := approved
	denied.ID, denied.Sequence, denied.PreviousDigest, denied.Approved = "deny-2", 2, approved.Digest, false
	denied.Reason = "The execution is no longer allowed."
	denied = SealGrantEntry(denied)
	log, err := ParseGrantLog(grantTestDocument(t, approved, denied), "task")
	if err != nil { t.Fatal(err) }
	ref := ActorReference{Kind: "grant", Path: "grant.md#"+approved.ID, SHA256: approved.Digest}
	if err := log.authorize(ref, approved.PlanReference.ActorReference, approved.PolicyReference.ActorReference, approved.Scope); err == nil { t.Fatal("old approval survived a later denial") }
	replacement := approved
	replacement.ID, replacement.Sequence, replacement.PreviousDigest = "approve-3", 3, denied.Digest
	replacement.Supersedes = []string{approved.ID}
	replacement = SealGrantEntry(replacement)
	log, err = ParseGrantLog(grantTestDocument(t, approved, denied, replacement), "task")
	if err != nil { t.Fatal(err) }
	ref.Path, ref.SHA256 = "grant.md#"+replacement.ID, replacement.Digest
	if err := log.authorize(ref, replacement.PlanReference.ActorReference, replacement.PolicyReference.ActorReference, replacement.Scope); err == nil { t.Fatal("replacement ignored the denial") }
	replacement.Supersedes = []string{approved.ID, denied.ID}
	replacement = SealGrantEntry(replacement)
	log, err = ParseGrantLog(grantTestDocument(t, approved, denied, replacement), "task")
	if err != nil { t.Fatal(err) }
	ref.SHA256 = replacement.Digest
	if err := log.authorize(ref, replacement.PlanReference.ActorReference, replacement.PolicyReference.ActorReference, replacement.Scope); err != nil { t.Fatal(err) }
}

func TestGrantScopeChangeCannotReuseApproval(t *testing.T) {
	entry := grantTestEntry()
	log, err := ParseGrantLog(grantTestDocument(t, entry), "task")
	if err != nil { t.Fatal(err) }
	ref := ActorReference{Kind: "grant", Path: "grant.md#"+entry.ID, SHA256: entry.Digest}
	scope := entry.Scope
	scope.Targets = []string{"C:/another-workspace/internal/workflow"}
	if err := log.authorize(ref, entry.PlanReference.ActorReference, entry.PolicyReference.ActorReference, scope); err == nil { t.Fatal("changed real target reused an approval") }
}
