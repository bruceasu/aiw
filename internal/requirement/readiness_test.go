package requirement

import (
	"context"
	"strings"
	"testing"
)

func readinessFixture(t *testing.T) (ConversationContext, CoverageAssessment, ConfirmedFacts, DeferredDesign) {
	t.Helper()
	inTempDir(t)
	if _, err := Create("ready", "Ready"); err != nil { t.Fatal(err) }
	quotes := []string{
		"Facts: only Task completion triggers a notice.",
		"Assumptions: the local notification service is available.",
		"Goals: the owner learns about completed work.",
		"Scope: notify for this project only.",
		"Non-goals: no Work Item notifications.",
		"Rules: failed delivery is shown to the owner.",
		"Acceptance: when a Task completes, one notice contains its ID.",
		"Sources: the project owner supplied the scope.",
		"Remaining decisions: defer renderer choice because it does not change the message contract.",
	}
	writeContextDocument(t, "plan.md", strings.Join(quotes, "\n"))
	refs, _, err := CaptureFactReferences("ready", "requirement-plan", "plan.md", quotes)
	if err != nil { t.Fatal(err) }
	meta, _, err := Capture("ready", "requirement-plan", "plan.md")
	if err != nil { t.Fatal(err) }
	snapshot, err := LoadConversationContext("ready", "Continue", nil)
	if err != nil { t.Fatal(err) }
	facts := ConfirmedFacts{RequirementID: meta.ID, Revision: meta.Revision, References: refs}
	assessment := CoverageAssessment{Version: 1, RequirementID: meta.ID, Revision: meta.Revision, PlanReview: &PlanReview{}}
	for _, dimension := range coverageDimensions {
		assessment.Items = append(assessment.Items, CoverageItem{Dimension: dimension, Status: "resolved", Conclusion: "Confirmed scope", Sources: []CoverageReference{refs[0]}, Impact: "Defines delivery", NextStep: "Review Plan"})
	}
	for i, section := range planSections {
		assessment.PlanReview.Sections = append(assessment.PlanReview.Sections, PlanSection{Section: section, Explanation: "Plan states this explicitly", Sources: []CoverageReference{refs[i]}})
	}
	deferred := DeferredDesign{Question: "renderer choice", Reason: "it does not change the message contract", Decision: refs[8]}
	assessment.PlanReview.DeferredDesign = []DeferredDesign{deferred}
	return snapshot, assessment, facts, deferred
}

func TestReadinessConversationRequiresHumanDeferral(t *testing.T) {
	_, assessment, facts, _ := readinessFixture(t)
	run := func(facts ConfirmedFacts) ConversationTurn {
		t.Helper()
		turn, err := RunConversationTurn(context.Background(), "ready", "Continue", nil, facts, func(_ context.Context, phase, prompt string) (string, error) {
			if phase == "method-selection" { return "{}", nil }
			if !strings.Contains(prompt, "deferred_design") { t.Fatal("Plan review instructions missing") }
			return encodeCoverage(t, assessment), nil
		})
		if err != nil { t.Fatal(err) }
		return turn
	}
	unconfirmed := facts
	unconfirmed.References = facts.References[:8]
	turn := run(unconfirmed)
	if turn.Readiness.Ready || len(turn.Readiness.UnconfirmedDeferrals) != 1 { t.Fatal("model postponed design without human confirmation") }
	turn = run(facts)
	if !turn.Readiness.Ready || len(turn.Readiness.Deferred) != 1 { t.Fatalf("confirmed postponement did not become ready: %#v", turn.Readiness) }
	meta, err := Read("ready")
	if err != nil || meta.Status != "DECIDED" { t.Fatal("readiness changed lifecycle") }
	assessment.Items[0].Status = "needs_input"
	assessment.Items[0].Question = "Who is the owner?"
	if turn := run(facts); turn.Readiness.Ready || len(turn.Readiness.BusinessBlockers) == 0 { t.Fatal("design postponement waived business blocker") }
}

func TestReadinessRejectsIncompletePlan(t *testing.T) {
	snapshot, assessment, facts, _ := readinessFixture(t)
	for _, name := range []string{"missing-review", "missing-section", "heading-only", "wrong-source", "stale-digest", "unconfirmed-plan", "invalid-output", "no-plan"} {
		t.Run(name, func(t *testing.T) {
			// Decode a fresh value to avoid mutating the shared fixture.
			var changed CoverageAssessment
			if err := decodeConversationJSON(encodeCoverage(t, assessment), &changed); err != nil { t.Fatal(err) }
			current, confirmed := snapshot, facts
			raw := ""
			switch name {
			case "missing-review": changed.PlanReview = nil
			case "missing-section": changed.PlanReview.Sections = changed.PlanReview.Sections[1:]
			case "heading-only": changed.PlanReview.Sections[0].Sources[0].Quote = "# Facts"
			case "wrong-source": changed.PlanReview.Sections[0].Sources[0].Source = "user-input"
			case "stale-digest": changed.PlanReview.Sections[0].Sources[0].Digest = "old"
			case "unconfirmed-plan": confirmed.References = facts.References[:1]
			case "invalid-output": raw = "invalid"
			case "no-plan": current.Sources = nil
			}
			if raw == "" { raw = encodeCoverage(t, changed) }
			if report := AssessReadiness(raw, current, confirmed); report.Ready { t.Fatalf("%s allowed approval advice", name) }
		})
	}
	if substantivePlanQuote("# Facts\n## Scope") || substantivePlanQuote("%% NEEDS_INPUT: scope") { t.Fatal("placeholder accepted") }
}
