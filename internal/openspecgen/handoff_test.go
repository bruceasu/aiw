package openspecgen

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/ai"
	"aiw/internal/requirement"
)

func handoffFixture(t *testing.T) (string, string) {
	t.Helper()
	t.Chdir(t.TempDir())
	meta, err := requirement.Create("export", "Export orders")
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile("plan.md", []byte("The system MUST export order IDs.\nThe system MUST enforce export permission.\n\n## OpenSpec Targets\n- order-export: new\n- order-access: modified\n"), 0o600); err != nil { t.Fatal(err) }
	if _, _, err := requirement.Capture(meta.ID, "requirement-plan", "plan.md"); err != nil { t.Fatal(err) }
	if _, err := requirement.Approve(meta.ID, "APPROVED", "owner", "Confirmed"); err != nil { t.Fatal(err) }
	if _, _, err := requirement.StartPromotion(meta.ID, "export-task"); err != nil { t.Fatal(err) }
	if err := os.MkdirAll("openspec/specs/order-access", 0o700); err != nil { t.Fatal(err) }
	if err := os.WriteFile("openspec/specs/order-access/spec.md", []byte("Existing access requirements"), 0o600); err != nil { t.Fatal(err) }
	return meta.ID, filepath.Join(".ai", "export-task")
}

type businessOrchestrationFixture struct {
	requirementID string
	title         string
	taskID        string
	plan          string
	stable        []string
	candidate     func(requirement.GenerationRequest) requirement.GenerationCandidate
	confirmation  func(requirement.GenerationRequest) requirement.GenerationConfirmation
}

