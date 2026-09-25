package openspecgen

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"aiw/internal/ai"
	"aiw/internal/requirement"
)

func qualityFixture() (requirement.GenerationRequest, requirement.GenerationCandidate) {
	request, candidate := businessCandidate()
	candidate.Artifacts[2].Content = strings.ReplaceAll(candidate.Artifacts[2].Content, "Enforce access.", "Enforce export permission.")
	return request, candidate
}

func TestCandidateQualityReportsVerifiedEvidence(t *testing.T) {
	request, candidate := qualityFixture()
	report, err := candidateQuality(request, candidate)
	if err != nil || !report.StructureValid || !report.CoverageValid || len(report.VerifiedCoverage) != 4 { t.Fatalf("%+v: %v", report, err) }
	if report.DesignReadiness != "not-assessed" || report.ImplementationAcceptance != "not-assessed" { t.Fatal("quality falsely claims acceptance") }
}

func TestQualityEvidencePreservesLimitsAndMatchesBusinessWords(t *testing.T) {
	if qualityNormalize("balance < -5") == qualityNormalize("balance > 5") { t.Fatal("limit direction or sign lost") }
	if !qualityRelated("Export order IDs", "Export visible orders") { t.Fatal("plural business words did not match") }
	if !qualityRelated("库存不足时发送提醒", "库存不足触发提醒") { t.Fatal("Chinese business words did not match") }
}

func TestCandidateQualityRejectsUnsupportedScope(t *testing.T) {
	cases := map[string]func(*requirement.GenerationRequest, *requirement.GenerationCandidate){
		"missing prohibition": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { r.Sources[0].Content += "\nNever export customer secrets." },
		"missing exception": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { r.Sources[0].Content += "\nAn empty result MUST contain CSV headers." },
		"wrong body": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { c.Artifacts[3].Content = strings.ReplaceAll(c.Artifacts[3].Content, "The system MUST export order IDs.", "The system MUST send inventory alerts.") },
		"unconfirmed source": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) {
			r.Sources = append(r.Sources, requirement.ConversationSource{Kind: "discussion", Path: "chat.md", Content: r.Sources[0].Content})
			c.Coverage[0].SourceID = "chat.md"
		},
		"wrong scenario": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { c.Coverage[0].Scenario = "Missing" },
		"source only in comment": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { c.Artifacts[3].Content = strings.ReplaceAll(c.Artifacts[3].Content, "The system MUST export order IDs.", "<!-- The system MUST export order IDs. -->\nThe system MUST do work.") },
		"source only in sibling": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) {
			c.Artifacts[3].Content = strings.ReplaceAll(c.Artifacts[3].Content, "The system MUST export order IDs.", "The system MUST export data.") + "\n#### Scenario: Other\nThe system MUST export order IDs.\n"
		},
		"template": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { c.Artifacts[1].Content += "\nThe system MUST generate the required OpenSpec artifacts from an approved Requirement.\n" },
		"off topic proposal": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { c.Artifacts[0].Content = "## Why\nInventory alerts.\n## What Changes\nLow stock warnings.\n" },
		"missing task mapping": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { c.Artifacts[2].Content += "- [ ] 1.3 Export extra data.\n" },
		"unresolved business": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { c.Unresolved = []requirement.GenerationIssue{{Kind: "business", Detail: "Export permission is unclear"}} },
		"duplicate heading": func(r *requirement.GenerationRequest, c *requirement.GenerationCandidate) { c.Artifacts[3].Content += "\n#### Scenario: Export visible orders\n- **WHEN** exporting\n- **THEN** return nothing\n" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			request, candidate := qualityFixture()
			mutate(&request, &candidate)
			report, err := candidateQuality(request, candidate)
			if err == nil || report.CoverageValid || len(report.Issues) == 0 { t.Fatalf("accepted unsupported scope: %+v", report) }
		})
	}
}

