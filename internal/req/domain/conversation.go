package requirement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ConversationModel adapts Session without making the domain own a provider.
type ConversationModel func(context.Context, string, string) (string, error)

// ConfirmedFacts records explicit fragment confirmation, not whole-document
// approval. The command adapter writes it only after a human checkpoint.
type ConfirmedFacts struct {
	RequirementID string `json:"requirement_id"`
	Revision int `json:"revision"`
	References []CoverageReference `json:"references"`
}

// ConversationTurn retains diagnostics even when no usable assessment exists.
type ConversationTurn struct {
	Readiness ReadinessReport
	MethodContext ConversationContext
	Context ConversationContext
	Discovery DiscoveryQuestions
	Phase string
	MethodOutput string
}

// RunConversationTurn performs two bounded model calls: method recommendation,
// then assessment with the actual loaded method. No retries or domain writes.
func RunConversationTurn(ctx context.Context, id, input string, candidate *ConversationCandidate, facts ConfirmedFacts, model ConversationModel) (ConversationTurn, error) {
	turn := ConversationTurn{Phase: "discovery"}
	snapshot, err := LoadConversationContext(id, input, candidate)
	turn.MethodContext = snapshot
	turn.Context = snapshot
	if err != nil { return turn, err }
	prompt, err := snapshot.Prompt()
	if err != nil { return turn, err }
	raw, err := model(ctx, "method-selection", prompt + "\n" + methodOutputInstructions + "\nuser-input digest: " + digest([]byte(input)))
	turn.MethodOutput = raw
	if err != nil { return turn, err }
	var method ConversationMethodSuggestion
	if err := decodeConversationJSON(raw, &method); err != nil {
		// An invalid recommendation explicitly degrades through the loader.
		method = ConversationMethodSuggestion{Domain: "invalid"}
	}
	snapshot, err = LoadConversationContextWithOptions(id, input, candidate, ConversationContextOptions{Method: &method})
	turn.Context = snapshot
	if err != nil { return turn, err }
	confirmed := facts.Current(snapshot)
	prompt, err = snapshot.Prompt()
	if err != nil { return turn, err }
	evidence, err := json.Marshal(confirmed)
	if err != nil { return turn, err }
	if len(evidence) > snapshot.BudgetBytes-snapshot.UsedBytes { return turn, errors.New("confirmed fragments exceed remaining context budget") }
	raw, err = model(ctx, "coverage-assessment", prompt + "\n" + coverageOutputInstructions +
		"\nuser-input digest: " + digest([]byte(input)) + "\nConfirmed fragments: " + string(evidence))
	if err != nil {
		turn.Discovery.Coverage = CoverageResult{Raw: raw, Diagnostics: []string{err.Error()}}
		return turn, err
	}
	turn.Discovery, err = SelectDiscoveryQuestions(raw, snapshot, confirmed)
	if err != nil { return turn, err }
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
		if item.Status == "conflict" { turn.Phase = "deep-discovery"; break }
		if item.Status == "needs_input" { turn.Phase = "discovery" }
	}
	turn.Readiness = AssessReadiness(raw, snapshot, facts)
	return turn, nil
}

// Current drops stale confirmation instead of silently moving it to new text.
func (facts ConfirmedFacts) Current(snapshot ConversationContext) []CoverageReference {
	var refs []CoverageReference
	if snapshot.Requirement == nil || facts.RequirementID != snapshot.Requirement.ID || facts.Revision != snapshot.Requirement.Revision { return refs }
	for _, ref := range facts.References {
		if ref.Source != "candidate" && ref.Source != "user-input" && snapshot.validCoverageReference(ref) { refs = append(refs, ref) }
	}
	return refs
}

