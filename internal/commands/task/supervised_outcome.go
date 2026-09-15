package task

import (
	"encoding/json"
	"strings"

	"aiw/internal/workflow"
)

type agentSupervisedOutcome struct {
	Outcome         workflow.SupervisedOutcomeKind  `json:"outcome"`
	BlockedCategory workflow.BlockedOutcomeCategory `json:"blocked_category,omitempty"`
	Detail          string                          `json:"detail,omitempty"`
}

func parseSupervisedOutcome(output, evidenceReference string) workflow.SupervisedOutcome {
	var reported agentSupervisedOutcome
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &reported); err != nil {
		return workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeNoProgress, Detail: "Agent did not return a valid structured outcome", EvidenceReference: evidenceReference}
	}
	outcome := workflow.SupervisedOutcome{Kind: reported.Outcome, BlockedCategory: reported.BlockedCategory, Detail: strings.TrimSpace(reported.Detail), EvidenceReference: evidenceReference}
	if err := workflow.ValidateSupervisedOutcome(outcome); err != nil {
		return workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeNoProgress, Detail: "Agent returned an invalid structured outcome: " + err.Error(), EvidenceReference: evidenceReference}
	}
	return outcome
}
