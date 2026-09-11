package openspecgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderAndValidateProducesSpecDrivenArtifacts(t *testing.T) {
	requirement := "## Confirmed Workflow\n\n1. Prepare artifacts.\n\nThe system MUST prepare artifacts."
	artifacts, err := Render(Input{Title: "Example", Requirement: requirement, Capability: "example"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(artifacts); err != nil {
		t.Fatal(err)
	}
}

func TestRenderUsesTitleForRequirementHeading(t *testing.T) {
	requirement := "# Approved Plan\n\n## Confirmed Workflow\n\n1. Prepare artifacts.\n\nThe system MUST prepare artifacts."
	artifacts, err := Render(Input{Title: "Example", Requirement: requirement, Capability: "example"})
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if artifact.Path != "specs/example/spec.md" {
			continue
		}
		content := string(artifact.Content)
		if !strings.Contains(content, "### Requirement: Example\n\n") {
			t.Fatalf("unexpected requirement heading: %s", content)
		}
		if strings.Contains(content, "### Requirement: # Approved Plan") {
			t.Fatalf("plan content was used as a requirement heading: %s", content)
		}
		return
	}
	t.Fatal("generated spec artifact not found")
}

func TestRenderCreatesTasksFromConfirmedWorkflow(t *testing.T) {
	requirement := "## Confirmed Workflow\n\n1. First approved step.\n2. Second approved step.\n\n## Scope\n\n1. This must not become a task."
	artifacts, err := Render(Input{Title: "Example", Requirement: requirement, Capability: "example"})
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if artifact.Path != "tasks.md" {
			continue
		}
		content := string(artifact.Content)
		if !strings.Contains(content, "- [ ] 1.1 First approved step.") || !strings.Contains(content, "- [ ] 1.2 Second approved step.") || strings.Contains(content, "This must not become a task.") {
			t.Fatalf("tasks do not reflect the confirmed workflow: %s", content)
		}
		return
	}
	t.Fatal("generated tasks artifact not found")
}

func TestRenderRejectsRequirementWithoutConfirmedWorkflow(t *testing.T) {
	_, err := Render(Input{
		Title:       "Example",
		Requirement: "The system MUST prepare artifacts.",
		Capability:  "example",
	})
	if err == nil || !strings.Contains(err.Error(), "Confirmed Workflow") {
		t.Fatalf("Render() error = %v, want Confirmed Workflow error", err)
	}
}

func TestWriteMissingPreservesAuthoredArtifact(t *testing.T) {
	root := t.TempDir()
	authored := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(authored, []byte("authored"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifacts, err := Render(Input{Title: "Example", Requirement: "## Confirmed Workflow\n\n1. Prepare artifacts.", Capability: "example"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteMissing(root, artifacts); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(authored)
	if err != nil || string(content) != "authored" {
		t.Fatalf("authored artifact changed: %q (%v)", content, err)
	}
}