// prepareBusinessOrchestration intentionally uses normal Requirement records
// and OpenSpec targets. The fixtures describe business behavior, not AIW
// workflow behavior, so generation quality must preserve their rules.
func prepareBusinessOrchestration(t *testing.T, fixture businessOrchestrationFixture) (string, string, requirement.GenerationConfirmation) {
	t.Helper()
	t.Chdir(t.TempDir())
	meta, err := requirement.Create(fixture.requirementID, fixture.title)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile("plan.md", []byte(fixture.plan), 0o600); err != nil { t.Fatal(err) }
	if _, _, err := requirement.Capture(meta.ID, "requirement-plan", "plan.md"); err != nil { t.Fatal(err) }
	if _, err := requirement.Approve(meta.ID, "APPROVED", "owner", "Confirmed business scope"); err != nil { t.Fatal(err) }
	if _, _, err := requirement.StartPromotion(meta.ID, fixture.taskID); err != nil { t.Fatal(err) }
	for _, capability := range fixture.stable {
		path := filepath.Join("openspec", "specs", capability, "spec.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { t.Fatal(err) }
		if err := os.WriteFile(path, []byte("Existing business behavior."), 0o600); err != nil { t.Fatal(err) }
	}
	template, err := requirement.PrepareGeneration(meta.ID, fixture.taskID, "confirmation-template", requirement.GenerationOptions{})
	if err != nil { t.Fatal(err) }
	confirmation := requirement.GenerationConfirmation{}
	if fixture.confirmation != nil { confirmation = fixture.confirmation(template) }
	return meta.ID, filepath.Join(".ai", fixture.taskID), confirmation
}

func TestBusinessRequirementOrchestrationFixtures(t *testing.T) {
	refund := businessOrchestrationFixture{
		requirementID: "refund-policy", title: "Revise invoice refund policy", taskID: "refund-policy-task",
		plan: "The system MUST refund no more than $1000 per day.\nThe system MUST NOT refund more than the settled invoice total.\nWhen an invoice was partially refunded, the system MUST return the remaining refundable amount.\nA successful refund MUST record the invoice ID and refunded amount.\nThe revised rule replaces the previous daily refund limit.\nThe system MUST refund no more than $500 per day.\n\n## OpenSpec Targets\n- invoice-refund: new\n- refund-access: modified\n",
		stable: []string{"refund-access"},
		confirmation: func(request requirement.GenerationRequest) requirement.GenerationConfirmation {
			source := request.Sources[0]
			ref := func(quote string) requirement.CoverageReference { return requirement.CoverageReference{Source: source.Path, Digest: source.Digest, Quote: quote} }
			old := requirement.GenerationFact{ID: "daily-limit-old", Key: "daily-limit", Reference: ref("The system MUST refund no more than $500 per day.")}
			replacement := ref("The revised rule replaces the previous daily refund limit.")
			current := requirement.GenerationFact{ID: "daily-limit-current", Key: "daily-limit", Reference: ref("The system MUST refund no more than $1000 per day."), Supersedes: old.ID, ReplacementEvidence: &replacement}
			return requirement.GenerationConfirmation{RequirementID: request.Requirement.ID, Revision: request.Requirement.Revision, Facts: []requirement.GenerationFact{old, current}}
		},
		candidate: refundPolicyCandidate,
	}
	allocation := businessOrchestrationFixture{
		requirementID: "inventory-allocation", title: "Allocate warehouse stock", taskID: "inventory-allocation-task",
		plan: "The system MUST reserve stock before confirming an order.\nThe system MUST NOT allocate stock from quarantined inventory.\nWhen stock is insufficient, the system MUST create a backorder without decrementing available stock.\nA confirmed allocation MUST record warehouse, SKU, and quantity.\n\n## OpenSpec Targets\n- inventory-allocation: new\n- inventory-audit: new\n",
		candidate: inventoryAllocationCandidate,
	}
	for _, fixture := range []businessOrchestrationFixture{refund, allocation} {
		t.Run(fixture.requirementID, func(t *testing.T) {
			id, runtime, confirmation := prepareBusinessOrchestration(t, fixture)
			initial, err := PrepareCandidate(context.Background(), runtime, id, fixture.taskID, PreparationOptions{Confirmation: confirmation, LoadProfiles: func() ([]ai.ArtifactProfile, error) { return nil, nil }})
			if err != nil || initial.Record.State != requirement.GenerationAwaitingAgent { t.Fatalf("prepare fixture: %+v: %v", initial.Record, err) }
			candidate := fixture.candidate(initial.Request)
			body, err := json.Marshal(candidate)
			if err != nil { t.Fatal(err) }
			if err := os.WriteFile("business-candidate.json", body, 0o600); err != nil { t.Fatal(err) }
			result, err := PrepareCandidate(context.Background(), runtime, id, fixture.taskID, PreparationOptions{Confirmation: confirmation, CandidatePath: "business-candidate.json"})
			if err != nil || result.Record.State != requirement.GenerationValidating || !result.Record.Report.CoverageValid { t.Fatalf("business candidate rejected: %+v: %v", result.Record.Report, err) }
		})
	}

	t.Run("unconfirmed-input-is-not-approved-scope", func(t *testing.T) {
		id, runtime, _ := prepareBusinessOrchestration(t, allocation)
		unconfirmed := requirement.GenerationConfirmation{RequirementID: id, Facts: []requirement.GenerationFact{{ID: "chat-only", Reference: requirement.CoverageReference{Quote: "A user requested a priority allocation."}}}}
		if _, err := PrepareCandidate(context.Background(), runtime, id, allocation.taskID, PreparationOptions{Confirmation: unconfirmed, LoadProfiles: func() ([]ai.ArtifactProfile, error) { return nil, nil }}); err == nil { t.Fatal("accepted unconfirmed input as approved scope") }
	})
}

func refundPolicyCandidate(request requirement.GenerationRequest) requirement.GenerationCandidate {
	return requirement.GenerationCandidate{
		SchemaVersion: request.SchemaVersion, RequestID: request.RequestID, InputDigest: request.InputDigest,
		Artifacts: []requirement.GenerationArtifact{
			{Path: "proposal.md", Content: "## Why\n\nFinance needs a safe refund policy.\n\n## What Changes\n\nApply invoice refund limits and records.\n\n## Capabilities\n\n### New Capabilities\n- `invoice-refund`: Refund invoices.\n\n### Modified Capabilities\n- `refund-access`: Protect refund totals.\n"},
			{Path: "design.md", Content: "## Context\n\nInvoice refund operations.\n\n## Goals / Non-Goals\n\nKeep refund limits and audit records.\n\n## Decisions\n\nUse settled invoice totals.\n"},
			{Path: "tasks.md", Content: "## Refund policy\n\n- [ ] 1.1 Apply the daily refund limit.\n- [ ] 1.2 Reject refunds above settled totals.\n- [ ] 1.3 Return remaining refundable amount.\n- [ ] 1.4 Record refund details.\n- [ ] 1.5 Apply the revised refund policy.\n"},
			{Path: "specs/invoice-refund/spec.md", Content: "## ADDED Requirements\n\n### Requirement: Daily refund limit\n\nThe system MUST refund no more than $1000 per day.\n\n#### Scenario: Reject refund above daily limit\n\n- **WHEN** a refund exceeds the daily limit\n- **THEN** The system MUST refund no more than $1000 per day.\n\n### Requirement: Settled invoice refund\n\nThe system MUST NOT refund more than the settled invoice total.\n\n#### Scenario: Reject refund above settled total\n\n- **WHEN** an operator requests too much\n- **THEN** The system MUST NOT refund more than the settled invoice total.\n\n### Requirement: Partially refunded invoice\n\nThe system MUST return the remaining refundable amount for a partially refunded invoice.\n\n#### Scenario: Return remaining refundable amount\n\n- **WHEN** an invoice was partially refunded\n- **THEN** When an invoice was partially refunded, the system MUST return the remaining refundable amount.\n"},
			{Path: "specs/refund-access/spec.md", Content: "## MODIFIED Requirements\n\n### Requirement: Refund audit record\n\nA successful refund MUST record the invoice ID and refunded amount.\n\n#### Scenario: Record successful refund\n\n- **WHEN** a refund succeeds\n- **THEN** A successful refund MUST record the invoice ID and refunded amount.\n\n### Requirement: Revised refund policy\n\nThe system MUST apply the revised refund policy.\n\n#### Scenario: Apply revised daily limit\n\n- **WHEN** the daily limit is evaluated\n- **THEN** The revised rule replaces the previous daily refund limit.\n"},
		},
		Coverage: []requirement.GenerationCoverage{
			{SourceID: "daily-limit-current", Path: "specs/invoice-refund/spec.md", Requirement: "Daily refund limit", Scenario: "Reject refund above daily limit", Task: "1.1"},
			{SourceID: request.Sources[0].Path, Path: "specs/invoice-refund/spec.md", Requirement: "Daily refund limit", Scenario: "Reject refund above daily limit", Task: "1.1"},
			{SourceID: request.Sources[0].Path, Path: "specs/invoice-refund/spec.md", Requirement: "Settled invoice refund", Scenario: "Reject refund above settled total", Task: "1.2"},
			{SourceID: request.Sources[0].Path, Path: "specs/invoice-refund/spec.md", Requirement: "Partially refunded invoice", Scenario: "Return remaining refundable amount", Task: "1.3"},
			{SourceID: request.Sources[0].Path, Path: "specs/refund-access/spec.md", Requirement: "Refund audit record", Scenario: "Record successful refund", Task: "1.4"},
			{SourceID: request.Sources[0].Path, Path: "specs/refund-access/spec.md", Requirement: "Revised refund policy", Scenario: "Apply revised daily limit", Task: "1.5"},
		},
	}
}

func inventoryAllocationCandidate(request requirement.GenerationRequest) requirement.GenerationCandidate {
	return requirement.GenerationCandidate{
		SchemaVersion: request.SchemaVersion, RequestID: request.RequestID, InputDigest: request.InputDigest,
		Artifacts: []requirement.GenerationArtifact{
			{Path: "proposal.md", Content: "## Why\n\nOperations need safe warehouse allocations.\n\n## What Changes\n\nReserve usable stock and record allocations.\n\n## Capabilities\n\n### New Capabilities\n- `inventory-allocation`: Reserve warehouse stock.\n- `inventory-audit`: Record allocations.\n"},
			{Path: "design.md", Content: "## Context\n\nWarehouse order allocation.\n\n## Goals / Non-Goals\n\nPrevent invalid stock allocation.\n\n## Decisions\n\nReserve stock before confirmation.\n"},
			{Path: "tasks.md", Content: "## Inventory allocation\n\n- [ ] 1.1 Reserve stock before confirmation.\n- [ ] 1.2 Exclude quarantined inventory.\n- [ ] 1.3 Create a backorder for insufficient stock.\n- [ ] 1.4 Record allocation details.\n"},
			{Path: "specs/inventory-allocation/spec.md", Content: "## ADDED Requirements\n\n### Requirement: Stock reservation\n\nThe system MUST reserve stock before confirming an order.\n\n#### Scenario: Reserve before confirmation\n\n- **WHEN** an order is confirmed\n- **THEN** The system MUST reserve stock before confirming an order.\n\n### Requirement: Quarantined inventory\n\nThe system MUST NOT allocate stock from quarantined inventory.\n\n#### Scenario: Exclude quarantined stock\n\n- **WHEN** inventory is quarantined\n- **THEN** The system MUST NOT allocate stock from quarantined inventory.\n\n### Requirement: Insufficient stock\n\nThe system MUST create a backorder when stock is insufficient.\n\n#### Scenario: Create backorder without decrement\n\n- **WHEN** stock is insufficient\n- **THEN** When stock is insufficient, the system MUST create a backorder without decrementing available stock.\n"},
			{Path: "specs/inventory-audit/spec.md", Content: "## ADDED Requirements\n\n### Requirement: Allocation audit record\n\nA confirmed allocation MUST record warehouse, SKU, and quantity.\n\n#### Scenario: Record confirmed allocation\n\n- **WHEN** an allocation is confirmed\n- **THEN** A confirmed allocation MUST record warehouse, SKU, and quantity.\n"},
		},
		Coverage: []requirement.GenerationCoverage{
			{SourceID: request.Sources[0].Path, Path: "specs/inventory-allocation/spec.md", Requirement: "Stock reservation", Scenario: "Reserve before confirmation", Task: "1.1"},
			{SourceID: request.Sources[0].Path, Path: "specs/inventory-allocation/spec.md", Requirement: "Quarantined inventory", Scenario: "Exclude quarantined stock", Task: "1.2"},
			{SourceID: request.Sources[0].Path, Path: "specs/inventory-allocation/spec.md", Requirement: "Insufficient stock", Scenario: "Create backorder without decrement", Task: "1.3"},
			{SourceID: request.Sources[0].Path, Path: "specs/inventory-audit/spec.md", Requirement: "Allocation audit record", Scenario: "Record confirmed allocation", Task: "1.4"},
		},
	}
}

func TestHandoffResumeCandidateAndExplicitRegeneration(t *testing.T) {
	id, runtime := handoffFixture(t)
	loads := 0
	options := PreparationOptions{LoadProfiles: func() ([]ai.ArtifactProfile, error) { loads++; return nil, nil }}
	first, err := PrepareCandidate(context.Background(), runtime, id, "export-task", options)
	if err != nil { t.Fatal(err) }
	if first.Record.State != requirement.GenerationAwaitingAgent { t.Fatal(first.Record.State) }
	for _, name := range []string{"input.json", "request.md", "candidate.json", "result.json"} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(first.RequestPath), name)); err != nil { t.Fatal(err) }
	}
	body, err := os.ReadFile(first.RequestPath)
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), first.Request.InputDigest) || !strings.Contains(string(body), "--candidate") { t.Fatal("handoff lacks identity or resume command") }
	resumed, err := PrepareCandidate(context.Background(), runtime, id, "export-task", options)
	if err != nil { t.Fatal(err) }
	if resumed.Request.RequestID != first.Request.RequestID || loads != 1 { t.Fatal("resume replenished the model budget") }
	_, candidate := businessCandidate()
	candidate.RequestID, candidate.InputDigest = first.Request.RequestID, first.Request.InputDigest
	for index := range candidate.Coverage { candidate.Coverage[index].SourceID = first.Request.Sources[0].Path }
	encoded, err := json.Marshal(candidate)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile("submitted.json", encoded, 0o600); err != nil { t.Fatal(err) }
	submitted, err := PrepareCandidate(context.Background(), runtime, id, "export-task", PreparationOptions{CandidatePath: "submitted.json"})
	if err != nil { t.Fatal(err) }
	if submitted.Record.State != requirement.GenerationValidating || submitted.Record.Method != "agent" || !submitted.Record.Report.StructureValid || !submitted.Record.Report.CoverageValid { t.Fatal("candidate falsely accepted or missing validation evidence") }
	if _, err := os.Stat("openspec/changes/export-task/proposal.md"); !os.IsNotExist(err) { t.Fatal("candidate preparation wrote formal artifacts") }
	options.Regenerate = true
	next, err := PrepareCandidate(context.Background(), runtime, id, "export-task", options)
	if err != nil { t.Fatal(err) }
	if next.Request.RequestID == first.Request.RequestID || loads != 2 { t.Fatal("explicit regeneration did not create a fresh budget") }
	if _, err := os.Stat(first.ResultPath); err != nil { t.Fatal("old audit removed", err) }
	if _, err := PrepareCandidate(context.Background(), runtime, id, "export-task", PreparationOptions{CandidatePath: "submitted.json"}); err == nil { t.Fatal("old candidate accepted by new generation") }
}

func TestHandoffBlocksStaleContextBeforeLoadingProfiles(t *testing.T) {
	id, runtime := handoffFixture(t)
	options := PreparationOptions{LoadProfiles: func() ([]ai.ArtifactProfile, error) { return nil, nil }}
	first, err := PrepareCandidate(context.Background(), runtime, id, "export-task", options)
	if err != nil { t.Fatal(err) }
	if err := os.MkdirAll("openspec/changes/export-task/specs/order-export", 0o700); err != nil { t.Fatal(err) }
	if err := os.WriteFile("openspec/changes/export-task/specs/order-export/spec.md", []byte("Human edit"), 0o600); err != nil { t.Fatal(err) }
	options.LoadProfiles = func() ([]ai.ArtifactProfile, error) { t.Fatal("stale request loaded profiles"); return nil, nil }
	result, err := PrepareCandidate(context.Background(), runtime, id, "export-task", options)
	if err == nil || result.Record.State != requirement.GenerationBlocked || result.Request.RequestID != first.Request.RequestID { t.Fatal("stale baseline was not blocked") }
}
