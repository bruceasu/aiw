package issue

import (
	"encoding/json"
	"reflect"
)

// ConversationEvidence is Session evidence, never a Requirement or approval.
// Running records deliberately supersede earlier successful candidates.
type ConversationEvidence struct {
	Version int `json:"version"`
	RequirementID string `json:"requirement_id"`
	State string `json:"state"`
	Turns []int `json:"turns"`
	Turn ConversationTurn `json:"turn"`
	Confirmed ConfirmedFacts `json:"confirmed"`
	Diagnostic string `json:"diagnostic,omitempty"`
}

// RestoreConversationCandidate reopens sources before accepting historical
// discussion. It never restores a phase, pending action, or confirmed facts.
func RestoreConversationCandidate(raw, id string, facts ConfirmedFacts) (*ConversationCandidate, string) {
	if raw == "" { return nil, "No structured history; rebuild from Requirement artifacts. Unsaved discussion cannot be recovered." }
	var saved ConversationEvidence
	if err := json.Unmarshal([]byte(raw), &saved); err != nil || saved.Version != 1 {
		return nil, "Unsupported or invalid history; rebuild from Requirement artifacts."
	}
	if saved.RequirementID != id || saved.State != "completed" || len(saved.Turns) != 2 || saved.Turns[0] <= 0 || saved.Turns[1] != saved.Turns[0]+1 {
		return nil, "Latest discussion is incomplete or belongs to another Requirement; reassess without an older candidate."
	}
	previous := saved.Turn.Context
	var method *ConversationMethodSuggestion
	if previous.MethodSelection != nil { method = previous.MethodSelection.Suggestion }
	current, err := LoadConversationContextWithOptions(id, previous.UserInput, previous.Candidate, ConversationContextOptions{Method: method})
	if err != nil || !reflect.DeepEqual(previous.Requirement, current.Requirement) ||
		!reflect.DeepEqual(previous.Sources, current.Sources) || !reflect.DeepEqual(previous.Methods, current.Methods) {
		return nil, "Historical candidate is stale: Requirement or sources changed; reassess."
	}
	// Validate historical quotations against historical input, but trust only
	// confirmations still supplied by the host at the current revision.
	if _, err := ParseCoverage(saved.Turn.Discovery.Coverage.Raw, current, facts.Current(current)); err != nil {
		return nil, "Historical candidate no longer validates; reassess."
	}
	candidate := &ConversationCandidate{RequirementID: id, Content: saved.Turn.Discovery.Coverage.Raw}
	if current.Requirement != nil { candidate.Revision = current.Requirement.Revision }
	return candidate, "Recovered unconfirmed discussion only; a fresh assessment is required."
}
