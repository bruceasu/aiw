package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const TaskUsageLedgerVersion = 1

// TaskUsageLedger is the Task-scoped accounting record carried by the existing
// Schema 10 execution protocol. Records retain the normalized usage-event JSON;
// the projection is a cached summary and never replaces its source records.
type TaskUsageLedger struct {
	Version    int                 `json:"version"`
	Records    []json.RawMessage   `json:"records"`
	Projection TaskUsageProjection `json:"projection"`
	Budget     *TaskUsageBudget    `json:"budget,omitempty"`
	PendingBudgetAuthorization *UsageBudgetAuthorization `json:"pending_budget_authorization,omitempty"`
	BudgetApprovals []UsageBudgetApproval `json:"budget_approvals,omitempty"`
	BudgetTerminations []UsageBudgetTermination `json:"budget_terminations,omitempty"`
}

// UsageEvent is the immutable accounting fact for one frozen managed Session
// turn. The identity is derived from Task, WorkItem, Attempt, Session, and turn
// so recovery can safely retry projection without counting the call twice.
type UsageEvent struct {
	Version              int             `json:"version"`
	ID                   string          `json:"id"`
	TaskID               TaskID          `json:"task_id"`
	WorkItemID           WorkItemID      `json:"work_item_id"`
	AttemptID            AttemptID       `json:"attempt_id"`
	SessionID            string          `json:"session_id"`
	SessionTurn          int             `json:"session_turn"`
	Provider             string          `json:"provider"`
	Model                string          `json:"model"`
	Profile              string          `json:"profile"`
	DifficultyLevel      int             `json:"difficulty_level"`
	ReasoningIntensity   string          `json:"reasoning_intensity,omitempty"`
	RequestedLevel       int             `json:"requested_level,omitempty"`
	AdjustmentReason     string          `json:"adjustment_reason,omitempty"`
	PreviousProfile      string          `json:"previous_profile,omitempty"`
	PreviousProvider     string          `json:"previous_provider,omitempty"`
	PreviousModel        string          `json:"previous_model,omitempty"`
	PreviousLevel        int             `json:"previous_level,omitempty"`
	PreviousReasoningIntensity string    `json:"previous_reasoning_intensity,omitempty"`
	ParameterDigest      string          `json:"parameter_digest,omitempty"`
	Outcome              string          `json:"outcome"`
	Usage                json.RawMessage `json:"usage"`
	RecordedAt           string          `json:"recorded_at"`
}

// TaskUsageProjection stores only Provider-reported known totals. Unknown and
// invalid field counts remain separate from those totals.
type TaskUsageProjection struct {
	Calls                 int64                        `json:"calls"`
	UsageUnknown          int64                        `json:"usage_unknown"`
	KnownInputTokens      int64                        `json:"known_input_tokens"`
	KnownOutputTokens     int64                        `json:"known_output_tokens"`
	KnownTotalTokens      int64                        `json:"known_total_tokens"`
	KnownCostByCurrency   map[string]string            `json:"known_cost_by_currency"`
	AvailabilityFields    UsageFieldCounts             `json:"availability_fields"`
	InputTokenFields      UsageFieldCounts             `json:"input_token_fields"`
	OutputTokenFields     UsageFieldCounts             `json:"output_token_fields"`
	TotalTokenFields      UsageFieldCounts             `json:"total_token_fields"`
	CostAmountFields      UsageFieldCounts             `json:"cost_amount_fields"`
	CostCurrencyFields    UsageFieldCounts             `json:"cost_currency_fields"`
}

// UsageFieldCounts counts calls by the field-level state in the Provider
// envelope. Values are counters, not usage amounts.
type UsageFieldCounts struct {
	Known   int64 `json:"known"`
	Unknown int64 `json:"unknown"`
	Invalid int64 `json:"invalid"`
}

type usageProjectionEnvelope struct {
	Availability string `json:"availability"`
	InputTokens usageProjectionField[int64] `json:"input_tokens"`
	OutputTokens usageProjectionField[int64] `json:"output_tokens"`
	TotalTokens usageProjectionField[int64] `json:"total_tokens"`
	CostAmount usageProjectionField[json.Number] `json:"cost_amount"`
	CostCurrency usageProjectionField[string] `json:"cost_currency"`
}

type usageProjectionField[T any] struct {
	State string `json:"state"`
	Value *T `json:"value"`
}