// CaptureFactReferences validates the exact draft bytes before a checkpoint.
// The returned path is the future canonical artifact path, not the draft path.
func CaptureFactReferences(id, kind, source string, quotes []string) ([]CoverageReference, string, error) {
	filename, ok := artifactFiles[kind]
	if !ok || !ValidID(id) || id == "." || id == ".." { return nil, "", errors.New("invalid capture target") }
	reader, err := newContextReader(ConversationContextOptions{})
	if err != nil { return nil, "", err }
	defer reader.root.Close()
	content, err := reader.read(source)
	if err != nil { return nil, "", err }
	if len(quotes) > 32 { return nil, "", errors.New("too many confirmation fragments") }
	var refs []CoverageReference
	seen := make(map[string]bool)
	for _, quote := range quotes {
		if strings.TrimSpace(quote) == "" || !strings.Contains(string(content), quote) || seen[quote] { return nil, "", errors.New("confirmation fragment is missing or duplicated") }
		seen[quote] = true
		refs = append(refs, CoverageReference{Source: Root + "/" + id + "/" + filename, Digest: digest(content), Quote: quote})
	}
	return refs, digest(content), nil
}

func decodeConversationJSON(raw string, target any) error {
	if len(raw) > defaultContextBytes { return errors.New("model output exceeds budget") }
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil { return err }
	if err := decoder.Decode(new(any)); err != io.EOF { return fmt.Errorf("expected one JSON object") }
	return nil
}

const methodOutputInstructions = `This call only selects a discussion method. Do not use tools or prepare actions.
Return one JSON object with domain (generic/unknown/finance), gap (intake/value/metric/feasibility/synthesis), reason, source, source_digest, quote and terms_unclear.
Cite user-input or a loaded Requirement source. Use its exact digest and quote. No Markdown.`

const coverageOutputInstructions = `Return only a CoverageAssessment JSON object. No Markdown or prose outside JSON.
An Issue may be a bug, feature, or modification. Do not turn routine strategy choices into user questions when evidence and stated preferences select a clear option; put the choice and rationale in conclusion and next_step.
Fields: version:1, requirement_id (empty for new), revision (0 for new), items, optional plan_review.
Include each dimension once: roles, problem, current_workflow, goals, scope, rules, exceptions, data, permissions, dependencies, acceptance.
Each item has dimension, status, conclusion, sources, question, impact, next_step, reason, optional impact_kind and options.
Each source has source, digest and quote copied from current input. Status is resolved, needs_input, conflict or not_applicable.
Use resolved only with a supplied confirmed fragment. A new answer is not confirmed. Use needs_input to propose confirming it.
Conflict needs both source excerpts. Not_applicable needs a reason. Questions and explanations should use the human's language.
You may prepare a requested draft and a pending action via aiw req chat prepare, but never execute a durable action.
For capture you may append --facts-json with a JSON array of exact draft excerpts to explicitly confirm. Omit it for draft-only capture.
Prepare drafts as project-relative UTF-8 files outside .ai and credential directories, within 64 KiB. Never replace a captured artifact directly.
Never prepare approval based only on a file or an empty question list. Do not write confirmation records yourself. The host displays checkpoints.
When a captured requirement-plan is loaded, include plan_review with sections and deferred_design.
Each section has section, explanation, and sources (source/digest/quote). Include facts, assumptions, goals, scope, non_goals, rules, acceptance, sources, remaining_decisions.
Quote relevant Plan body text, not headings. Explain missing content with empty sources. Explicitly state when assumptions or remaining decisions are absent.
Facts, goals, scope, rules and acceptance Plan statements need human-confirmed fragments before approval advice.
Each deferred_design item has question, reason, and decision (source/digest/quote). The Plan decision quote must include both question and reason and explicitly propose postponement to engineering.
Only a matching supplied confirmed fragment accepts postponement. Use capture --facts-json to ask the human to confirm that exact decision. Never claim your own confirmation.
Keep unresolved business rules and conflicts in coverage as needs_input or conflict. Design postponements never waive business gaps.
List all remaining design decisions; do not hide them to make the Plan ready. Approval advice is still subject to the host's report and human review.`