func TestCandidateQualityAllowsDocumentedEngineeringDeferral(t *testing.T) {
	request, candidate := qualityFixture()
	candidate.Unresolved = []requirement.GenerationIssue{{Kind: "engineering", Detail: "Choose export buffering later"}}
	candidate.Artifacts[1].Content += "\n%% Choose export buffering later\n"
	candidate.Artifacts[2].Content += "\n## TODO\nChoose export buffering later.\n## Verification\nReview export buffering before implementation.\n"
	report, err := candidateQuality(request, candidate)
	if err != nil || !report.CoverageValid || len(report.Issues) != 1 || report.Issues[0].Blocking { t.Fatalf("%+v: %v", report, err) }
}

func TestCandidateQualityUsesConfirmedReplacement(t *testing.T) {
	request, candidate := qualityFixture()
	request.Confirmation.Facts = append(request.Confirmation.Facts, requirement.GenerationFact{ID: "retired-rule", Reference: requirement.CoverageReference{Quote: "The system MUST export private emails."}})
	request.Confirmation.Facts[0].Supersedes = "retired-rule"
	request.Sources[0].Content += "\nThe system MUST export private emails."
	if _, err := candidateQuality(request, candidate); err != nil { t.Fatal(err) }
	candidate.Artifacts[3].Content += "\nThe system MUST export private emails.\n"
	if _, err := candidateQuality(request, candidate); err == nil { t.Fatal("retired rule accepted") }
}

func TestModelQualityFailureFallsBackAndPersistsReport(t *testing.T) {
	request, candidate := qualityFixture()
	record := requirement.GenerationRecord{SchemaVersion: 1, RequestID: request.RequestID, InputDigest: request.InputDigest}
	calls := 0
	sawRejection := false
	_, err := GenerateCandidate(context.Background(), request, &record, []ai.ArtifactProfile{{ID: "one"}, {ID: "two"}}, GenerationHooks{
		CheckContext: func() error { return nil },
		Save: func(saved requirement.GenerationRecord) error { if len(saved.Report.Issues) != 0 && !saved.Report.CoverageValid { sawRejection = true }; return nil },
		NewProvider: func(ai.Config) (ai.Provider, error) {
			return generationFakeProvider{generate: func(context.Context, ai.Request) (ai.Response, error) {
				calls++
				value := candidate
				if calls == 1 { value.Coverage = nil }
				body, err := json.Marshal(value)
				return ai.Response{FinalOutput: string(body)}, err
			}}, nil
		},
	})
	if err != nil || calls != 2 || !sawRejection || !record.Report.CoverageValid || record.State != requirement.GenerationValidating { t.Fatalf("%+v: %v", record, err) }
}

func TestAgentQualityRejectionPersistsReport(t *testing.T) {
	id, runtime := handoffFixture(t)
	first, err := PrepareCandidate(context.Background(), runtime, id, "export-task", PreparationOptions{LoadProfiles: func() ([]ai.ArtifactProfile, error) { return nil, nil }})
	if err != nil { t.Fatal(err) }
	_, candidate := qualityFixture()
	candidate.RequestID, candidate.InputDigest, candidate.Coverage = first.Request.RequestID, first.Request.InputDigest, nil
	body, err := json.Marshal(candidate)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile("rejected.json", body, 0o600); err != nil { t.Fatal(err) }
	result, err := PrepareCandidate(context.Background(), runtime, id, "export-task", PreparationOptions{CandidatePath: "rejected.json"})
	if err == nil || result.Record.State != requirement.GenerationAwaitingAgent || result.Record.Candidate != nil { t.Fatal("invalid candidate retained") }
	var saved requirement.GenerationRecord
	if err := readGenerationJSON(result.ResultPath, &saved, 8*maxGenerationOutput); err != nil { t.Fatal(err) }
	if !saved.Report.StructureValid || saved.Report.CoverageValid || len(saved.Report.Issues) == 0 { t.Fatal("missing persisted rejection evidence") }
}
