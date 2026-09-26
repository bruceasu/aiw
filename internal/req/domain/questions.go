package requirement

import (
	"errors"
	"sort"
	"strings"
)

// DiscoveryOption is an evidence-linked candidate alternative, not a decision.
// No options are invented by the deterministic selector.
type DiscoveryOption struct {
	Label    string              `json:"label"`
	Tradeoff string              `json:"tradeoff"`
	Sources  []CoverageReference `json:"sources"`
}

// DiscoveryQuestions keeps settled conclusions available without re-asking
// them. Empty Questions means no unresolved dimension, not permission to approve.
type DiscoveryQuestions struct {
	Coverage  CoverageResult
	Questions []CoverageItem
	Settled   []CoverageItem
}

const discoveryQuestionInstructions = `Assess all discovery dimensions, but do not ask a full questionnaire.
For each open question, cite known evidence, name the missing decision and explain its impact.
Use impact_kind: irreversible, correctness, scope, goal or other. Do not exaggerate impact.
Ask at most three distinct questions in a turn. Put conflicts first, then high-impact gaps.
Keep resolved conclusions as context. Do not ask them again without changed evidence or a conflict.
For a conflict, show both source excerpts and ask the human to decide. Never choose a side silently.
Only include options when the evidence supports real alternatives. Give each option a label, tradeoff and source references.
If nothing important is missing, do not invent a question. An empty question list does not mean approval.`

// SelectDiscoveryQuestions validates the raw response against current sources
// before ranking. It does not trust a mutable previously validated assessment.
// Source freshness and human confirmation use the same boundary as ParseCoverage.
func SelectDiscoveryQuestions(raw string, snapshot ConversationContext, confirmed []CoverageReference) (DiscoveryQuestions, error) {
	coverage, err := ParseCoverage(raw, snapshot, confirmed)
	result := DiscoveryQuestions{Coverage: coverage, Questions: []CoverageItem{}, Settled: []CoverageItem{}}
	if err != nil {
		return result, err
	}
	var pending []CoverageItem
	for _, item := range coverage.Assessment.Items {
		if item.Status == "resolved" || item.Status == "not_applicable" {
			result.Settled = append(result.Settled, item)
		} else {
			pending = append(pending, item)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		left, right := pending[i], pending[j]
		if (left.Status == "conflict") != (right.Status == "conflict") {
			return left.Status == "conflict"
		}
		if questionPriority(left) != questionPriority(right) {
			return questionPriority(left) < questionPriority(right)
		}
		return left.Dimension < right.Dimension
	})
	seen := make(map[string]bool)
	for _, item := range pending {
		key := strings.ToLower(strings.Join(strings.Fields(item.Question), " "))
		if seen[key] {
			continue
		}
		seen[key] = true
		result.Questions = append(result.Questions, item)
		if len(result.Questions) == 3 {
			break
		}
	}
	return result, nil
}

// questionPriority is a stable policy, not semantic scoring of free text.
// Older candidates without impact_kind retain useful dimension-based ordering.
func questionPriority(item CoverageItem) int {
	kind := item.ImpactKind
	if kind == "" {
		switch item.Dimension {
		case "rules", "exceptions", "data", "permissions", "acceptance": kind = "correctness"
		case "scope", "dependencies": kind = "scope"
		case "goals", "problem": kind = "goal"
		default: kind = "other"
		}
	}
	switch kind {
	case "irreversible": return 0
	case "correctness": return 1
	case "scope": return 2
	case "goal": return 3
	default: return 4
	}
}

func (snapshot ConversationContext) validateQuestionDetails(item CoverageItem) error {
	switch item.ImpactKind {
	case "", "irreversible", "correctness", "scope", "goal", "other":
	default: return errors.New("unsupported impact_kind")
	}
	if len(item.Options) == 0 { return nil }
	if len(item.Options) < 2 || len(item.Options) > 3 || (item.Status != "needs_input" && item.Status != "conflict") {
		return errors.New("only open questions may have two or three alternatives")
	}
	seen := make(map[string]bool)
	for _, option := range item.Options {
		label := strings.ToLower(strings.Join(strings.Fields(option.Label), " "))
		if label == "" || seen[label] || strings.TrimSpace(option.Tradeoff) == "" || len(option.Sources) == 0 || len(option.Sources) > 8 {
			return errors.New("options need unique labels, tradeoffs and one to eight references")
		}
		seen[label] = true
		for _, ref := range option.Sources {
			if !snapshot.validCoverageReference(ref) { return errors.New("option evidence is missing or stale") }
		}
	}
	return nil
}
