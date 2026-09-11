package openspecgen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidateWithCLI runs OpenSpec validation when an executable is available.
// available=false is a normal offline condition, not a validation failure.
func ValidateWithCLI(ctx context.Context, changeID string) (available bool, err error) {
	bin, err := exec.LookPath("openspec")
	if err != nil {
		return false, nil
	}
	cmd := exec.CommandContext(ctx, bin, "validate", changeID, "--type", "change", "--json", "--no-interactive")
	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		return true, fmt.Errorf("openspec validation failed: %s: %w", strings.TrimSpace(string(output)), runErr)
	}
	return true, nil
}

// Input contains the approved context needed to render a spec-driven change.
type Input struct {
	Title       string
	Requirement string
	Capability  string
}

// Artifact is a canonical OpenSpec file produced by the renderer.
type Artifact struct {
	Path    string
	Content []byte
}

// Render returns the schema-shaped artifact set without touching the file
// system. Callers decide whether to preserve existing files and how to report
// validation or partial-write failures.
func Render(input Input) ([]Artifact, error) {
	if input.Title == "" || input.Requirement == "" || input.Capability == "" {
		return nil, fmt.Errorf("title, requirement, and capability are required")
	}
	tasks, err := renderTasks(input.Requirement)
	if err != nil {
		return nil, err
	}
	capability := input.Capability
	return []Artifact{
		{Path: "proposal.md", Content: []byte(fmt.Sprintf("## Why\n\nAfter promotion, the approved Requirement context must remain available for implementation.\n\n## What Changes\n\n- Generate schema-valid OpenSpec artifacts from the approved Requirement.\n\n## Capabilities\n\n### New Capabilities\n- `%s`: %s\n\n### Modified Capabilities\n\n## Impact\n\nAIW requirement promotion and specification workflows.\n", capability, input.Title))},
		{Path: "design.md", Content: []byte("## Context\n\nAIW must generate OpenSpec artifacts consistently when the OpenSpec CLI is optional.\n\n## Goals / Non-Goals\n\n**Goals:**\n- Produce canonical spec-driven artifacts.\n- Preserve authored files through caller-controlled idempotent writes.\n\n**Non-Goals:**\n- Starting implementation or release workflows.\n\n## Decisions\n\nUse centralized templates for proposal, design, delta specs, and tasks.\n\n## Risks / Trade-offs\n\n- [Schema evolution] -> Keep schema rules centralized and validate structure locally.\n")},
		{Path: fmt.Sprintf("specs/%s/spec.md", capability), Content: []byte(fmt.Sprintf("## ADDED Requirements\n\n### Requirement: %s\n\nThe system MUST generate the required OpenSpec artifacts from an approved Requirement.\n\n#### Scenario: Generate artifacts\n\n- **WHEN** the workflow prepares a promoted Requirement\n- **THEN** it produces proposal, design, delta spec, and tasks files\n\n#### Scenario: Preserve authored content\n\n- **WHEN** a target artifact already exists\n- **THEN** the workflow MUST preserve its content\n", input.Title))},
		{Path: "tasks.md", Content: []byte(tasks)},
	}, nil
}

func renderTasks(requirement string) (string, error) {
	lines := strings.Split(requirement, "\n")
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == "## Confirmed Workflow" {
			start = index + 1
			break
		}
	}
	if start == -1 {
		return "", fmt.Errorf("requirement plan has no Confirmed Workflow section")
	}
	itemPattern := regexp.MustCompile(`^\s*([0-9]+)\.\s+(.+?)\s*$`)
	items := []string{}
	for _, line := range lines[start:] {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		match := itemPattern.FindStringSubmatch(line)
		if match != nil {
			items = append(items, match[2])
		}
	}
	if len(items) == 0 {
		return "", fmt.Errorf("requirement plan has no numbered Confirmed Workflow items")
	}
	var builder strings.Builder
	builder.WriteString("## 1. Confirmed Workflow\n\n")
	for index, item := range items {
		fmt.Fprintf(&builder, "- [ ] 1.%d %s\n", index+1, item)
	}
	builder.WriteString("\n## 2. Verification\n\n- [ ] 2.1 Verify the implementation satisfies the approved acceptance criteria.\n")
	return builder.String(), nil
}

// Validate checks the schema-critical structure without requiring OpenSpec.
func Validate(artifacts []Artifact) error {
	if len(artifacts) == 0 {
		return fmt.Errorf("no OpenSpec artifacts")
	}
	seen := make(map[string]bool, len(artifacts))
	for _, artifact := range artifacts {
		seen[artifact.Path] = true
		content := string(artifact.Content)
		switch {
		case artifact.Path == "proposal.md":
			if !strings.Contains(content, "## Why") || !strings.Contains(content, "## What Changes") {
				return fmt.Errorf("proposal.md must contain Why and What Changes sections")
			}
		case artifact.Path == "design.md":
			if !strings.Contains(content, "## Context") || !strings.Contains(content, "## Goals / Non-Goals") || !strings.Contains(content, "## Decisions") {
				return fmt.Errorf("design.md is missing required sections")
			}
		case strings.HasPrefix(artifact.Path, "specs/") && strings.HasSuffix(artifact.Path, "/spec.md"):
			if !strings.Contains(content, "## ADDED Requirements") && !strings.Contains(content, "## MODIFIED Requirements") && !strings.Contains(content, "## REMOVED Requirements") && !strings.Contains(content, "## RENAMED Requirements") {
				return fmt.Errorf("%s has no delta sections", artifact.Path)
			}
			if !strings.Contains(content, "### Requirement:") || !strings.Contains(content, "#### Scenario:") {
				return fmt.Errorf("%s must contain requirements and level-four scenarios", artifact.Path)
			}
			if !strings.Contains(content, " MUST ") && !strings.Contains(content, " SHALL ") {
				return fmt.Errorf("%s has no normative MUST or SHALL requirement", artifact.Path)
			}
		case artifact.Path == "tasks.md":
			if !regexp.MustCompile(`(?m)^- \[[ xX]\] [0-9]+(?:\.[0-9]+)+\s+.+$`).MatchString(content) {
				return fmt.Errorf("tasks.md has no numbered checkbox items")
			}
		}
	}
	for _, required := range []string{"proposal.md", "design.md", "tasks.md"} {
		if !seen[required] {
			return fmt.Errorf("missing required artifact: %s", required)
		}
	}
	return nil
}

// WriteMissing writes only absent artifacts and returns the paths it created.
// Existing files are intentionally left untouched so retries are safe.
func WriteMissing(root string, artifacts []Artifact) ([]string, error) {
	created := []string{}
	for _, artifact := range artifacts {
		path := filepath.Join(root, filepath.FromSlash(artifact.Path))
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return created, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return created, err
		}
		if err := os.WriteFile(path, artifact.Content, 0o644); err != nil {
			return created, err
		}
		created = append(created, artifact.Path)
	}
	return created, nil
}
