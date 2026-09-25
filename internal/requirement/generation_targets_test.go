package requirement

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerationTargetsRejectUndeclaredMissingAndChangedBaselines(t *testing.T) {
	for _, mode := range []string{"missing", "duplicate", "undeclared", "new-target-created", "new-stable-created", "unrelated"} {
		t.Run(mode, func(t *testing.T) {
			request := generationFixture(t)
			candidate := candidateFor(request)
			switch mode {
			case "missing": candidate.Artifacts = candidate.Artifacts[1:]
			case "duplicate": candidate.Artifacts[0] = candidate.Artifacts[1]
			case "undeclared": candidate.Artifacts[0].Path = "specs/unapproved/spec.md"
			default:
				path := "openspec/changes/orders-task/specs/orders/spec.md"
				if mode == "new-stable-created" { path = "openspec/specs/orders/spec.md" }
				if mode == "unrelated" { path = "openspec/specs/other/spec.md" }
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { t.Fatal(err) }
				if err := os.WriteFile(path, []byte("Third-party content"), 0o600); err != nil { t.Fatal(err) }
			}
			err := ValidateGenerationContext(request, candidate, GenerationConfirmation{})
			if mode == "unrelated" && err != nil { t.Fatal(err) }
			if mode != "unrelated" && err == nil { t.Fatal("invalid candidate or stale target accepted") }
		})
	}
}

func TestGenerationTargetsRequireApprovedDeclarations(t *testing.T) {
	for _, plan := range []string{"No targets", "## OpenSpec Targets\n- orders: inferred", "## OpenSpec Targets\n- CON: new", "## OpenSpec Targets\n- orders: new\n- ORDERS: modified", "## OpenSpec Targets\n- ../orders: new"} {
		if _, err := planGenerationTargets(plan); err == nil { t.Fatalf("accepted invalid target Plan: %s", plan) }
	}
}

func TestGenerationModifiedTargetRequiresStableSpec(t *testing.T) {
	request := generationFixture(t)
	if err := os.WriteFile("plan.md", []byte("Export CSV.\n\n## OpenSpec Targets\n- orders: modified\n- alerts: new\n"), 0o600); err != nil { t.Fatal(err) }
	if _, _, err := Capture("orders", "requirement-plan", "plan.md"); err != nil { t.Fatal(err) }
	// Re-enter the approval fixture explicitly; Capture does not reset a promoted status.
	meta, err := Read("orders")
	if err != nil { t.Fatal(err) }
	meta.Status = "DECIDED"
	if err := Write(meta); err != nil { t.Fatal(err) }
	if _, err := Approve("orders", "APPROVED", "owner", "Confirm modified capability"); err != nil { t.Fatal(err) }
	if _, err := PrepareGeneration("orders", request.TaskID, "modified-1", GenerationOptions{}); err == nil { t.Fatal("accepted missing modified capability") }
	path := "openspec/specs/orders/spec.md"
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte("Existing order behavior"), 0o600); err != nil { t.Fatal(err) }
	updated, err := PrepareGeneration("orders", request.TaskID, "modified-2", GenerationOptions{})
	if err != nil { t.Fatal(err) }
	if len(updated.Targets) != 5 || len(updated.StableSpecs) != 2 { t.Fatal("missing declared target baseline") }
	if err := ValidateGenerationContext(updated, candidateFor(updated), GenerationConfirmation{}); err != nil { t.Fatal(err) }
}
