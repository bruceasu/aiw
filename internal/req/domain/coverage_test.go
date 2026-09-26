package requirement

import (
	"encoding/json"
	"strings"
	"testing"
)

func coverageFixture() (ConversationContext, CoverageAssessment, CoverageReference) {
	content := "Notify only on Task completion. Delivery failures need a decision."
	ref := CoverageReference{Source: "docs/requirements/notify/problem-brief.md", Digest: digest([]byte(content)), Quote: "Notify only on Task completion."}
	snapshot := ConversationContext{
		Requirement: &ConversationIdentity{ID: "notify", Revision: 2},
		UserInput: "Continue", SourcesLoaded: true,
		Sources: []ConversationSource{{Kind: "problem-brief", Path: ref.Source, Status: "loaded", Content: content, Digest: ref.Digest}},
	}
	assessment := CoverageAssessment{Version: 1, RequirementID: "notify", Revision: 2}
	for _, dimension := range coverageDimensions {
		assessment.Items = append(assessment.Items, CoverageItem{
			Dimension: dimension, Status: "needs_input", Conclusion: "More evidence is needed",
			Sources: []CoverageReference{ref}, Question: "What should happen on failure?",
			Impact: "Changes acceptance", NextStep: "Ask the owner",
		})
	}
	return snapshot, assessment, ref
}

func encodeCoverage(t *testing.T, assessment CoverageAssessment) string {
	t.Helper()
	data, err := json.Marshal(assessment)
	if err != nil { t.Fatal(err) }
	return string(data)
}

func TestCoverageAcceptsFourStates(t *testing.T) {
	snapshot, assessment, ref := coverageFixture()
	assessment.Items[0].Status, assessment.Items[0].Question = "resolved", ""
	assessment.Items[1].Status = "conflict"
	other := ref
	other.Quote = "Delivery failures need a decision."
	assessment.Items[1].Sources = append(assessment.Items[1].Sources, other)
	assessment.Items[2].Status, assessment.Items[2].Question = "not_applicable", ""
	assessment.Items[2].Reason = "This dimension does not apply to the stated scope"
	raw := encodeCoverage(t, assessment)
	result, err := ParseCoverage(raw, snapshot, []CoverageReference{ref})
	if err != nil || result.Assessment == nil || result.Raw != raw || len(result.Diagnostics) != 0 {
		t.Fatalf("valid candidate rejected: %#v, %v", result, err)
	}
	if snapshot.Requirement.Revision != 2 { t.Fatal("validation mutated Requirement") }
}

func TestCoverageRejectsInvalidCandidates(t *testing.T) {
	for _, name := range []string{"status", "missing-dimension", "duplicate-dimension", "unknown-dimension", "stale-revision", "wrong-id", "version", "missing-source", "stale-source", "false-quote", "omitted-source", "unconfirmed-resolved", "one-sided-conflict", "duplicate-conflict", "missing-reason", "missing-question", "missing-impact", "incomplete-context"} {
		t.Run(name, func(t *testing.T) {
			snapshot, assessment, ref := coverageFixture()
			item := &assessment.Items[0]
			switch name {
			case "status": item.Status = "approved"
			case "missing-dimension": assessment.Items = assessment.Items[1:]
			case "duplicate-dimension": item.Dimension = assessment.Items[1].Dimension
			case "unknown-dimension": item.Dimension = "invented"
			case "stale-revision": assessment.Revision--
			case "wrong-id": assessment.RequirementID = "other"
			case "version": assessment.Version = 0
			case "missing-source": item.Sources[0].Source = "missing.md"
			case "stale-source": item.Sources[0].Digest = "old"
			case "false-quote": item.Sources[0].Quote = "invented fact"
			case "omitted-source": snapshot.Sources[0].Status = "omitted"
			case "unconfirmed-resolved": item.Status, item.Question = "resolved", ""
			case "one-sided-conflict": item.Status = "conflict"
			case "duplicate-conflict": item.Status, item.Sources = "conflict", []CoverageReference{ref, ref}
			case "missing-reason": item.Status, item.Question = "not_applicable", ""
			case "missing-question": item.Question = ""
			case "missing-impact": item.Impact = ""
			case "incomplete-context": snapshot.SourcesLoaded = false
			}
			raw := encodeCoverage(t, assessment)
			result, err := ParseCoverage(raw, snapshot, nil)
			if err == nil || result.Assessment != nil || result.Raw != raw || len(result.Diagnostics) == 0 {
				t.Fatalf("invalid candidate accepted: %#v, %v", result, err)
			}
		})
	}
}

func TestCoverageRejectsMalformedOutput(t *testing.T) {
	snapshot, assessment, _ := coverageFixture()
	valid := encodeCoverage(t, assessment)
	for _, raw := range []string{"", "null", "[]", "{", valid + " {}", strings.Replace(valid, `"version":1`, `"approved":true,"version":1`, 1), strings.Repeat("x", defaultContextBytes+1)} {
		result, err := ParseCoverage(raw, snapshot, nil)
		if err == nil || result.Assessment != nil || result.Raw != raw {
			t.Fatalf("malformed output accepted: %v", err)
		}
	}
}

func TestCoverageUserAnswerCannotConfirmItself(t *testing.T) {
	snapshot, assessment, _ := coverageFixture()
	ref := CoverageReference{Source: "user-input", Digest: digest([]byte(snapshot.UserInput)), Quote: snapshot.UserInput}
	assessment.Items[0].Sources = []CoverageReference{ref}
	if _, err := ParseCoverage(encodeCoverage(t, assessment), snapshot, nil); err != nil { t.Fatal(err) }
	assessment.Items[0].Status, assessment.Items[0].Question = "resolved", ""
	result, err := ParseCoverage(encodeCoverage(t, assessment), snapshot, []CoverageReference{ref})
	if err == nil || result.Assessment != nil { t.Fatal("current answer promoted to confirmed fact") }
}
