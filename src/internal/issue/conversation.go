package issue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ConversationModel adapts Session without making the domain own a provider.
type ConversationModel func(context.Context, string, string) (string, error)

// ConfirmedFacts records explicit fragment confirmation, not whole-document
// approval. The command adapter writes it only after a human checkpoint.
type ConfirmedFacts struct {
	RequirementID string              `json:"requirement_id"`
	Revision      int                 `json:"revision"`
	References    []CoverageReference `json:"references"`
}

// ConversationTurn retains diagnostics even when no usable assessment exists.
type ConversationTurn struct {
	Readiness     ReadinessReport
	MethodContext ConversationContext
	Context       ConversationContext
	Discovery     DiscoveryQuestions
	Phase         string
	MethodOutput  string
}

// RunConversationTurn performs two bounded model calls: method recommendation,
// then assessment with the actual loaded method. No retries or domain writes.
func RunConversationTurn(ctx context.Context, id, input string, candidate *ConversationCandidate, facts ConfirmedFacts, model ConversationModel) (ConversationTurn, error) {
	turn := ConversationTurn{Phase: "discovery"}
	snapshot, err := LoadConversationContext(id, input, candidate)
	turn.MethodContext = snapshot
	turn.Context = snapshot
	if err != nil {
		return turn, err
	}
	prompt, err := snapshot.Prompt()
	if err != nil {
		return turn, err
	}
	raw, err := model(ctx, "method-selection", prompt+"\n"+methodOutputInstructions+"\nuser-input digest: "+digest([]byte(input)))
	turn.MethodOutput = raw
	if err != nil {
		return turn, err
	}
	var method ConversationMethodSuggestion
	if err := decodeConversationJSON(raw, &method); err != nil {
		// An invalid recommendation explicitly degrades through the loader.
		method = ConversationMethodSuggestion{Domain: "invalid"}
	}
	snapshot, err = LoadConversationContextWithOptions(id, input, candidate, ConversationContextOptions{Method: &method})
	turn.Context = snapshot
	if err != nil {
		return turn, err
	}
	confirmed := facts.Current(snapshot)
	prompt, err = snapshot.Prompt()
	if err != nil {
		return turn, err
	}
	evidence, err := json.Marshal(confirmed)
	if err != nil {
		return turn, err
	}
	if len(evidence) > snapshot.BudgetBytes-snapshot.UsedBytes {
		return turn, errors.New("confirmed fragments exceed remaining context budget")
	}
	raw, err = model(ctx, "coverage-assessment", prompt+"\n"+coverageOutputInstructions+
		"\nuser-input digest: "+digest([]byte(input))+"\nConfirmed fragments: "+string(evidence))
	if err != nil {
		turn.Discovery.Coverage = CoverageResult{Raw: raw, Diagnostics: []string{err.Error()}}
		return turn, err
	}
	turn.Discovery, err = SelectDiscoveryQuestions(raw, snapshot, confirmed)
	if err != nil {
		return turn, err
	}
	// A tool-capable provider must not leave us accepting an assessment against
	// artifacts that changed during its response.
	latest, err := LoadConversationContextWithOptions(id, input, candidate, ConversationContextOptions{Method: &method})
	if err != nil {
		turn.Discovery = DiscoveryQuestions{Coverage: CoverageResult{Raw: raw, Diagnostics: []string{err.Error()}}}
		return turn, err
	}
	before, _ := json.Marshal(snapshot)
	after, _ := json.Marshal(latest)
	if string(before) != string(after) {
		turn.Discovery = DiscoveryQuestions{Coverage: CoverageResult{Raw: raw, Diagnostics: []string{"context changed during assessment"}}}
		return turn, errors.New("Requirement context changed during assessment; reassess")
	}
	turn.Phase = "synthesis"
	for _, item := range turn.Discovery.Coverage.Assessment.Items {
		if item.Status == "conflict" {
			turn.Phase = "deep-discovery"
			break
		}
		if item.Status == "needs_input" {
			turn.Phase = "discovery"
		}
	}
	turn.Readiness = AssessReadiness(raw, snapshot, facts)
	return turn, nil
}

// Current drops stale confirmation instead of silently moving it to new text.
func (facts ConfirmedFacts) Current(snapshot ConversationContext) []CoverageReference {
	var refs []CoverageReference
	if snapshot.Requirement == nil || facts.RequirementID != snapshot.Requirement.ID || facts.Revision != snapshot.Requirement.Revision {
		return refs
	}
	for _, ref := range facts.References {
		if ref.Source != "candidate" && ref.Source != "user-input" && snapshot.validCoverageReference(ref) {
			refs = append(refs, ref)
		}
	}
	return refs
}

