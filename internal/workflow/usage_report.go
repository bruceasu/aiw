package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// UsageReportQuery bounds a read-only view of the Task's normalized usage
// records. An absent time bound is open ended.
type UsageReportQuery struct {
	WorkItemID WorkItemID
	AttemptID  AttemptID
	Provider   string
	Profile    string
	From       *time.Time
	To         *time.Time
}

// UsageReportCall intentionally excludes Provider evidence and raw usage
// payloads. It is the safe per-call surface used by the usage report.
type UsageReportCall struct {
	ID                 string    `json:"id"`
	WorkItemID         WorkItemID `json:"work_item_id"`
	AttemptID          AttemptID `json:"attempt_id"`
	SessionID          string    `json:"session_id"`
	SessionTurn        int       `json:"session_turn"`
	Provider           string    `json:"provider"`
	Model              string    `json:"model"`
	Profile            string    `json:"profile"`
	DifficultyLevel    int       `json:"difficulty_level"`
	ReasoningIntensity string    `json:"reasoning_intensity,omitempty"`
	Outcome            string    `json:"outcome"`
	RecordedAt         time.Time `json:"recorded_at"`
}

// UsageReportTotals contains only Provider-reported known values. Monetary
// totals are exact rational strings grouped by currency.
type UsageReportTotals struct {
	Calls             int64             `json:"calls"`
	UsageUnknown      int64             `json:"usage_unknown"`
	KnownInputTokens  int64             `json:"known_input_tokens"`
	KnownOutputTokens int64             `json:"known_output_tokens"`
	KnownTotalTokens  int64             `json:"known_total_tokens"`
	KnownCostByCurrency map[string]string `json:"known_cost_by_currency"`
}

// UsageDifficultyChange records the previous and selected configuration for
// a call whose difficulty or Profile selection was adjusted.
type UsageDifficultyChange struct {
	CallID                       string    `json:"call_id"`
	RecordedAt                   time.Time `json:"recorded_at"`
	RequestedLevel               int       `json:"requested_level"`
	PreviousProfile              string    `json:"previous_profile,omitempty"`
	PreviousProvider             string    `json:"previous_provider,omitempty"`
	PreviousModel                string    `json:"previous_model,omitempty"`
	PreviousLevel                int       `json:"previous_level,omitempty"`
	PreviousReasoningIntensity   string    `json:"previous_reasoning_intensity,omitempty"`
	SelectedProfile              string    `json:"selected_profile"`
	SelectedProvider             string    `json:"selected_provider"`
	SelectedModel                string    `json:"selected_model"`
	SelectedLevel                int       `json:"selected_level"`
	SelectedReasoningIntensity   string    `json:"selected_reasoning_intensity,omitempty"`
	Reason                       string    `json:"reason"`
}

// UsageReport is a bounded read-only projection. Audit lists do not expose
// Provider raw evidence.
type UsageReport struct {
	TaskID           TaskID                      `json:"task_id"`
	Totals           UsageReportTotals           `json:"totals"`
	Outcomes         map[string]int64             `json:"outcomes"`
	BudgetApprovals  []UsageBudgetApproval        `json:"budget_approvals"`
	BudgetTerminations []UsageBudgetTermination  `json:"budget_terminations"`
	DifficultyChanges []UsageDifficultyChange    `json:"difficulty_changes"`
	Calls            []UsageReportCall            `json:"calls"`
}

// GetUsageReport reads normalized usage events through the Workflow Store and
// applies every supplied filter without exposing raw Provider evidence.
func GetUsageReport(ctx context.Context, store *Store, taskID TaskID, query UsageReportQuery) (UsageReport, error) {
	if store == nil {
		return UsageReport{}, errors.New("usage report requires a Store")
	}
	return store.GetUsageReport(ctx, taskID, query)
}

