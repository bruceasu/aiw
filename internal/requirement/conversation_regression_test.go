package requirement

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequirementRegressionStopsBeforeModelOnInvalidContext(t *testing.T) {
	for _, kind := range []string{"missing", "changed", "budget"} {
		t.Run(kind, func(t *testing.T) {
			inTempDir(t)
			if _, err := Create("notify", "Notify"); err != nil { t.Fatal(err) }
			writeContextDocument(t, "draft.md", "Only notify on Task completion")
			if _, _, err := Capture("notify", "problem-brief", "draft.md"); err != nil { t.Fatal(err) }
			path := filepath.Join(Dir("notify"), "problem-brief.md")
			input := "Continue"
			switch kind {
			case "missing":
				if err := os.Remove(path); err != nil { t.Fatal(err) }
			case "changed":
				writeContextDocument(t, path, "Unregistered change")
			case "budget":
				input = strings.Repeat("x", defaultContextBytes+1)
			}
			calls := 0
			turn, err := RunConversationTurn(context.Background(), "notify", input, nil, ConfirmedFacts{}, func(context.Context, string, string) (string, error) {
				calls++
				return "{}", nil
			})
			if err == nil || calls != 0 || turn.Readiness.Ready || turn.Discovery.Coverage.Assessment != nil {
				t.Fatalf("invalid context reached model or advanced: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestRequirementRegressionModelFailureRetainsDiagnostics(t *testing.T) {
	for _, failAt := range []string{"method-selection", "coverage-assessment"} {
		t.Run(failAt, func(t *testing.T) {
			inTempDir(t)
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			turn, err := RunConversationTurn(ctx, "", "Notify me on completion", nil, ConfirmedFacts{}, func(ctx context.Context, phase, _ string) (string, error) {
				calls++
				if phase == failAt {
					cancel()
					return "partial response", ctx.Err()
				}
				return "{}", nil
			})
			expected := 1
			if failAt == "coverage-assessment" { expected = 2 }
			if !errors.Is(err, context.Canceled) || calls != expected || turn.Readiness.Ready || turn.Discovery.Coverage.Assessment != nil { t.Fatalf("failed call retried or accepted: %d %v", calls, err) }
			if failAt == "method-selection" && turn.MethodOutput != "partial response" { t.Fatal("method output lost") }
			if failAt == "coverage-assessment" && (turn.Discovery.Coverage.Raw != "partial response" || len(turn.Discovery.Coverage.Diagnostics) == 0) { t.Fatal("coverage diagnostics lost") }
			if _, err := os.Stat(Root); !os.IsNotExist(err) { t.Fatal("failed new conversation wrote a Requirement") }
		})
	}
}

func TestRequirementRegressionRejectsContextChangedDuringResponse(t *testing.T) {
	_, assessment, facts, _ := readinessFixture(t)
	raw := encodeCoverage(t, assessment)
	turn, err := RunConversationTurn(context.Background(), "ready", "Continue", nil, facts, func(_ context.Context, phase, _ string) (string, error) {
		if phase == "method-selection" { return "{}", nil }
		// Simulate an external writer during a tool-capable model response.
		writeContextDocument(t, "new-brief.md", "New requirement scope")
		if _, _, err := Capture("ready", "problem-brief", "new-brief.md"); err != nil { t.Fatal(err) }
		return raw, nil
	})
	if err == nil || turn.Readiness.Ready || turn.Discovery.Coverage.Assessment != nil || turn.Discovery.Coverage.Raw != raw || len(turn.Discovery.Coverage.Diagnostics) == 0 { t.Fatalf("stale in-flight output accepted: %v", err) }
}

func TestRequirementRegressionSettledScopeAndNewConflict(t *testing.T) {
	_, assessment, facts, _ := readinessFixture(t)
	run := func() ConversationTurn {
		t.Helper()
		turn, err := RunConversationTurn(context.Background(), "ready", "Review scope", nil, facts, func(_ context.Context, phase, prompt string) (string, error) {
			if !strings.Contains(prompt, facts.References[0].Quote) { t.Fatal("captured body absent without provider history") }
			if phase == "method-selection" { return "{}", nil }
			return encodeCoverage(t, assessment), nil
		})
		if err != nil { t.Fatal(err) }
		return turn
	}
	if turn := run(); len(turn.Discovery.Questions) != 0 || turn.Phase != "synthesis" { t.Fatal("settled scope was re-asked") }
	for i := range assessment.Items {
		if assessment.Items[i].Dimension != "scope" { continue }
		assessment.Items[i].Status = "conflict"
		assessment.Items[i].Question = "Which scope applies?"
		assessment.Items[i].Sources = []CoverageReference{facts.References[0], facts.References[3]}
	}
	turn := run()
	if turn.Phase != "deep-discovery" || len(turn.Discovery.Questions) != 1 || turn.Discovery.Questions[0].Dimension != "scope" || len(turn.Discovery.Questions[0].Sources) != 2 || turn.Readiness.Ready { t.Fatal("new conflict did not reopen scope with both sources") }
	// These controlled excerpts test conflict handling, not semantic contradiction.
}

func TestRequirementRegressionOldApprovalIsNotRevoked(t *testing.T) {
	_, assessment, facts, _ := readinessFixture(t)
	approved, err := Approve("ready", "APPROVED", "owner", "Reviewed earlier")
	if err != nil { t.Fatal(err) }
	assessment.Revision = approved.Revision
	for i := range assessment.Items {
		assessment.Items[i].Status = "needs_input"
		assessment.Items[i].Question = "Review a new risk"
	}
	turn, err := RunConversationTurn(context.Background(), "ready", "Review new evidence", nil, facts, func(_ context.Context, phase, _ string) (string, error) {
		if phase == "method-selection" { return "{}", nil }
		return encodeCoverage(t, assessment), nil
	})
	if err != nil || turn.Readiness.Ready || len(turn.Readiness.Confirmed) != 0 { t.Fatalf("stale confirmations or readiness reused: %v", err) }
	current, err := Read("ready")
	if err != nil || current.Status != "APPROVED" || current.Revision != approved.Revision || current.Approval != approved.Approval { t.Fatal("discussion altered an old human approval") }
}

func TestRequirementRegressionRecoveryRejectsChangedMethodFile(t *testing.T) {
	_, assessment, facts, _ := readinessFixture(t)
	input := "Define a financial metric"
	methodPath := ".agents/skills/finance-metric-brief/SKILL.md"
	writeContextDocument(t, methodPath, "Review the settlement cut-off")
	turn, err := RunConversationTurn(context.Background(), "ready", input, nil, facts, func(_ context.Context, phase, prompt string) (string, error) {
		if phase == "method-selection" {
			b, err := json.Marshal(methodSuggestion(input))
			return string(b), err
		}
		if !strings.Contains(prompt, "Review the settlement cut-off") { t.Fatal("actual method missing") }
		return encodeCoverage(t, assessment), nil
	})
	if err != nil { t.Fatal(err) }
	record := ConversationEvidence{Version: 1, RequirementID: "ready", State: "completed", Turns: []int{1, 2}, Turn: turn}
	b, err := json.Marshal(record)
	if err != nil { t.Fatal(err) }
	if candidate, notice := RestoreConversationCandidate(string(b), "ready", facts); candidate == nil { t.Fatal(notice) }
	writeContextDocument(t, methodPath, "Changed method guidance")
	if candidate, notice := RestoreConversationCandidate(string(b), "ready", facts); candidate != nil || !strings.Contains(notice, "stale") { t.Fatal("changed on-disk method did not invalidate history") }
	// Saved bytes remain inspectable and do not become the new method body.
	var saved ConversationEvidence
	if err := json.Unmarshal(b, &saved); err != nil { t.Fatal(err) }
	found := false
	for _, source := range saved.Turn.Context.Methods {
		if source.Path == methodPath { found = source.Content == "Review the settlement cut-off" }
	}
	if !found { t.Fatal("historical input lost") }
}