// CaptureFactReferences validates the exact draft bytes before a checkpoint.
// The returned path is the future canonical artifact path, not the draft path.
func CaptureFactReferences(id, kind, source string, quotes []string) ([]CoverageReference, string, error) {
	kind = canonicalArtifactKind(kind)
	_, ok := artifactFiles[kind]
	if !ok || !ValidID(id) || id == "." || id == ".." {
		return nil, "", errors.New("invalid capture target")
	}
	reader, err := newContextReader(ConversationContextOptions{})
	if err != nil {
		return nil, "", err
	}
	defer reader.root.Close()
	meta, dir, err := reader.readRequirement(id)
	if err != nil { return nil, "", err }
	if meta.Status == "ARCHIVED" || meta.Status == "CANCELLED" { return nil, "", fmt.Errorf("Issue is %s", meta.Status) }
	filename := artifactFilename(dir, kind)
	content, err := reader.readDraft(source)
	if err != nil {
		return nil, "", err
	}
	if len(quotes) > 32 {
		return nil, "", errors.New("too many confirmation fragments")
	}
	var refs []CoverageReference
	seen := make(map[string]bool)
	for _, quote := range quotes {
		if strings.TrimSpace(quote) == "" || !strings.Contains(string(content), quote) || seen[quote] {
			return nil, "", errors.New("confirmation fragment is missing or duplicated")
		}
		seen[quote] = true
		refs = append(refs, CoverageReference{Source: filepath.ToSlash(filepath.Join(dir, filename)), Digest: digest(content), Quote: quote})
	}
	return refs, digest(content), nil
}

func decodeConversationJSON(raw string, target any) error {
	if len(raw) > defaultContextBytes {
		return errors.New("model output exceeds budget")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected one JSON object")
	}
	return nil
}

const methodOutputInstructions = `Select a discussion method only. Do not use tools or prepare actions.

Return exactly one JSON object with these fields:
- domain: generic, unknown, or finance
- gap: intake, value, metric, feasibility, or synthesis
- reason
- source
- source_digest
- quote
- terms_unclear

Cite the user input or a loaded Requirement source. Copy its digest and relevant quote exactly. Do not invent source evidence. Return JSON only, with no Markdown.`

const coverageOutputInstructions = `Return only a CoverageAssessment JSON object. Do not add Markdown or text outside the JSON.

An Issue can be a bug, feature, or modification. Use the human's language for questions and explanations.

Output fields:
- version: 1
- requirement_id: empty for a new Issue
- revision: 0 for a new Issue
- items
- plan_review: include only when a captured requirement-plan is loaded

Include exactly one item for each dimension: roles, problem, current_workflow, goals, scope, rules, exceptions, data, permissions, dependencies, and acceptance.

Each item has dimension, status, conclusion, sources, question, impact, next_step, and reason. It may also have impact_kind and options. Status must be resolved, needs_input, conflict, or not_applicable.

For every source, provide source, digest, and an exact quote from the current input.
- Use resolved only when a supplied fragment is human-confirmed. A new answer is not yet confirmed; use needs_input to request confirmation.
- Use conflict only when you can quote both conflicting sources.
- Use not_applicable only with a reason.
- If evidence and stated preferences clearly select a routine strategy, put the choice and reason in conclusion and next_step. Do not ask the human to choose again.

You may write a requested draft and prepare a pending action with "aiw issue resume". Never execute a durable action or write a confirmation record. The host displays confirmation checkpoints.

Write drafts as project-relative UTF-8 files under ".ai/requirements/drafts/", within 64 KiB. This directory is allowed only as a capture source; it is not a general context source. Never directly replace a captured artifact. For capture, use "--facts-json" only to request confirmation of an array of exact draft excerpts; omit it for a draft-only capture. Never prepare approval from a file alone or merely because the question list is empty.

When a captured requirement-plan is loaded, include plan_review with sections and deferred_design.

Include these sections: facts, assumptions, goals, scope, non_goals, rules, acceptance, sources, and remaining_decisions. Each section has section, explanation, and sources. Quote relevant Plan body text, not headings. If content is missing, explain the gap and use an empty sources array. State explicitly when assumptions or remaining decisions are absent.

Before advising approval, require human-confirmed fragments for Plan statements about facts, goals, scope, rules, and acceptance. Keep every unresolved business rule or source conflict in coverage as needs_input or conflict.

Each deferred_design item has question, reason, and decision (source, digest, quote). The quoted Plan decision must include the question, the reason, and an explicit proposal to postpone it to engineering. Accept postponement only when a matching human-confirmed fragment is supplied. Otherwise, use capture with "--facts-json" to request confirmation of that exact decision. Never claim that you confirmed it yourself.

List every remaining design decision. Do not hide decisions to make the Plan appear ready. A design postponement does not resolve a business gap. Approval advice remains subject to the host's report and human review.`
