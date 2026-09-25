package openspecgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/requirement"
)

func businessCandidate() (requirement.GenerationRequest, requirement.GenerationCandidate) {
	request := requirement.GenerationRequest{
		SchemaVersion: requirement.GenerationSchemaVersion, RequestID: "generation-1", InputDigest: "snapshot",
		ActiveFacts: []string{"export-rule", "access-rule"},
	}
	request.Sources = []requirement.ConversationSource{{Kind: "requirement-plan", Path: "plan.md", Content: "The system MUST export order IDs.\nThe system MUST enforce export permission."}}
	request.Confirmation.Facts = []requirement.GenerationFact{
		{ID: "export-rule", Reference: requirement.CoverageReference{Quote: "The system MUST export order IDs."}},
		{ID: "access-rule", Reference: requirement.CoverageReference{Quote: "The system MUST enforce export permission."}},
	}
	candidate := requirement.GenerationCandidate{
		SchemaVersion: request.SchemaVersion, RequestID: request.RequestID, InputDigest: request.InputDigest,
		Artifacts: []requirement.GenerationArtifact{
			{Path: "proposal.md", Content: "## Why\n\nOperators need order exports.\n\n## What Changes\n\nExport visible orders.\n\n## Capabilities\n\n### New Capabilities\n- `order-export`: CSV export.\n\n### Modified Capabilities\n- `order-access`: Restrict export access.\n"},
			{Path: "design.md", Content: "## Context\n\nOrder operations.\n\n## Goals / Non-Goals\n\nExport orders.\n\n## Decisions\n\nUse CSV.\n"},
			{Path: "tasks.md", Content: "## Export\n\n- [ ] 1.1 Export visible orders.\n- [ ] 1.2 Enforce export permission.\n"},
			{Path: "specs/order-export/spec.md", Content: "## ADDED Requirements\n\n### Requirement: Export orders\n\nThe system MUST export order IDs.\n\n#### Scenario: Export visible orders\n\n- **WHEN** an operator exports orders\n- **THEN** the CSV contains visible order IDs\n"},
			{Path: "specs/order-access/spec.md", Content: "## MODIFIED Requirements\n\n### Requirement: Export permission\n\nThe system MUST enforce export permission.\n\n#### Scenario: Denied export\n\n- **WHEN** a user has no export permission\n- **THEN** access is denied\n"},
		},
		Coverage: []requirement.GenerationCoverage{
			{SourceID: "export-rule", Path: "specs/order-export/spec.md", Requirement: "Export orders", Scenario: "Export visible orders", Task: "1.1"},
			{SourceID: "access-rule", Path: "specs/order-access/spec.md", Requirement: "Export permission", Scenario: "Denied export", Task: "1.2"},
		},
	}
	for _, artifact := range candidate.Artifacts { request.Targets = append(request.Targets, requirement.GenerationFile{Path: artifact.Path}) }
	for _, mapping := range append([]requirement.GenerationCoverage{}, candidate.Coverage...) { mapping.SourceID = "plan.md"; candidate.Coverage = append(candidate.Coverage, mapping) }
	return request, candidate
}

func TestRenderBusinessCandidatePreservesMultipleCapabilitiesAndMappings(t *testing.T) {
	request, candidate := businessCandidate()
	artifacts, err := Render(Input{Request: &request, Candidate: &candidate})
	if err != nil { t.Fatal(err) }
	if len(artifacts) != len(candidate.Artifacts) { t.Fatal("artifact set changed") }
	for index, artifact := range artifacts {
		if artifact.Path != candidate.Artifacts[index].Path || string(artifact.Content) != candidate.Artifacts[index].Content {
			t.Fatalf("business content changed: %s", artifact.Path)
		}
	}
	if len(artifacts[3].Coverage) != 2 || artifacts[3].Coverage[0] != candidate.Coverage[0] ||
		len(artifacts[4].Coverage) != 2 || artifacts[4].Coverage[0] != candidate.Coverage[1] { t.Fatal("source mappings lost") }
}

func TestRenderBusinessCandidateRejectsInvalidStructure(t *testing.T) {
	cases := map[string]func(*requirement.GenerationCandidate){
		"missing spec": func(c *requirement.GenerationCandidate) { c.Artifacts = c.Artifacts[:3] },
		"duplicate": func(c *requirement.GenerationCandidate) { c.Artifacts = append(c.Artifacts, c.Artifacts[3]) },
		"case alias": func(c *requirement.GenerationCandidate) { a := c.Artifacts[3]; a.Path = "specs/ORDER-EXPORT/spec.md"; c.Artifacts = append(c.Artifacts, a) },
		"traversal": func(c *requirement.GenerationCandidate) { c.Artifacts[3].Path = "specs/../spec.md" },
		"absolute": func(c *requirement.GenerationCandidate) { c.Artifacts[3].Path = "C:/specs/export/spec.md" },
		"backslash": func(c *requirement.GenerationCandidate) { c.Artifacts[3].Path = `specs\export\spec.md` },
		"device": func(c *requirement.GenerationCandidate) { c.Artifacts[3].Path = "specs/CON/spec.md" },
		"extra directory": func(c *requirement.GenerationCandidate) { c.Artifacts[3].Path = "specs/orders/export/spec.md" },
		"protected": func(c *requirement.GenerationCandidate) { c.Artifacts[3].Path = "task.toml" },
		"stale identity": func(c *requirement.GenerationCandidate) { c.InputDigest = "old" },
		"missing source": func(c *requirement.GenerationCandidate) { c.Coverage[0].SourceID = "unknown" },
		"wrong requirement": func(c *requirement.GenerationCandidate) { c.Coverage[0].Requirement = "Missing" },
		"wrong scenario": func(c *requirement.GenerationCandidate) { c.Coverage[0].Scenario = "Denied export" },
		"wrong task": func(c *requirement.GenerationCandidate) { c.Coverage[0].Task = "1.9" },
		"missing delta": func(c *requirement.GenerationCandidate) { c.Artifacts[3].Content = strings.ReplaceAll(c.Artifacts[3].Content, "## ADDED Requirements", "## Requirements") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			request, candidate := businessCandidate()
			mutate(&candidate)
			if _, err := RenderCandidate(request, candidate); err == nil { t.Fatal("invalid candidate accepted") }
		})
	}
}

func TestWriteMissingRejectsAllTargetsBeforeWriting(t *testing.T) {
	request, candidate := businessCandidate()
	artifacts, err := RenderCandidate(request, candidate)
	if err != nil { t.Fatal(err) }
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "specs", "order-access", "spec.md"), 0o755); err != nil { t.Fatal(err) }
	if _, err := WriteMissing(root, artifacts); err == nil { t.Fatal("directory target accepted") }
	if _, err := os.Stat(filepath.Join(root, "proposal.md")); !os.IsNotExist(err) { t.Fatalf("wrote before preflight completed: %v", err) }
}

func TestWriteMissingRejectsSymlinkParent(t *testing.T) {
	request, candidate := businessCandidate()
	artifacts, err := RenderCandidate(request, candidate)
	if err != nil { t.Fatal(err) }
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "specs")); err != nil { t.Skipf("symlink unavailable: %v", err) }
	if _, err := WriteMissing(root, artifacts); err == nil { t.Fatal("symlink accepted") }
	if _, err := os.Stat(filepath.Join(root, "proposal.md")); !os.IsNotExist(err) { t.Fatalf("wrote before symlink rejection: %v", err) }
}
