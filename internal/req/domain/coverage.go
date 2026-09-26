package requirement

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// CoverageReference binds a literal excerpt to the input version seen by the
// model. A valid reference alone is not evidence of human confirmation.
type CoverageReference struct {
	Source string `json:"source"`
	Digest string `json:"digest"`
	Quote  string `json:"quote"`
}

// CoverageItem describes one dimension. All conclusions remain candidates even
// when validation succeeds; status never authorizes a durable action.
type CoverageItem struct {
	Dimension string              `json:"dimension"`
	Status    string              `json:"status"`
	Conclusion string             `json:"conclusion"`
	Sources   []CoverageReference `json:"sources"`
	Question  string              `json:"question"`
	Impact    string              `json:"impact"`
	NextStep  string              `json:"next_step"`
	Reason    string              `json:"reason"`
	ImpactKind string             `json:"impact_kind,omitempty"`
	Options   []DiscoveryOption   `json:"options,omitempty"`
}

// CoverageAssessment is a versioned, non-authoritative model output contract.
type CoverageAssessment struct {
	PlanReview    *PlanReview    `json:"plan_review,omitempty"`
	Version       int            `json:"version"`
	RequirementID string         `json:"requirement_id"`
	Revision      int            `json:"revision"`
	Items         []CoverageItem `json:"items"`
}

// CoverageResult retains rejected output for the caller's existing Session
// records. Assessment is nil on any error, preventing partial acceptance.
type CoverageResult struct {
	Raw         string
	Assessment  *CoverageAssessment
	Diagnostics []string
}

var coverageDimensions = [...]string{
	"roles", "problem", "current_workflow", "goals", "scope", "rules",
	"exceptions", "data", "permissions", "dependencies", "acceptance",
}

// ParseCoverage validates one complete JSON object without file writes or model
// calls. confirmed must come from human-confirmed conclusions at the trusted
// caller boundary, never from the model or from capture status alone. Until
// that boundary has such evidence, pass nil and keep conclusions needs_input.
// This conservative first version requires confirmation for every resolved
// dimension. Validity is structural, not a business-readiness decision.
func ParseCoverage(raw string, snapshot ConversationContext, confirmed []CoverageReference) (CoverageResult, error) {
	result := CoverageResult{Raw: raw}
	reject := func(err error) (CoverageResult, error) {
		result.Diagnostics = []string{err.Error()}
		return result, err
	}
	if len(raw) > defaultContextBytes {
		return reject(errors.New("coverage output exceeds 64 KiB"))
	}
	if !snapshot.SourcesLoaded || strings.TrimSpace(snapshot.UserInput) == "" {
		return reject(errors.New("coverage requires complete context"))
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var assessment CoverageAssessment
	if err := decoder.Decode(&assessment); err != nil {
		return reject(fmt.Errorf("decode coverage: %w", err))
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return reject(errors.New("coverage must contain exactly one JSON object"))
	}
	id, revision := "", 0
	if snapshot.Requirement != nil {
		id, revision = snapshot.Requirement.ID, snapshot.Requirement.Revision
	}
	if assessment.Version != 1 || assessment.RequirementID != id || assessment.Revision != revision {
		return reject(errors.New("coverage version or Requirement binding is invalid"))
	}
	if len(assessment.Items) != len(coverageDimensions) {
		return reject(errors.New("coverage must include each discovery dimension once"))
	}
	trusted := make(map[CoverageReference]bool)
	for _, ref := range confirmed {
		if !snapshot.validCoverageReference(ref) || ref.Source == "user-input" || ref.Source == "candidate" {
			return reject(errors.New("confirmed evidence is unavailable, stale or unconfirmed input"))
		}
		trusted[ref] = true
	}
	seen := make(map[string]bool)
	for _, item := range assessment.Items {
		known := false
		for _, dimension := range coverageDimensions {
			known = known || dimension == item.Dimension
		}
		if !known || seen[item.Dimension] {
			return reject(fmt.Errorf("unknown or duplicate coverage dimension: %s", item.Dimension))
		}
		seen[item.Dimension] = true
		if err := snapshot.validateCoverageItem(item, trusted); err != nil {
			return reject(fmt.Errorf("coverage %s: %w", item.Dimension, err))
		}
	}
	result.Assessment = &assessment
	return result, nil
}

// validCoverageReference excludes method instructions and omitted sources from
// business evidence. Quotes and fingerprints must match the actual snapshot.
func (snapshot ConversationContext) validCoverageReference(ref CoverageReference) bool {
	content := ""
	switch ref.Source {
	case "user-input":
		content = snapshot.UserInput
	case "candidate":
		if snapshot.Candidate != nil {
			id, revision := "", 0
			if snapshot.Requirement != nil {
				id, revision = snapshot.Requirement.ID, snapshot.Requirement.Revision
			}
			if snapshot.Candidate.RequirementID == id && snapshot.Candidate.Revision == revision {
				content = snapshot.Candidate.Content
			}
		}
	default:
		for _, source := range snapshot.Sources {
			if source.Path == ref.Source && source.Status == "loaded" && source.Kind != "method" {
				content = source.Content
				break
			}
		}
	}
	return strings.TrimSpace(ref.Quote) != "" && content != "" &&
		ref.Digest == digest([]byte(content)) && strings.Contains(content, ref.Quote)
}

func (snapshot ConversationContext) validateCoverageItem(item CoverageItem, trusted map[CoverageReference]bool) error {
	if err := snapshot.validateQuestionDetails(item); err != nil {
		return err
	}
	if strings.TrimSpace(item.Conclusion) == "" || strings.TrimSpace(item.Impact) == "" || strings.TrimSpace(item.NextStep) == "" {
		return errors.New("conclusion, impact and next_step are required")
	}
	if len(item.Sources) == 0 || len(item.Sources) > 8 {
		return errors.New("provide between one and eight source references")
	}
	seen := make(map[CoverageReference]bool)
	confirmed := false
	for _, ref := range item.Sources {
		if seen[ref] || !snapshot.validCoverageReference(ref) {
			return errors.New("source reference is duplicate, missing, stale or has an invalid quote")
		}
		seen[ref] = true
		confirmed = confirmed || trusted[ref]
	}
	switch item.Status {
	case "resolved":
		if !confirmed || strings.TrimSpace(item.Question) != "" {
			return errors.New("resolved needs confirmed evidence and no open question")
		}
	case "needs_input", "conflict":
		if strings.TrimSpace(item.Question) == "" {
			return errors.New("unresolved coverage needs an open question")
		}
		if item.Status == "conflict" && len(seen) < 2 {
			return errors.New("conflict needs both distinct evidence excerpts")
		}
	case "not_applicable":
		if strings.TrimSpace(item.Reason) == "" || strings.TrimSpace(item.Question) != "" {
			return errors.New("not_applicable needs a reason and no open question")
		}
	default:
		return errors.New("unsupported coverage status")
	}
	return nil
}
