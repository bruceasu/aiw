package openspecgen

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/requirement"
)

// ApplyHooks keep protected writes independent from the Workflow Core. The
// acceptance stage supplies the durable record adapter and context check.
type ApplyHooks struct {
	Save         func(requirement.GenerationRecord) error
	CheckContext func() error
}

type ApplyResult struct {
	Written []string
	Pending []string
}

// AcceptanceHooks add the managed checklist projection to protected artifact
// application. The caller supplies the projection because Workflow Core owns
// its storage and lifecycle; this package only controls the generation record.
type AcceptanceHooks struct {
	ApplyHooks
	SyncChecklist func() error
}

// ApplyCandidate applies an already validated candidate without accepting it
// or synchronizing Work Items. It writes only content that this request owns,
// a complete known legacy placeholder, or a currently absent target. A saved
// application manifest makes an interrupted write resumable without replacing
// intervening human edits.
func ApplyCandidate(root string, request requirement.GenerationRequest, record *requirement.GenerationRecord, hooks ApplyHooks) (ApplyResult, error) {
	result := ApplyResult{}
	if record == nil || hooks.Save == nil || hooks.CheckContext == nil {
		return result, errors.New("protected application requires a record, durable checkpoint and context validator")
	}
	if record.State != requirement.GenerationValidating && record.State != requirement.GenerationApplying {
		return result, fmt.Errorf("generation state %s cannot apply a candidate", record.State)
	}
	if record.Candidate == nil {
		return result, errors.New("generation has no candidate to apply")
	}
	if err := hooks.CheckContext(); err != nil {
		return result, err
	}
	artifacts, err := RenderCandidate(request, *record.Candidate)
	if err != nil {
		return result, err
	}
	if err := Validate(artifacts); err != nil {
		return result, err
	}
	if record.Application == nil {
		application, err := prepareApplication(root, request, artifacts)
		if err != nil {
			return result, err
		}
		record.Application = application
		record.State = requirement.GenerationApplying
		if err := hooks.Save(*record); err != nil {
			return result, err
		}
	}
	if err := validateApplication(record.Application, artifacts); err != nil {
		return result, err
	}
	contents := make(map[string][]byte, len(artifacts))
	for _, artifact := range artifacts {
		contents[artifact.Path] = artifact.Content
	}
	for index := range record.Application.Files {
		entry := &record.Application.Files[index]
		content := contents[entry.Path]
		current, err := artifactDigest(root, entry.Path)
		if err != nil {
			return result, err
		}
		switch current {
		case entry.WrittenDigest:
			if entry.Status != "written" {
				entry.Status = "written"
				if err := hooks.Save(*record); err != nil {
					return result, err
				}
			}
			result.Written = append(result.Written, entry.Path)
		case entry.OriginalDigest:
			if err := replaceArtifact(root, entry.Path, content); err != nil {
				return result, err
			}
			current, err = artifactDigest(root, entry.Path)
			if err != nil {
				return result, err
			}
			if current != entry.WrittenDigest {
				return result, fmt.Errorf("artifact changed while applying: %s", entry.Path)
			}
			entry.Status = "written"
			if err := hooks.Save(*record); err != nil {
				return result, err
			}
			result.Written = append(result.Written, entry.Path)
		default:
			return result, fmt.Errorf("artifact has an unowned or human edit: %s", entry.Path)
		}
	}
	for _, entry := range record.Application.Files {
		if entry.Status != "written" {
			result.Pending = append(result.Pending, entry.Path)
		}
	}
	return result, nil
}

// AcceptCandidate completes one validated request while its generation lock is
// still held by the caller. A failed checklist sync deliberately leaves the
// record applying so a later prepare-spec call can resume without reapplying
// unrelated content or claiming the Requirement was promoted.
func AcceptCandidate(root string, request requirement.GenerationRequest, record *requirement.GenerationRecord, hooks AcceptanceHooks) (ApplyResult, error) {
	result := ApplyResult{}
	if hooks.SyncChecklist == nil {
		return result, errors.New("generation acceptance requires checklist synchronization")
	}
	result, err := ApplyCandidate(root, request, record, hooks.ApplyHooks)
	if err != nil {
		return result, err
	}
	if len(result.Pending) != 0 {
		return result, errors.New("generation application has pending artifacts")
	}
	if err := hooks.SyncChecklist(); err != nil {
		record.Diagnostics = append(record.Diagnostics, "synchronize generated Work Items: "+err.Error())
		if saveErr := hooks.Save(*record); saveErr != nil {
			return result, fmt.Errorf("synchronize generated Work Items: %v; save recovery state: %w", err, saveErr)
		}
		return result, fmt.Errorf("synchronize generated Work Items: %w", err)
	}
	record.State = requirement.GenerationAccepted
	if err := hooks.Save(*record); err != nil {
		return result, err
	}
	return result, nil
}

