package requirement

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestConversationHistoryRecovery(t *testing.T) {
	inTempDir(t)
	if _, err := Create("notify", "Notify"); err != nil { t.Fatal(err) }
	input := "Notify on completion"
	turn, err := RunConversationTurn(context.Background(), "notify", input, nil, ConfirmedFacts{}, func(_ context.Context, phase, _ string) (string, error) {
		if phase == "method-selection" { return "{}", nil }
		snapshot, err := LoadConversationContext("notify", input, nil)
		if err != nil { t.Fatal(err) }
		assessment := CoverageAssessment{Version: 1, RequirementID: "notify", Revision: snapshot.Requirement.Revision}
		for _, dimension := range coverageDimensions {
			assessment.Items = append(assessment.Items, CoverageItem{
				Dimension: dimension, Status: "needs_input", Conclusion: "Completion needs definition",
				Sources: []CoverageReference{{Source: "user-input", Digest: digest([]byte(input)), Quote: input}},
				Question: "What counts as completion?", Impact: "Defines correctness", NextStep: "Ask the owner",
			})
		}
		return encodeCoverage(t, assessment), nil
	})
	if err != nil { t.Fatal(err) }
	record := ConversationEvidence{Version: 1, RequirementID: "notify", State: "completed", Turns: []int{1, 2}, Turn: turn}
	encode := func(record ConversationEvidence) string {
		t.Helper()
		b, err := json.Marshal(record)
		if err != nil { t.Fatal(err) }
		return string(b)
	}
	raw := encode(record)
	candidate, notice := RestoreConversationCandidate(raw, "notify", ConfirmedFacts{})
	if candidate == nil || candidate.Content != turn.Discovery.Coverage.Raw || !strings.Contains(notice, "unconfirmed") { t.Fatalf("recovery failed: %s", notice) }
	// A recovered candidate must still pass through a new model turn.
	calls := 0
	_, err = RunConversationTurn(context.Background(), "notify", "Clarify delivery failures", candidate, ConfirmedFacts{}, func(_ context.Context, _, prompt string) (string, error) {
		calls++
		if calls == 1 && !strings.Contains(prompt, "What counts as completion?") { t.Fatal("recovered question missing from input") }
		if calls == 1 { return "{}", nil }
		return "invalid", nil
	})
	if err == nil || calls != 2 { t.Fatal("history bypassed a fresh assessment") }
	for _, state := range []string{"running", "failed"} {
		record.State = state
		if candidate, _ := RestoreConversationCandidate(encode(record), "notify", ConfirmedFacts{}); candidate != nil { t.Fatal("incomplete history reused") }
	}
	record.State = "completed"
	// A method fingerprint change invalidates reuse even at the same revision.
	methodRecord := record
	methodRecord.Turn.Context.Methods = append([]ConversationSource(nil), record.Turn.Context.Methods...)
	methodRecord.Turn.Context.Methods[0].Digest = "older-method"
	if candidate, notice := RestoreConversationCandidate(encode(methodRecord), "notify", ConfirmedFacts{}); candidate != nil || !strings.Contains(notice, "stale") { t.Fatal("changed source fingerprint reused") }
	record.Turn.Discovery.Coverage.Raw = "invalid"
	if candidate, _ := RestoreConversationCandidate(encode(record), "notify", ConfirmedFacts{}); candidate != nil { t.Fatal("invalid output reused") }
	if candidate, _ := RestoreConversationCandidate(raw, "other", ConfirmedFacts{}); candidate != nil { t.Fatal("cross-Requirement recovery") }
	writeContextDocument(t, "draft.md", "Changed scope")
	if _, _, err := Capture("notify", "problem-brief", "draft.md"); err != nil { t.Fatal(err) }
	if candidate, notice := RestoreConversationCandidate(raw, "notify", ConfirmedFacts{}); candidate != nil || !strings.Contains(notice, "stale") { t.Fatal("stale candidate reused") }
	for _, old := range []string{"", "invalid", `{"version":99}`} {
		if candidate, notice := RestoreConversationCandidate(old, "notify", ConfirmedFacts{}); candidate != nil || notice == "" { t.Fatal("legacy history was silently accepted") }
	}
}
