package requirement

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const GenerationSchemaVersion = 1

type GenerationState string

const (
	GenerationPrepared GenerationState = "prepared"
	GenerationGenerating GenerationState = "generating"
	GenerationAwaitingAgent GenerationState = "awaiting-agent"
	GenerationValidating GenerationState = "validating"
	GenerationApplying GenerationState = "applying"
	GenerationAccepted GenerationState = "accepted"
	GenerationBlocked GenerationState = "blocked"
)

// GenerationFact is supplied by a trusted confirmation adapter, never by the
// artifact candidate. Key identifies a rule; competing values require an
// explicit confirmed replacement. Source text alone is not confirmation.
type GenerationFact struct {
	ID string `json:"id"`
	Key string `json:"key"`
	Reference CoverageReference `json:"reference"`
	Supersedes string `json:"supersedes,omitempty"`
	ReplacementEvidence *CoverageReference `json:"replacement_evidence,omitempty"`
}

type GenerationConfirmation struct {
	RequirementID string `json:"requirement_id"`
	Revision int `json:"revision"`
	Facts []GenerationFact `json:"facts"`
}

type GenerationFile struct {
	Path string `json:"path"`
	Intent string `json:"intent,omitempty"`
	Exists bool `json:"exists"`
	Digest string `json:"digest,omitempty"`
	Content string `json:"content,omitempty"`
}

// GenerationRequest is a value snapshot. JSON encoding is also its canonical
// digest encoding; callers must retain the full snapshot, not just its hash.
type GenerationRequest struct {
	SchemaVersion int `json:"schema_version"`
	RequestID string `json:"request_id"`
	InputDigest string `json:"input_digest"`
	Requirement ConversationIdentity `json:"requirement"`
	TaskID string `json:"task_id"`
	Sources []ConversationSource `json:"sources"`
	Confirmation GenerationConfirmation `json:"confirmation"`
	ActiveFacts []string `json:"active_facts"`
	StableSpecs []GenerationFile `json:"stable_specs"`
	Targets []GenerationFile `json:"targets"`
}

type GenerationArtifact struct {
	Path string `json:"path"`
	Content string `json:"content"`
}

type GenerationCoverage struct {
	SourceID string `json:"source_id"`
	Path string `json:"path"`
	Requirement string `json:"requirement"`
	Scenario string `json:"scenario"`
	Task string `json:"task,omitempty"`
}

type GenerationIssue struct {
	SourceID string `json:"source_id,omitempty"`
	Kind string `json:"kind"`
	Detail string `json:"detail"`
	Blocking bool `json:"blocking"`
}

type GenerationCandidate struct {
	SchemaVersion int `json:"schema_version"`
	RequestID string `json:"request_id"`
	InputDigest string `json:"input_digest"`
	Artifacts []GenerationArtifact `json:"artifacts"`
	Coverage []GenerationCoverage `json:"coverage"`
	Unresolved []GenerationIssue `json:"unresolved"`
}

// Coverage reports are separate from candidate claims and from implementation
// acceptance. Only the validator may populate verified coverage.
type GenerationReport struct {
	CoverageMethod string `json:"coverage_method,omitempty"`
	DesignReadiness string `json:"design_readiness,omitempty"`
	ImplementationAcceptance string `json:"implementation_acceptance,omitempty"`
	StructureValid bool `json:"structure_valid"`
	CoverageValid bool `json:"coverage_valid"`
	VerifiedCoverage []GenerationCoverage `json:"verified_coverage"`
	Issues []GenerationIssue `json:"issues"`
}

// GenerationApplication records the ownership proof and progress for one
// protected application attempt. It belongs to the generation request so a
// later process can resume without treating a human edit as generated output.
type GenerationApplication struct {
	Files []GenerationApplicationFile `json:"files"`
}

type GenerationApplicationFile struct {
	Path           string `json:"path"`
	OriginalDigest string `json:"original_digest"`
	WrittenDigest  string `json:"written_digest"`
	Ownership      string `json:"ownership"`
	Status         string `json:"status"`
}

// GenerationRecord belongs to a request, not to Workflow Core or its lease.
type GenerationRecord struct {
	SchemaVersion int `json:"schema_version"`
	RequestID string `json:"request_id"`
	InputDigest string `json:"input_digest"`
	State GenerationState `json:"state"`
	Method string `json:"method,omitempty"`
	Report GenerationReport `json:"report"`
	Diagnostics []string `json:"diagnostics,omitempty"`
	ModelBudgetPrepared bool `json:"model_budget_prepared,omitempty"`
	ModelAttempts []GenerationModelAttempt `json:"model_attempts,omitempty"`
	Candidate *GenerationCandidate `json:"candidate,omitempty"`
	Application *GenerationApplication `json:"application,omitempty"`
}