// addUsageProjection updates Task-wide totals from Provider-reported values.
// It deliberately sums each Token field independently and never derives a
// total from input/output components.
func addUsageProjection(projection *TaskUsageProjection, raw json.RawMessage) error {
	var envelope usageProjectionEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode usage envelope for Task projection: %w", err)
	}
	if envelope.Availability != "known" && envelope.Availability != "unknown" && envelope.Availability != "invalid" {
		return errors.New("usage envelope has invalid availability state")
	}
	if err := addFieldCount(&projection.AvailabilityFields, envelope.Availability); err != nil { return err }
	unknown := envelope.Availability != "known"
	for _, field := range []struct {
		name string
		value *usageProjectionField[int64]
		counts *UsageFieldCounts
		total *int64
	}{
		{"input_tokens", &envelope.InputTokens, &projection.InputTokenFields, &projection.KnownInputTokens},
		{"output_tokens", &envelope.OutputTokens, &projection.OutputTokenFields, &projection.KnownOutputTokens},
		{"total_tokens", &envelope.TotalTokens, &projection.TotalTokenFields, &projection.KnownTotalTokens},
	} {
		if err := addFieldCount(field.counts, field.value.State); err != nil { return fmt.Errorf("%s: %w", field.name, err) }
		if field.value.State != "known" { unknown = true; continue }
		if field.value.Value == nil || *field.value.Value < 0 { return fmt.Errorf("%s is known without a valid value", field.name) }
		if *field.total > int64(^uint64(0)>>1)-*field.value.Value { return fmt.Errorf("%s projection overflows int64", field.name) }
		*field.total += *field.value.Value
	}
	if err := addFieldCount(&projection.CostAmountFields, envelope.CostAmount.State); err != nil { return fmt.Errorf("cost_amount: %w", err) }
	if err := addFieldCount(&projection.CostCurrencyFields, envelope.CostCurrency.State); err != nil { return fmt.Errorf("cost_currency: %w", err) }
	if envelope.CostAmount.State != "known" || envelope.CostCurrency.State != "known" {
		unknown = true
	} else {
		if envelope.CostAmount.Value == nil || envelope.CostCurrency.Value == nil || strings.TrimSpace(*envelope.CostCurrency.Value) == "" {
			return errors.New("known monetary usage requires both amount and currency")
		}
		currency := strings.ToUpper(strings.TrimSpace(*envelope.CostCurrency.Value))
		amount, ok := new(big.Rat).SetString(envelope.CostAmount.Value.String())
		if !ok || amount.Sign() < 0 { return errors.New("known monetary usage has an invalid amount") }
		current := new(big.Rat)
		if existing := projection.KnownCostByCurrency[currency]; existing != "" {
			if _, ok := current.SetString(existing); !ok { return errors.New("existing monetary projection is invalid") }
		}
		current.Add(current, amount)
		if projection.KnownCostByCurrency == nil {
			projection.KnownCostByCurrency = make(map[string]string)
		}
		projection.KnownCostByCurrency[currency] = current.RatString()
	}
	projection.Calls++
	if unknown { projection.UsageUnknown++ }
	return nil
}

func addFieldCount(counts *UsageFieldCounts, state string) error {
	switch state {
	case "known": counts.Known++
	case "unknown": counts.Unknown++
	case "invalid": counts.Invalid++
	default: return fmt.Errorf("unsupported field state %q", state)
	}
	return nil
}

func validateTaskUsageLedger(ledger *TaskUsageLedger) error {
	if ledger == nil {
		return nil
	}
	if ledger.Version != TaskUsageLedgerVersion {
		return errors.New("unsupported Task usage ledger version")
	}
	p := ledger.Projection
	if p.Calls < 0 || p.UsageUnknown < 0 || p.UsageUnknown > p.Calls || p.KnownInputTokens < 0 || p.KnownOutputTokens < 0 || p.KnownTotalTokens < 0 {
		return errors.New("invalid Task usage projection totals")
	}
	for _, counts := range []UsageFieldCounts{
		p.AvailabilityFields, p.InputTokenFields, p.OutputTokenFields,
		p.TotalTokenFields, p.CostAmountFields, p.CostCurrencyFields,
	} {
		if counts.Known < 0 || counts.Unknown < 0 || counts.Invalid < 0 || counts.Known > p.Calls || counts.Unknown > p.Calls-counts.Known || counts.Invalid > p.Calls-counts.Known-counts.Unknown {
			return errors.New("invalid Task usage field counters")
		}
	}
	for currency, amount := range p.KnownCostByCurrency {
		value, ok := new(big.Rat).SetString(amount)
		if currency == "" || !ok || value.Sign() < 0 {
			return errors.New("invalid Task usage monetary projection")
		}
	}
	if ledger.Budget != nil {
		if err := validateTaskUsageBudget(*ledger.Budget); err != nil { return fmt.Errorf("invalid Task usage budget: %w", err) }
	}
	if pending := ledger.PendingBudgetAuthorization; pending != nil {
		if ledger.Budget == nil || strings.TrimSpace(pending.Reason) == "" || len(pending.SourceUsageDigest) != 64 || strings.TrimSpace(pending.OpenedAt) == "" {
			return errors.New("Task usage budget authorization is incomplete")
		}
	}
	for _, approval := range ledger.BudgetApprovals {
		if strings.TrimSpace(approval.Actor) == "" || strings.TrimSpace(approval.Reason) == "" || strings.TrimSpace(approval.At) == "" || len(approval.SourceUsageDigest) != 64 {
			return errors.New("Task usage budget approval has incomplete audit history")
		}
		if err := validateTaskUsageBudget(approval.Previous); err != nil { return fmt.Errorf("invalid previous Task usage budget: %w", err) }
		if err := validateTaskUsageBudget(approval.New); err != nil { return fmt.Errorf("invalid approved Task usage budget: %w", err) }
		if err := validateUsageBudgetIncrease(approval.Previous, approval.New); err != nil { return fmt.Errorf("invalid Task usage budget increase: %w", err) }
	}
	for _, termination := range ledger.BudgetTerminations {
		if strings.TrimSpace(termination.Actor) == "" || strings.TrimSpace(termination.Reason) == "" || strings.TrimSpace(termination.At) == "" || len(termination.SourceUsageDigest) != 64 {
			return errors.New("Task usage budget termination has incomplete audit history")
		}
	}
	for _, record := range ledger.Records {
		var object map[string]json.RawMessage
		if !json.Valid(record) || json.Unmarshal(record, &object) != nil || object == nil {
			return errors.New("Task usage record must be a JSON object")
		}
		var event UsageEvent
		if err := json.Unmarshal(record, &event); err != nil || event.Version != TaskUsageLedgerVersion || event.ID == "" || event.TaskID == "" || event.WorkItemID == "" || event.AttemptID == "" || event.SessionID == "" || event.SessionTurn <= 0 || event.Provider == "" || event.Model == "" || strings.TrimSpace(event.Outcome) == "" || !json.Valid(event.Usage) {
			return errors.New("Task usage record has incomplete managed-call identity")
		}
	}
	return nil
}
