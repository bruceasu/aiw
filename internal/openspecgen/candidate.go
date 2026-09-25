package openspecgen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"aiw/internal/requirement"
)

// RenderCandidate preserves business content and source mappings. This is a
// structural boundary, not approval, freshness, semantic coverage or acceptance.
// Callers must also validate the current generation context before applying it.
func RenderCandidate(request requirement.GenerationRequest, candidate requirement.GenerationCandidate) ([]Artifact, error) {
	if err := requirement.ValidateGenerationTargets(request, candidate); err != nil { return nil, err }
	if request.SchemaVersion != requirement.GenerationSchemaVersion || candidate.SchemaVersion != request.SchemaVersion ||
		request.RequestID == "" || request.InputDigest == "" || candidate.RequestID != request.RequestID || candidate.InputDigest != request.InputDigest {
		return nil, fmt.Errorf("candidate does not match generation request")
	}
	artifacts := make([]Artifact, 0, len(candidate.Artifacts))
	for _, item := range candidate.Artifacts {
		artifacts = append(artifacts, Artifact{Path: item.Path, Content: []byte(item.Content)})
	}
	if err := Validate(artifacts); err != nil { return nil, err }
	targets := make(map[string]int, len(artifacts))
	for index, artifact := range artifacts { targets[artifact.Path] = index }
	sources := make(map[string]bool)
	for _, source := range request.Sources { sources[source.Path] = true }
	for _, id := range request.ActiveFacts { sources[id] = true }
	for _, mapping := range candidate.Coverage {
		index, exists := targets[mapping.Path]
		if !sources[mapping.SourceID] || mapping.SourceID == "" || !exists || !strings.HasPrefix(mapping.Path, "specs/") {
			return nil, fmt.Errorf("invalid source mapping: %s -> %s", mapping.SourceID, mapping.Path)
		}
		body := section(string(artifacts[index].Content), "### Requirement: "+mapping.Requirement, "### ")
		if strings.TrimSpace(mapping.Requirement) == "" || strings.TrimSpace(mapping.Scenario) == "" ||
			section(body, "#### Scenario: "+mapping.Scenario, "#### ") == "" {
			return nil, fmt.Errorf("source mapping has no matching requirement/scenario: %s", mapping.SourceID)
		}
		if mapping.Task != "" {
			tasks := string(artifacts[targets["tasks.md"]].Content)
			pattern := `(?m)^- \[[ xX]\] ` + regexp.QuoteMeta(mapping.Task) + `\s+\S.*$`
			if !regexp.MustCompile(pattern).MatchString(tasks) {
				return nil, fmt.Errorf("source mapping has no matching task: %s", mapping.Task)
			}
		}
		artifacts[index].Coverage = append(artifacts[index].Coverage, mapping)
	}
	return artifacts, nil
}

// section matches whole headings and stops at a sibling or higher heading.
func section(content, heading, level string) string {
	var lines []string
	found := false
	depth := len(strings.TrimSpace(level))
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !found {
			if line == heading { found = true }
			continue
		}
		if strings.HasPrefix(line, "#") {
			count := len(line) - len(strings.TrimLeft(line, "#"))
			if count <= depth && len(line) > count && line[count] == ' ' { break }
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func validateArtifactPath(path string) error {
	if path == "proposal.md" || path == "design.md" || path == "tasks.md" { return nil }
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "specs" || parts[2] != "spec.md" ||
		!regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`).MatchString(parts[1]) {
		return fmt.Errorf("unsupported artifact path: %s", path)
	}
	// Reject Windows device aliases even when generating on another platform.
	if regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])$`).MatchString(parts[1]) {
		return fmt.Errorf("reserved capability path: %s", path)
	}
	return nil
}

// Reject links in every existing component, including the change root and
// ancestors. Missing descendants are permitted, but never followed through links.
func validateTarget(root, path string) error {
	if err := validateArtifactPath(path); err != nil { return err }
	abs, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil { return err }
	for current := abs; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil && !os.IsNotExist(err) { return err }
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 { return fmt.Errorf("artifact target contains symlink: %s", current) }
			if current == abs && !info.Mode().IsRegular() { return fmt.Errorf("artifact target is not a regular file: %s", current) }
			if current != abs && !info.IsDir() { return fmt.Errorf("artifact parent is not a directory: %s", current) }
		}
		if filepath.Dir(current) == current { break }
	}
	return nil
}
