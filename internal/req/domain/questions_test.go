package requirement

import "testing"

func TestDiscoveryQuestionsPrioritizeConflictAndImpact(t *testing.T) {
	snapshot, assessment, ref := coverageFixture()
	for i := range assessment.Items {
		assessment.Items[i].Question = "Decide " + assessment.Items[i].Dimension
		assessment.Items[i].ImpactKind = "other"
	}
	assessment.Items[0].Status = "conflict"
	other := ref
	other.Quote = "Delivery failures need a decision."
	assessment.Items[0].Sources = append(assessment.Items[0].Sources, other)
	assessment.Items[1].ImpactKind = "irreversible"
	assessment.Items[2].ImpactKind = "correctness"
	result, err := SelectDiscoveryQuestions(encodeCoverage(t, assessment), snapshot, nil)
	if err != nil { t.Fatal(err) }
	if len(result.Questions) != 3 { t.Fatalf("wrong question count: %d", len(result.Questions)) }
	for i, dimension := range []string{"roles", "problem", "current_workflow"} {
		if result.Questions[i].Dimension != dimension { t.Fatalf("wrong order: %#v", result.Questions) }
	}
	if len(result.Questions[0].Sources) != 2 || result.Questions[0].Impact == "" { t.Fatal("conflict evidence or impact lost") }
}

func TestDiscoveryQuestionsReuseSettledAndAllowNone(t *testing.T) {
	snapshot, assessment, ref := coverageFixture()
	for i := range assessment.Items {
		assessment.Items[i].Status, assessment.Items[i].Question = "resolved", ""
	}
	result, err := SelectDiscoveryQuestions(encodeCoverage(t, assessment), snapshot, []CoverageReference{ref})
	if err != nil || len(result.Questions) != 0 || len(result.Settled) != 11 { t.Fatalf("settled facts re-asked: %#v, %v", result, err) }
	assessment.Items[4].Status, assessment.Items[4].Question = "conflict", "Which scope applies?"
	other := ref
	other.Quote = "Delivery failures need a decision."
	assessment.Items[4].Sources = append(assessment.Items[4].Sources, other)
	result, err = SelectDiscoveryQuestions(encodeCoverage(t, assessment), snapshot, []CoverageReference{ref})
	if err != nil || len(result.Questions) != 1 || result.Questions[0].Dimension != "scope" { t.Fatalf("new conflict not surfaced: %#v, %v", result, err) }
}

func TestDiscoveryQuestionsDeduplicateAndRejectInvalid(t *testing.T) {
	snapshot, assessment, _ := coverageFixture()
	result, err := SelectDiscoveryQuestions(encodeCoverage(t, assessment), snapshot, nil)
	if err != nil || len(result.Questions) != 1 { t.Fatalf("duplicate question repeated: %#v, %v", result, err) }
	assessment.Items[0].ImpactKind = "urgent-invented"
	result, err = SelectDiscoveryQuestions(encodeCoverage(t, assessment), snapshot, nil)
	if err == nil || len(result.Questions) != 0 || result.Coverage.Assessment != nil { t.Fatal("invalid coverage produced questions") }
}

func TestDiscoveryQuestionsOptionsKeepEvidenceAndTradeoffs(t *testing.T) {
	snapshot, assessment, ref := coverageFixture()
	assessment.Items[0].ImpactKind = "irreversible"
	assessment.Items[0].Options = []DiscoveryOption{
		{Label: "Retry", Tradeoff: "May send duplicates", Sources: []CoverageReference{ref}},
		{Label: "Report failure", Tradeoff: "Needs human action", Sources: []CoverageReference{ref}},
	}
	result, err := SelectDiscoveryQuestions(encodeCoverage(t, assessment), snapshot, nil)
	if err != nil || len(result.Questions[0].Options) != 2 { t.Fatalf("options lost: %#v, %v", result, err) }
	assessment.Items[0].Options[0].Tradeoff = ""
	if _, err := SelectDiscoveryQuestions(encodeCoverage(t, assessment), snapshot, nil); err == nil { t.Fatal("option without tradeoff accepted") }
	assessment.Items[0].Options[0].Tradeoff = "May send duplicates"
	assessment.Items[0].Options[0].Sources[0].Digest = "stale"
	if _, err := SelectDiscoveryQuestions(encodeCoverage(t, assessment), snapshot, nil); err == nil { t.Fatal("stale option accepted") }
}
