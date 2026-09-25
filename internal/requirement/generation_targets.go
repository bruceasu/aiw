package requirement

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The approved Plan uses an exact section and one declaration per line:
// ## OpenSpec Targets
// - order-export: new
// - order-access: modified
// Do not infer capability names from prose or candidate output.
func planGenerationTargets(plan string) (map[string]string, error) {
	targets := map[string]string{}
	seen := map[string]bool{}
	inside, found := false, false
	entry := regexp.MustCompile("^- `?([a-zA-Z0-9][a-zA-Z0-9_-]*)`?: (new|modified)$")
	reserved := regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])$`)
	for _, line := range strings.Split(plan, "\n") {
		line = strings.TrimSpace(line)
		if line == "## OpenSpec Targets" {
			if found { return nil, errors.New("duplicate OpenSpec Targets section") }
			inside, found = true, true
			continue
		}
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") { inside = false }
		if !inside || line == "" { continue }
		match := entry.FindStringSubmatch(line)
		if match == nil { return nil, errors.New("OpenSpec Targets requires '- capability-id: new' or '- capability-id: modified'") }
		id := match[1]
		if reserved.MatchString(id) || seen[strings.ToLower(id)] { return nil, fmt.Errorf("invalid or duplicate OpenSpec target: %s", id) }
		seen[strings.ToLower(id)] = true
		targets["specs/"+id+"/spec.md"] = match[2]
	}
	if len(targets) == 0 || len(targets)*2+3 > maxContextReferences {
		return nil, errors.New("approved Plan requires a bounded, nonempty OpenSpec Targets section; update and approve the Plan")
	}
	return targets, nil
}

func (request *GenerationRequest) freezeTargets(reader *contextReader, options GenerationOptions) error {
	plan := ""
	for _, source := range request.Sources { if source.Kind == "requirement-plan" { plan = source.Content } }
	intents, err := planGenerationTargets(plan)
	if err != nil { return err }
	// Legacy options may assert a subset but cannot authorize additional targets.
	for _, path := range options.TargetPaths { if intents[path] == "" { return fmt.Errorf("target not declared by approved Plan: %s", path) } }
	for _, path := range options.StableSpecPaths { if intents["specs/"+path] == "" { return fmt.Errorf("stable spec not declared by approved Plan: %s", path) } }
	paths := []string{"proposal.md", "design.md", "tasks.md"}
	for path := range intents { paths = append(paths, path) }
	request.Targets, err = generationFiles(reader, filepath.Join("openspec/changes", request.TaskID), paths, true)
	if err != nil { return err }
	for index := range request.Targets {
		target := &request.Targets[index]
		target.Intent = intents[target.Path]
		if target.Intent == "" { target.Intent = "base"; continue }
		stable := GenerationFile{Path: strings.TrimPrefix(target.Path, "specs/"), Digest: "absent", Intent: target.Intent}
		body, readErr := reader.read(filepath.Join("openspec/specs", stable.Path))
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) { return readErr }
		if readErr == nil { stable.Exists, stable.Content, stable.Digest = true, string(body), digest(body) }
		if target.Intent == "new" && (stable.Exists || target.Exists) { return fmt.Errorf("new OpenSpec target already exists: %s", target.Path) }
		if target.Intent == "modified" && !stable.Exists { return fmt.Errorf("modified OpenSpec capability is missing: %s", stable.Path) }
		request.StableSpecs = append(request.StableSpecs, stable)
	}
	return nil
}

// ValidateGenerationTargets enforces exact manifest membership for both providers
// and Agent candidates. It never adds baselines from candidate file names.
func ValidateGenerationTargets(request GenerationRequest, candidate GenerationCandidate) error {
	if candidate.SchemaVersion != GenerationSchemaVersion || candidate.RequestID != request.RequestID || candidate.InputDigest != request.InputDigest {
		return errors.New("candidate belongs to a different or stale generation request")
	}
	if len(request.Targets) < 4 || len(candidate.Artifacts) != len(request.Targets) { return errors.New("candidate must contain every frozen target exactly once") }
	remaining := map[string]bool{}
	for _, target := range request.Targets {
		if remaining[target.Path] { return errors.New("duplicate target in generation manifest") }
		remaining[target.Path] = true
	}
	for _, artifact := range candidate.Artifacts {
		if !remaining[artifact.Path] { return fmt.Errorf("candidate target is undeclared or duplicated: %s", artifact.Path) }
		delete(remaining, artifact.Path)
	}
	return nil
}
