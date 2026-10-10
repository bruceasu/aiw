package issue

import (
	"errors"
	"fmt"
	"strings"
)

// The baseline is built in so a new project needs no installed domain skill.
// Its digest identifies the exact version included in this turn.
const discoveryBaseline = `Explore roles, the problem, current workflow, goals, scope, rules, exceptions, data, permissions, external dependencies and acceptance examples.
Use known evidence. Ask about the most important missing decision. Do not repeat settled questions without new evidence or a conflict.
Keep facts, assumptions and open questions separate. Unknown domains use this baseline. Missing information is not approval. Keep human confirmation for every durable action.`

// ConversationMethodSuggestion is an unconfirmed model recommendation. Evidence
// must quote current input or a loaded Requirement source, not model history.
// Gap is intake, value, metric, feasibility or synthesis. Domain is finance,
// generic or unknown. TermsUnclear adds only the domain-modeling supplement.
type ConversationMethodSuggestion struct {
	Domain       string `json:"domain"`
	Gap          string `json:"gap"`
	Reason       string `json:"reason"`
	Source       string `json:"source"`
	SourceDigest string `json:"source_digest"`
	Quote        string `json:"quote"`
	TermsUnclear bool   `json:"terms_unclear"`
}

// ConversationMethodSelection records a routing decision, not a confirmed
// business fact. Degraded means only the generic baseline was selected.
type ConversationMethodSelection struct {
	Suggestion *ConversationMethodSuggestion `json:"suggestion,omitempty"`
	Primary    string                        `json:"primary"`
	Supplement string                        `json:"supplement,omitempty"`
	Status     string                        `json:"status"`
	Reason     string                        `json:"reason"`
}

// methodEvidence checks existence, freshness and the literal quote. Whether
// that quote supports the proposed domain remains a model/human judgement.
func (snapshot ConversationContext) methodEvidence(s ConversationMethodSuggestion) bool {
	content := ""
	if s.Source == "user-input" {
		content = snapshot.UserInput
	} else {
		for _, source := range snapshot.Sources {
			if source.Path == s.Source && source.Status == "loaded" {
				content = source.Content
				break
			}
		}
	}
	return content != "" && strings.TrimSpace(s.Quote) != "" &&
		s.SourceDigest == digest([]byte(content)) && strings.Contains(content, s.Quote)
}

// loadMethods runs before optional content so background cannot crowd out a
// selected method. Only fixed project-local installed paths can be read; no
// home directory search, arbitrary model-supplied path or recursive expansion.
func (snapshot *ConversationContext) loadMethods(reader *contextReader, suggestion *ConversationMethodSuggestion) error {
	selection := &ConversationMethodSelection{Primary: "generic-discovery", Status: "loaded", Reason: "Use the generic discovery baseline"}
	snapshot.MethodSelection = selection
	baseline := ConversationSource{Kind: "method", Path: "builtin:generic-discovery", Purpose: "Generic requirement discovery baseline"}
	if len(discoveryBaseline) > reader.remaining {
		baseline.Status, baseline.Reason = "omitted", errContextBudget.Error()
		snapshot.Methods = append(snapshot.Methods, baseline)
		selection.Status = "blocked"
		return fmt.Errorf("load generic discovery: %w", errContextBudget)
	}
	reader.remaining -= len(discoveryBaseline)
	baseline.Status, baseline.Content, baseline.Digest = "loaded", discoveryBaseline, digest([]byte(discoveryBaseline))
	snapshot.Methods = append(snapshot.Methods, baseline)
	if suggestion == nil {
		return nil
	}
	// Bound diagnostic data too; an invalid model response must not inflate the
	// prompt merely because its method recommendation was rejected.
	size := len(suggestion.Domain) + len(suggestion.Gap) + len(suggestion.Reason) +
		len(suggestion.Source) + len(suggestion.SourceDigest) + len(suggestion.Quote)
	if size > 4096 || size > reader.remaining {
		selection.Status, selection.Reason = "degraded", "Method suggestion omitted: recommendation exceeds budget; use generic discovery"
		return nil
	}
	copy := *suggestion
	selection.Suggestion = &copy
	reader.remaining -= size
	validDomain := copy.Domain == "finance" || copy.Domain == "generic" || copy.Domain == "unknown"
	methods := map[string]string{
		"intake": "finance-requirement-intake",
		"value": "finance-value-assessment",
		"metric": "finance-metric-brief",
		"feasibility": "finance-engineering-options",
		"synthesis": "finance-requirement-synthesis",
	}
	method, validGap := methods[copy.Gap]
	if !validDomain || !validGap || strings.TrimSpace(copy.Reason) == "" || !snapshot.methodEvidence(copy) {
		selection.Status, selection.Reason = "degraded", "Invalid or stale method evidence; use generic discovery and clarify the domain or gap"
		return nil
	}
	selection.Reason = "Evidence-linked recommendation; domain and gap are not confirmed facts"
	var selected []string
	if copy.Domain == "finance" {
		selection.Primary = method
		selected = append(selected, method)
	}
	if copy.TermsUnclear {
		selection.Supplement = "domain-modeling"
		selected = append(selected, "domain-modeling")
	}
	var failures []error
	for _, id := range selected {
		source := ConversationSource{
			Kind: "method", Path: ".agents/skills/" + id + "/SKILL.md",
			Purpose: "Selected discussion method in Requirement mode: " + id,
		}
		err := source.load(reader, false)
		if err == nil && strings.TrimSpace(source.Content) == "" {
			source.Status, source.Reason = "unavailable", "Selected method is empty"
			err = errors.New(source.Reason)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("method %s: %w", id, err))
		}
		snapshot.Methods = append(snapshot.Methods, source)
	}
	if len(failures) != 0 {
		selection.Status, selection.Reason = "blocked", "Selected method could not be loaded; restore it or narrow the scope before continuing"
	}
	return errors.Join(failures...)
}