func prepareApplication(root string, request requirement.GenerationRequest, artifacts []Artifact) (*requirement.GenerationApplication, error) {
	targets := make(map[string]requirement.GenerationFile, len(request.Targets))
	for _, target := range request.Targets {
		targets[target.Path] = target
	}
	application := &requirement.GenerationApplication{Files: make([]requirement.GenerationApplicationFile, 0, len(artifacts))}
	for _, artifact := range artifacts {
		if err := validateTarget(root, artifact.Path); err != nil {
			return nil, err
		}
		target, exists := targets[artifact.Path]
		if !exists {
			return nil, fmt.Errorf("artifact is absent from the frozen manifest: %s", artifact.Path)
		}
		current, err := artifactDigest(root, artifact.Path)
		if err != nil {
			return nil, err
		}
		written := generationDigest(artifact.Content)
		entry := requirement.GenerationApplicationFile{Path: artifact.Path, OriginalDigest: current, WrittenDigest: written, Status: "pending"}
		switch {
		case current == written:
			entry.Ownership, entry.Status = "matching-candidate", "written"
		case current == "absent" && target.Digest == "absent":
			entry.Ownership = "created"
		case current == target.Digest && knownLegacyPlaceholder(artifact.Path, artifact.Content, root):
			entry.Ownership = "known-legacy-template"
		default:
			return nil, fmt.Errorf("artifact has unowned existing content: %s", artifact.Path)
		}
		application.Files = append(application.Files, entry)
	}
	return application, nil
}

func validateApplication(application *requirement.GenerationApplication, artifacts []Artifact) error {
	if application == nil || len(application.Files) != len(artifacts) {
		return errors.New("generation application manifest does not match candidate")
	}
	expected := make(map[string]string, len(artifacts))
	for _, artifact := range artifacts {
		expected[artifact.Path] = generationDigest(artifact.Content)
	}
	seen := make(map[string]bool, len(application.Files))
	for _, entry := range application.Files {
		if entry.Path == "" || entry.OriginalDigest == "" || entry.WrittenDigest == "" ||
			(entry.Status != "pending" && entry.Status != "written") || seen[entry.Path] ||
			entry.WrittenDigest != expected[entry.Path] {
			return errors.New("generation application manifest is corrupt")
		}
		seen[entry.Path] = true
	}
	for _, artifact := range artifacts {
		if !seen[artifact.Path] {
			return errors.New("generation application manifest has a missing artifact")
		}
	}
	return nil
}

func artifactDigest(root, path string) (string, error) {
	if err := validateTarget(root, path); err != nil {
		return "", err
	}
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if errors.Is(err, os.ErrNotExist) {
		return "absent", nil
	}
	if err != nil {
		return "", err
	}
	return generationDigest(body), nil
}

func replaceArtifact(root, path string, content []byte) error {
	if err := validateTarget(root, path); err != nil {
		return err
	}
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := validateTarget(root, path); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".generation-*.tmp")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, target)
}

func generationDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum[:])
}

// The old promotion path creates exactly this generic checklist. Matching the
// complete body, rather than a heading or a substring, prevents user edits
// from being mistaken for generator-owned content.
const legacyTasksPlaceholder = "# Tasks\n\n- [x] Create Requirement handoff\n- [x] Generate proposal, design, capability spec, and implementation checklist after promotion\n- [ ] Review generated OpenSpec artifacts\n- [ ] Implement the resulting change\n\n## TODO\n\n- [ ] Review generated OpenSpec artifacts\n- [ ] Implement the resulting change\n\n## Verification\n\n- [ ] Required OpenSpec files exist\n- [ ] Existing authored artifacts are preserved\n"

func knownLegacyPlaceholder(path string, candidate []byte, root string) bool {
	if path != "tasks.md" || string(candidate) == "" {
		return false
	}
	current, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return false
	}
	return strings.ReplaceAll(string(current), "\r\n", "\n") == legacyTasksPlaceholder
}