// A started attempt is consumed even if the process exits before saving its
// result. Only a new generation request can obtain a fresh model budget.
type GenerationModelAttempt struct {
	Report *GenerationReport `json:"report,omitempty"`
	ID string `json:"id"`
	Profile string `json:"profile,omitempty"`
	Provider string `json:"provider"`
	Model string `json:"model"`
	Endpoint string `json:"endpoint,omitempty"`
	Command string `json:"command,omitempty"`
	TimeoutSeconds int `json:"timeout_seconds"`
	Status string `json:"status"`
	Diagnostic string `json:"diagnostic,omitempty"`
}

type GenerationOptions struct {
	Confirmation GenerationConfirmation
	StableSpecPaths []string
	TargetPaths []string
}

// approvalSourceDigest binds approval to the registered capture set. Promotion
// changes revision/status without changing the approved business context.
func approvalSourceDigest(meta Meta) string {
	b, _ := json.Marshal(struct {
		ID string
		Title string
		Artifacts map[string]Artifact
	}{meta.ID, meta.Title, meta.Artifacts})
	return digest(b)
}

// PrepareGeneration reads bounded local sources and baselines without writing
// lifecycle data or generated artifacts. Missing legacy approval bindings are
// reported, never silently upgraded to proof of approval.
func PrepareGeneration(id, taskID, requestID string, options GenerationOptions) (GenerationRequest, error) {
	result := GenerationRequest{}
	for _, value := range []string{id, taskID, requestID} {
		if !ValidID(value) || value == "." || value == ".." {
			return result, errors.New("invalid generation identity")
		}
	}
	reader, err := newContextReader(ConversationContextOptions{})
	if err != nil { return result, err }
	defer reader.root.Close()
	meta, dir, err := reader.readRequirement(id)
	if err != nil { return result, err }
	if meta.Approval.Status != "APPROVED" || meta.Approval.By == "" || meta.Approval.At == "" || meta.Approval.Reason == "" ||
		(meta.Status != "APPROVED" && meta.Status != "PROMOTED") {
		return result, errors.New("generation requires an active approved Requirement")
	}
	if meta.Promotion.TaskID != taskID {
		return result, errors.New("generation Task does not match the promoted Requirement")
	}
	if meta.Approval.SourceDigest == "" || meta.Approval.SourceDigest != approvalSourceDigest(meta) {
		return result, errors.New("approved source binding is missing or stale; review approval context")
	}
	if _, ok := meta.Artifacts["requirement-plan"]; !ok {
		return result, errors.New("approved requirement-plan source is missing")
	}
	result = GenerationRequest{SchemaVersion: GenerationSchemaVersion, RequestID: requestID, TaskID: taskID,
		Requirement: ConversationIdentity{ID: meta.ID, Title: meta.Title, Status: meta.Status, Revision: meta.Revision, Approval: meta.Approval}}
	kinds := make([]string, 0, len(meta.Artifacts))
	for kind := range meta.Artifacts { kinds = append(kinds, kind) }
	sort.Strings(kinds)
	for _, kind := range kinds {
		source := ConversationSource{Kind: kind, Path: filepath.ToSlash(filepath.Join(dir, artifactFiles[kind])),
			Purpose: "Captured evidence; only the approved Plan and confirmed fragments define scope", RecordedDigest: meta.Artifacts[kind].Digest}
		if err := source.load(reader, true); err != nil { return result, err }
		if strings.TrimSpace(source.Content) == "" { return result, fmt.Errorf("empty generation source: %s", kind) }
		result.Sources = append(result.Sources, source)
	}
	// Copy caller-owned slices and reference pointers before canonicalizing.
	encoded, err := json.Marshal(options.Confirmation)
	if err != nil { return result, err }
	if err := json.Unmarshal(encoded, &result.Confirmation); err != nil { return result, err }
	if err := result.resolveFacts(); err != nil { return result, err }
	if err := result.freezeTargets(reader, options); err != nil { return result, err }
	result.InputDigest, err = result.computedDigest()
	return result, err
}

func (request GenerationRequest) computedDigest() (string, error) {
	request.InputDigest = ""
	b, err := json.Marshal(request)
	if err != nil { return "", err }
	return digest(b), nil
}