// GetUsageReport is the Workflow facade for read-only usage statistics.
func (s *Store) GetUsageReport(ctx context.Context, taskID TaskID, query UsageReportQuery) (UsageReport, error) {
	if ctx == nil {
		return UsageReport{}, errors.New("usage report requires a context")
	}
	if err := ctx.Err(); err != nil {
		return UsageReport{}, err
	}
	if s == nil || taskID == "" {
		return UsageReport{}, errors.New("usage report requires a Store and Task ID")
	}
	if query.From != nil && query.To != nil && query.From.After(*query.To) {
		return UsageReport{}, errors.New("usage report start time is after end time")
	}
	state, err := s.Load(taskID)
	if err != nil {
		return UsageReport{}, fmt.Errorf("load Task usage state: %w", err)
	}
	report := UsageReport{
		TaskID: taskID,
		Totals: UsageReportTotals{KnownCostByCurrency: map[string]string{}},
		Outcomes: map[string]int64{},
		BudgetApprovals: []UsageBudgetApproval{},
		BudgetTerminations: []UsageBudgetTermination{},
		DifficultyChanges: []UsageDifficultyChange{},
		Calls: []UsageReportCall{},
	}
	if state.Protocol == nil || state.Protocol.Usage == nil {
		return report, nil
	}
	ledger := state.Protocol.Usage
	projection := TaskUsageProjection{KnownCostByCurrency: map[string]string{}}
	for _, raw := range ledger.Records {
		if err := ctx.Err(); err != nil {
			return UsageReport{}, err
		}
		var event UsageEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return UsageReport{}, fmt.Errorf("decode Task usage record: %w", err)
		}
		if (query.WorkItemID != "" && query.WorkItemID != event.WorkItemID) ||
			(query.AttemptID != "" && query.AttemptID != event.AttemptID) ||
			(query.Provider != "" && query.Provider != event.Provider) ||
			(query.Profile != "" && query.Profile != event.Profile) {
			continue
		}
		recordedAt, err := time.Parse(time.RFC3339Nano, event.RecordedAt)
		if err != nil {
			return UsageReport{}, fmt.Errorf("usage record %q has invalid recorded_at: %w", event.ID, err)
		}
		if (query.From != nil && recordedAt.Before(*query.From)) || (query.To != nil && recordedAt.After(*query.To)) {
			continue
		}
		if err := addUsageProjection(&projection, event.Usage); err != nil {
			return UsageReport{}, fmt.Errorf("aggregate usage record %q: %w", event.ID, err)
		}
		report.Calls = append(report.Calls, UsageReportCall{
			ID: event.ID, WorkItemID: event.WorkItemID, AttemptID: event.AttemptID,
			SessionID: event.SessionID, SessionTurn: event.SessionTurn,
			Provider: event.Provider, Model: event.Model, Profile: event.Profile,
			DifficultyLevel: event.DifficultyLevel, ReasoningIntensity: event.ReasoningIntensity,
			Outcome: event.Outcome, RecordedAt: recordedAt,
		})
		report.Outcomes[event.Outcome]++
		if event.AdjustmentReason != "" {
			report.DifficultyChanges = append(report.DifficultyChanges, UsageDifficultyChange{
				CallID: event.ID, RecordedAt: recordedAt, RequestedLevel: event.RequestedLevel,
				PreviousProfile: event.PreviousProfile, PreviousProvider: event.PreviousProvider,
				PreviousModel: event.PreviousModel, PreviousLevel: event.PreviousLevel,
				PreviousReasoningIntensity: event.PreviousReasoningIntensity,
				SelectedProfile: event.Profile, SelectedProvider: event.Provider, SelectedModel: event.Model,
				SelectedLevel: event.DifficultyLevel, SelectedReasoningIntensity: event.ReasoningIntensity,
				Reason: event.AdjustmentReason,
			})
		}
	}
	report.Totals = UsageReportTotals{
		Calls: projection.Calls, UsageUnknown: projection.UsageUnknown,
		KnownInputTokens: projection.KnownInputTokens, KnownOutputTokens: projection.KnownOutputTokens,
		KnownTotalTokens: projection.KnownTotalTokens,
		KnownCostByCurrency: projection.KnownCostByCurrency,
	}
	for _, approval := range ledger.BudgetApprovals {
		at, err := time.Parse(time.RFC3339Nano, approval.At)
		if err != nil {
			return UsageReport{}, fmt.Errorf("budget approval has invalid timestamp: %w", err)
		}
		if reportTimeMatches(at, query.From, query.To) {
			report.BudgetApprovals = append(report.BudgetApprovals, approval)
		}
	}
	for _, termination := range ledger.BudgetTerminations {
		at, err := time.Parse(time.RFC3339Nano, termination.At)
		if err != nil {
			return UsageReport{}, fmt.Errorf("budget termination has invalid timestamp: %w", err)
		}
		if reportTimeMatches(at, query.From, query.To) {
			report.BudgetTerminations = append(report.BudgetTerminations, termination)
		}
	}
	return report, nil
}

func reportTimeMatches(value time.Time, from, to *time.Time) bool {
	return (from == nil || !value.Before(*from)) && (to == nil || !value.After(*to))
}
