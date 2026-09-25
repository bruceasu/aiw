package task

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aiw/internal/requirement"
	"aiw/internal/session"
)

// approvalCheckpoint binds advice to the exact displayed action and evidence.
// It is checked again before chat approval; direct approve CLI is unchanged.
type approvalCheckpoint struct {
	Action string
	HistoryDigest string
	FactsDigest string
}

func printRequirementReadiness(report requirement.ReadinessReport) {
	fmt.Printf("Approval advice ready: %t (not approval). Draft capture remains available.\n", report.Ready)
	for _, ref := range report.Confirmed { fmt.Printf("Confirmed: %s: %s\n", ref.Source, ref.Quote) }
	for _, gap := range report.BusinessBlockers { fmt.Printf("Business blocker: %s\n", gap) }
	for _, gap := range report.PlanGaps { fmt.Printf("Plan gap: %s\n", gap) }
	for _, item := range report.UnconfirmedDeferrals { fmt.Printf("Needs human postponement confirmation: %s; reason: %s\n", item.Question, item.Reason) }
	for _, item := range report.Deferred { fmt.Printf("Human-confirmed design postponement: %s; reason: %s\n", item.Question, item.Reason) }
}

// currentApprovalCheckpoint always recomputes readiness from raw output after
// reopening sources. A saved Ready boolean is not trusted.
func currentApprovalCheckpoint(store *session.Store, sessionID, actionText, id string) (approvalCheckpoint, error) {
	history, err := readRequirementHistory(store, sessionID)
	if err != nil { return approvalCheckpoint{}, err }
	content, err := store.ReadArtifact(sessionID, "confirmed-requirement-facts.json")
	if err != nil { return approvalCheckpoint{}, err }
	var facts requirement.ConfirmedFacts
	if err := json.Unmarshal([]byte(content), &facts); err != nil { return approvalCheckpoint{}, err }
	if candidate, notice := requirement.RestoreConversationCandidate(history, id, facts); candidate == nil {
		return approvalCheckpoint{}, errors.New(notice)
	}
	var saved requirement.ConversationEvidence
	if err := json.Unmarshal([]byte(history), &saved); err != nil { return approvalCheckpoint{}, err }
	report := requirement.AssessReadiness(saved.Turn.Discovery.Coverage.Raw, saved.Turn.Context, facts)
	if !report.Ready { return approvalCheckpoint{}, errors.New("current coverage and Plan are not ready for approval advice") }
	return approvalCheckpoint{Action: actionText, HistoryDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(history))), FactsDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(content)))}, nil
}

func confirmRequirementApproval(store *session.Store, sessionID, pending string, action requirementPendingAction) error {
	if strings.ToUpper(action.Decision) == "APPROVED" {
		content, err := store.ReadArtifact(sessionID, "approval-checkpoint.json")
		if err != nil { return err }
		var displayed approvalCheckpoint
		if err := json.Unmarshal([]byte(content), &displayed); err != nil { return err }
		current, err := currentApprovalCheckpoint(store, sessionID, pending, action.RequirementID)
		if err != nil { return err }
		if current != displayed { return errors.New("approval evidence changed after display; reassess and display again") }
	}
	_, err := requirement.Approve(action.RequirementID, action.Decision, action.By, action.Reason)
	return err
}