func (request *GenerationRequest) resolveFacts() error {
	confirmation := &request.Confirmation
	if len(confirmation.Facts) == 0 && confirmation.RequirementID == "" && confirmation.Revision == 0 { return nil }
	if confirmation.RequirementID != request.Requirement.ID || confirmation.Revision != request.Requirement.Revision {
		return errors.New("confirmed generation fragments are stale")
	}
	if len(confirmation.Facts) > maxContextReferences { return errors.New("too many confirmed generation fragments") }
	validRef := func(ref CoverageReference) bool {
		for _, source := range request.Sources {
			if ref.Source == source.Path && ref.Digest == source.Digest && strings.TrimSpace(ref.Quote) != "" && strings.Contains(source.Content, ref.Quote) { return true }
		}
		return false
	}
	sort.Slice(confirmation.Facts, func(i, j int) bool { return confirmation.Facts[i].ID < confirmation.Facts[j].ID })
	facts := make(map[string]GenerationFact)
	for _, fact := range confirmation.Facts {
		if fact.ID == "" || strings.TrimSpace(fact.Key) == "" || !validRef(fact.Reference) { return fmt.Errorf("missing confirmed source: %s", fact.ID) }
		if _, exists := facts[fact.ID]; exists { return fmt.Errorf("duplicate confirmed source: %s", fact.ID) }
		facts[fact.ID] = fact
	}
	replaced := make(map[string]bool)
	for _, fact := range confirmation.Facts {
		if fact.Supersedes == "" {
			if fact.ReplacementEvidence != nil { return fmt.Errorf("replacement has no prior source: %s", fact.ID) }
			continue
		}
		old, exists := facts[fact.Supersedes]
		if !exists || old.Key != fact.Key || replaced[old.ID] || fact.ReplacementEvidence == nil || !validRef(*fact.ReplacementEvidence) {
			return fmt.Errorf("missing or conflicting confirmed replacement: %s", fact.ID)
		}
		replaced[old.ID] = true
		seen := map[string]bool{fact.ID: true}
		for next := fact.Supersedes; next != ""; next = facts[next].Supersedes {
			if seen[next] { return fmt.Errorf("cyclic confirmed replacement: %s", fact.ID) }
			seen[next] = true
		}
	}
	active := make(map[string]string)
	for _, fact := range confirmation.Facts {
		if replaced[fact.ID] { continue }
		if prior, exists := active[fact.Key]; exists { return fmt.Errorf("unresolved source conflict: %s and %s", prior, fact.ID) }
		active[fact.Key] = fact.ID
		request.ActiveFacts = append(request.ActiveFacts, fact.ID)
	}
	return nil
}

func generationFiles(reader *contextReader, base string, paths []string, targets bool) ([]GenerationFile, error) {
	paths = append([]string{}, paths...)
	sort.Strings(paths)
	var files []GenerationFile
	seen := make(map[string]bool)
	for _, path := range paths {
		local, err := contextLocalPath(path)
		if err != nil { return nil, err }
		canonical := filepath.ToSlash(local)
		if canonical != path { return nil, fmt.Errorf("noncanonical generation path: %s", path) }
		parts := strings.Split(path, "/")
		allowed := len(parts) == 2 && parts[1] == "spec.md"
		if targets {
			allowed = path == "proposal.md" || path == "design.md" || path == "tasks.md" ||
				(len(parts) == 3 && parts[0] == "specs" && parts[2] == "spec.md")
		}
		if !allowed { return nil, fmt.Errorf("unsupported generation path: %s", path) }
		key := strings.ToLower(path)
		if seen[key] { return nil, fmt.Errorf("duplicate generation path: %s", path) }
		seen[key] = true
		file := GenerationFile{Path: path, Digest: "absent"}
		content, err := reader.read(filepath.Join(base, local))
		if errors.Is(err, os.ErrNotExist) && targets {
			files = append(files, file)
			continue
		}
		if err != nil { return nil, fmt.Errorf("generation file %s: %w", path, err) }
		file.Exists, file.Content, file.Digest = true, string(content), digest(content)
		files = append(files, file)
	}
	return files, nil
}

// ValidateGenerationContext only validates identity and freshness. Structural
// and semantic candidate checks must still run before any artifact is accepted.
// The caller supplies current trusted confirmation records, not candidate data.
func ValidateGenerationContext(request GenerationRequest, candidate GenerationCandidate, current GenerationConfirmation) error {
	if err := ValidateGenerationTargets(request, candidate); err != nil { return err }
	return ValidateGenerationFreshness(request, current)
}

// ValidateGenerationFreshness also serves pre-call checks, before a candidate exists.
func ValidateGenerationFreshness(request GenerationRequest, current GenerationConfirmation) error {
	computed, err := request.computedDigest()
	if err != nil { return err }
	if request.SchemaVersion != GenerationSchemaVersion || request.InputDigest == "" || request.InputDigest != computed {
		return errors.New("generation request is corrupt or unsupported")
	}
	options := GenerationOptions{Confirmation: current}
	latest, err := PrepareGeneration(request.Requirement.ID, request.TaskID, request.RequestID, options)
	if err != nil { return err }
	if latest.InputDigest != request.InputDigest { return errors.New("generation sources, approval, confirmation, specs or target baseline changed") }
	return nil
}
