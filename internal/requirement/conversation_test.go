package requirement

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestConversationTurnLoadsMethodAndRejectsBadAssessment(t *testing.T) {
	inTempDir(t)
	input := "Define the financial metric"
	writeContextDocument(t, ".agents/skills/finance-metric-brief/SKILL.md", "Discuss settlement cut-off")
	calls := 0
	turn, err := RunConversationTurn(context.Background(), "", input, nil, ConfirmedFacts{}, func(_ context.Context, phase, prompt string) (string, error) {
		calls++
		if calls == 1 {
			b, _ := json.Marshal(methodSuggestion(input))
			return string(b), nil
		}
		if phase != "coverage-assessment" || !strings.Contains(prompt, "Discuss settlement cut-off") { t.Fatal("selected method was not delivered") }
		return "invalid coverage", nil
	})
	if err == nil || calls != 2 || turn.Discovery.Coverage.Assessment != nil || turn.Discovery.Coverage.Raw != "invalid coverage" { t.Fatalf("invalid turn accepted: %#v, %v", turn, err) }
}

func TestConversationTurnRefreshesAfterCapture(t *testing.T) {
	inTempDir(t)
	if _, err := Create("notify", "Notify"); err != nil { t.Fatal(err) }
	input := "Continue"
	run := func(facts ConfirmedFacts) ConversationTurn {
		t.Helper()
		turn, err := RunConversationTurn(context.Background(), "notify", input, nil, facts, func(_ context.Context, phase, prompt string) (string, error) {
			if phase == "method-selection" { return "{}", nil }
			snapshot, err := LoadConversationContext("notify", input, nil)
			if err != nil { t.Fatal(err) }
			assessment := CoverageAssessment{Version: 1, RequirementID: "notify", Revision: snapshot.Requirement.Revision}
			ref := CoverageReference{Source: "user-input", Digest: digest([]byte(input)), Quote: input}
			status, question := "needs_input", "What counts as completion?"
			if refs := facts.Current(snapshot); len(refs) != 0 {
				ref, status, question = refs[0], "resolved", ""
				if !strings.Contains(prompt, ref.Quote) { t.Fatal("confirmed content missing") }
			}
			for _, dimension := range coverageDimensions {
				assessment.Items = append(assessment.Items, CoverageItem{Dimension: dimension, Status: status, Question: question, Conclusion: "Completion scope", Impact: "Correctness", NextStep: "Review", Sources: []CoverageReference{ref}})
			}
			return encodeCoverage(t, assessment), nil
		})
		if err != nil { t.Fatal(err) }
		return turn
	}
	if turn := run(ConfirmedFacts{}); turn.Phase != "discovery" { t.Fatal("new request prematurely synthesized") }
	writeContextDocument(t, "draft.md", "Only notify on Task completion")
	refs, _, err := CaptureFactReferences("notify", "problem-brief", "draft.md", []string{"Only notify on Task completion"})
	if err != nil { t.Fatal(err) }
	meta, _, err := Capture("notify", "problem-brief", "draft.md")
	if err != nil { t.Fatal(err) }
	if turn := run(ConfirmedFacts{}); turn.Phase != "discovery" { t.Fatal("capture alone advanced phase") }
	facts := ConfirmedFacts{RequirementID: meta.ID, Revision: meta.Revision, References: refs}
	if turn := run(facts); turn.Phase != "synthesis" || turn.Context.Requirement.Revision != meta.Revision { t.Fatal("confirmed capture did not refresh phase") }
	facts.Revision--
	if turn := run(facts); turn.Phase != "discovery" { t.Fatal("stale confirmation reused") }
}

func TestConversationTurnMissingMethodStopsBeforeAssessment(t *testing.T) {
	inTempDir(t)
	input := "Discuss a financial metric"
	calls := 0
	_, err := RunConversationTurn(context.Background(), "", input, nil, ConfirmedFacts{}, func(_ context.Context, _, _ string) (string, error) {
		calls++
		b, _ := json.Marshal(methodSuggestion(input))
		return string(b), nil
	})
	if err == nil || calls != 1 { t.Fatal("missing method did not stop assessment") }
}
