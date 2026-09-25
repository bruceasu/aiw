package requirement

import (
	"os"
	"path/filepath"
	"testing"
)

func generationFixture(t *testing.T) GenerationRequest {
	t.Helper()
	t.Chdir(t.TempDir())
	if _, err := createExact("orders", "Order export"); err != nil { t.Fatal(err) }
	if err := os.WriteFile("plan.md", []byte("Export CSV. Old limit 10. New limit 20. Replace old limit with new limit.\n\n## OpenSpec Targets\n- orders: new\n"), 0o600); err != nil { t.Fatal(err) }
	if _, _, err := Capture("orders", "requirement-plan", "plan.md"); err != nil { t.Fatal(err) }
	if _, err := Approve("orders", "APPROVED", "owner", "Confirmed scope"); err != nil { t.Fatal(err) }
	if _, _, err := StartPromotion("orders", "orders-task"); err != nil { t.Fatal(err) }
	request, err := PrepareGeneration("orders", "orders-task", "generation-1", GenerationOptions{TargetPaths: []string{"specs/orders/spec.md"}})
	if err != nil { t.Fatal(err) }
	return request
}

func candidateFor(request GenerationRequest) GenerationCandidate {
	candidate := GenerationCandidate{SchemaVersion: GenerationSchemaVersion, RequestID: request.RequestID, InputDigest: request.InputDigest}
	for _, target := range request.Targets { candidate.Artifacts = append(candidate.Artifacts, GenerationArtifact{Path: target.Path}) }
	return candidate
}

func TestGenerationSnapshotFreshness(t *testing.T) {
	for _, change := range []string{"none", "source", "recapture", "target", "approval", "revision", "missing", "legacy", "candidate", "unbound-target"} {
		t.Run(change, func(t *testing.T) {
			request := generationFixture(t)
			candidate := candidateFor(request)
			switch change {
			case "source":
				if err := os.WriteFile(filepath.Join(Dir("orders"), "requirement-plan.md"), []byte("Changed without capture"), 0o600); err != nil { t.Fatal(err) }
			case "recapture":
				if err := os.WriteFile("plan.md", []byte("Different approved scope?"), 0o600); err != nil { t.Fatal(err) }
				if _, _, err := Capture("orders", "requirement-plan", "plan.md"); err != nil { t.Fatal(err) }
				if _, err := PrepareGeneration("orders", "orders-task", "generation-2", GenerationOptions{}); err == nil { t.Fatal("accepted recaptured scope under old approval") }
			case "target":
				path := filepath.Join("openspec", "changes", "orders-task")
				if err := os.MkdirAll(path, 0o700); err != nil { t.Fatal(err) }
				if err := os.WriteFile(filepath.Join(path, "tasks.md"), []byte("Human work"), 0o600); err != nil { t.Fatal(err) }
			case "approval", "revision", "legacy":
				meta, err := Read("orders")
				if err != nil { t.Fatal(err) }
				if change == "approval" { meta.Approval.Reason = "Changed decision" }
				if change == "revision" { meta.Revision++ }
				if change == "legacy" { meta.Approval.SourceDigest = "" }
				if err := Write(meta); err != nil { t.Fatal(err) }
			case "missing":
				if err := os.Remove(filepath.Join(Dir("orders"), "requirement-plan.md")); err != nil { t.Fatal(err) }
			case "candidate":
				candidate.RequestID = "other-request"
			case "unbound-target":
				candidate.Artifacts = []GenerationArtifact{{Path: "specs/other/spec.md", Content: "Not snapshotted"}}
			}
			err := ValidateGenerationContext(request, candidate, GenerationConfirmation{})
			if change == "none" && err != nil { t.Fatal(err) }
			if change != "none" && err == nil { t.Fatal("accepted stale or missing generation context") }
		})
	}
}

func TestGenerationConfirmedReplacement(t *testing.T) {
	request := generationFixture(t)
	source := request.Sources[0]
	ref := func(quote string) CoverageReference { return CoverageReference{Source: source.Path, Digest: source.Digest, Quote: quote} }
	old := GenerationFact{ID: "old", Key: "limit", Reference: ref("Old limit 10.")}
	replacement := ref("Replace old limit with new limit.")
	newFact := GenerationFact{ID: "new", Key: "limit", Reference: ref("New limit 20."), Supersedes: "old", ReplacementEvidence: &replacement}
	options := GenerationOptions{Confirmation: GenerationConfirmation{RequirementID: "orders", Revision: request.Requirement.Revision, Facts: []GenerationFact{old, newFact}}}
	current, err := PrepareGeneration("orders", "orders-task", "generation-2", options)
	if err != nil { t.Fatal(err) }
	if len(current.ActiveFacts) != 1 || current.ActiveFacts[0] != "new" { t.Fatalf("wrong active rules: %v", current.ActiveFacts) }
	if err := ValidateGenerationContext(current, candidateFor(current), options.Confirmation); err != nil { t.Fatal(err) }
	if err := ValidateGenerationContext(current, candidateFor(current), GenerationConfirmation{}); err == nil { t.Fatal("accepted missing confirmations") }
	options.Confirmation.Facts[1].Supersedes = ""
	options.Confirmation.Facts[1].ReplacementEvidence = nil
	if _, err := PrepareGeneration("orders", "orders-task", "generation-3", options); err == nil { t.Fatal("accepted competing rules based on order alone") }
	options.Confirmation.Facts = []GenerationFact{{ID: "unconfirmed", Key: "limit", Reference: ref("Unsaved chat")}}
	if _, err := PrepareGeneration("orders", "orders-task", "generation-4", options); err == nil { t.Fatal("accepted missing chat evidence") }
}
