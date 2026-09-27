package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"
)

const UsageBudgetGateID GateID = "ai-usage-budget-authorization"

// TaskUsageBudget stores independent Task-wide limits. Monetary amounts are
// exact decimal values represented as rational strings and remain per currency.
type TaskUsageBudget struct {
	TokenLimit int64             `json:"token_limit"`
	InputTokenLimit int64        `json:"input_token_limit,omitempty"`
	OutputTokenLimit int64       `json:"output_token_limit,omitempty"`
	MonetaryLimits map[string]string `json:"monetary_limits"`
}

type UsageBudgetAuthorization struct {
	Reason string `json:"reason"`
	SourceUsageDigest string `json:"source_usage_digest"`
	OpenedAt string `json:"opened_at"`
}

type UsageBudgetApproval struct {
	Actor string `json:"actor"`
	At string `json:"at"`
	Reason string `json:"reason"`
	Previous TaskUsageBudget `json:"previous_limits"`
	New TaskUsageBudget `json:"new_limits"`
	SourceUsageDigest string `json:"source_usage_digest"`
}

// UsageBudgetTermination preserves the human decision that ended execution
// while a budget authorization was pending. The pending authorization and all
// usage and recovery records remain available for a later explicit resume.
type UsageBudgetTermination struct {
	Actor             string `json:"actor"`
	At                string `json:"at"`
	Reason            string `json:"reason"`
	SourceUsageDigest string `json:"source_usage_digest"`
}

// TerminateTaskUsageBudget records the human termination through the ordered
// Task event path and persists an execution Stop. Reopening the Task does not
// erase this decision or its accounting history.
func (s *Store) TerminateTaskUsageBudget(id TaskID, actor, reason string) (RuntimeState, error) {
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return RuntimeState{}, errors.New("budget termination requires a human actor and reason")
	}
	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if state.SchemaVersion != DurableSchemaVersion || state.Protocol == nil || state.Protocol.Usage == nil {
		return RuntimeState{}, errors.New("Task usage budget is unavailable")
	}
	return s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.usage.budget-terminated", Detail: actor}, func(current *RuntimeState) error {
		if current.Protocol == nil || current.Protocol.Usage == nil {
			return errors.New("Task usage budget is unavailable")
		}
		ledger := current.Protocol.Usage
		if ledger.Budget == nil || ledger.PendingBudgetAuthorization == nil {
			return errors.New("no pending Task usage budget authorization")
		}
		if current.Protocol.Stop != nil {
			return errors.New("Task execution is already stopped")
		}
		digest, err := usageRecordsDigest(ledger.Records)
		if err != nil {
			return err
		}
		if digest != ledger.PendingBudgetAuthorization.SourceUsageDigest {
			return errors.New("budget termination source usage changed; refresh the pending authorization")
		}
		at := time.Now().UTC().Format(time.RFC3339Nano)
		decision := UsageBudgetTermination{Actor: actor, At: at, Reason: reason, SourceUsageDigest: digest}
		ref, err := s.persistProtocolArtifactLocked(id, "usage-budget-termination", decision)
		if err != nil {
			return err
		}
		ledger.BudgetTerminations = append(ledger.BudgetTerminations, decision)
		current.Protocol.Stop = &ExecutionStop{Kind: "usage-budget-termination", Reason: reason, Reference: ref}
		return nil
	})
}

// ConfigureTaskUsageBudget installs the initial limits through the ordered
// Schema 10 event path. Subsequent changes require a pending human decision.
func (s *Store) ConfigureTaskUsageBudget(id TaskID, budget TaskUsageBudget) (RuntimeState, error) {
	if err := validateTaskUsageBudget(budget); err != nil { return RuntimeState{}, err }
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, err }
	if state.SchemaVersion != DurableSchemaVersion || state.Protocol == nil { return RuntimeState{}, errors.New("usage budget requires the migrated Schema 10 protocol") }
	return s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.usage.budget-configured"}, func(current *RuntimeState) error {
		if current.Protocol == nil { return errors.New("Schema 10 execution protocol is missing") }
		ledger := current.Protocol.Usage
		if ledger == nil { ledger = &TaskUsageLedger{Version: TaskUsageLedgerVersion, Records: []json.RawMessage{}, Projection: TaskUsageProjection{KnownCostByCurrency: map[string]string{}}}; current.Protocol.Usage = ledger }
		if ledger.Budget != nil || ledger.PendingBudgetAuthorization != nil || len(ledger.BudgetApprovals) != 0 { return errors.New("Task usage budget is already configured") }
		copy := cloneTaskUsageBudget(budget)
		ledger.Budget = &copy
		return refreshUsageBudgetGate(current, ledger, time.Now().UTC())
	})
}

// ApproveTaskUsageBudget records a human authorization. A nil replacement
// applies a simultaneous 30 percent increase to the Token and every currency
// limit; an explicit replacement must increase all existing dimensions.
func (s *Store) ApproveTaskUsageBudget(id TaskID, actor, reason string, replacement *TaskUsageBudget) (RuntimeState, error) {
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if actor == "" || reason == "" { return RuntimeState{}, errors.New("budget approval requires a human actor and reason") }
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, err }
	if state.SchemaVersion != DurableSchemaVersion || state.Protocol == nil || state.Protocol.Usage == nil { return RuntimeState{}, errors.New("Task usage budget is unavailable") }
	return s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.usage.budget-approved", Detail: actor}, func(current *RuntimeState) error {
		ledger := current.Protocol.Usage
		if ledger == nil || ledger.Budget == nil || ledger.PendingBudgetAuthorization == nil { return errors.New("no pending Task usage budget authorization") }
		previous := cloneTaskUsageBudget(*ledger.Budget)
		var next TaskUsageBudget
		if replacement == nil {
			next, err = defaultUsageBudgetIncrease(previous)
			if err != nil { return err }
		} else {
			next = cloneTaskUsageBudget(*replacement)
		}
		if err := validateTaskUsageBudget(next); err != nil { return err }
		if err := validateUsageBudgetIncrease(previous, next); err != nil { return err }
		initial := previous
		if len(ledger.BudgetApprovals) != 0 {
			initial = ledger.BudgetApprovals[0].Previous
		}
		if err := validateUsageBudgetApprovalCeiling(initial, next); err != nil { return err }
		sourceDigest, err := usageRecordsDigest(ledger.Records)
		if err != nil { return err }
		pending := ledger.PendingBudgetAuthorization
		if sourceDigest != pending.SourceUsageDigest { return errors.New("budget approval source usage changed; refresh the pending authorization") }
		ledger.BudgetApprovals = append(ledger.BudgetApprovals, UsageBudgetApproval{
			Actor: actor, At: time.Now().UTC().Format(time.RFC3339Nano), Reason: reason,
			Previous: previous, New: cloneTaskUsageBudget(next), SourceUsageDigest: sourceDigest,
		})
		ledger.Budget = &next
		ledger.PendingBudgetAuthorization = nil
		closeUsageBudgetGate(current)
		return refreshUsageBudgetGate(current, ledger, time.Now().UTC())
	})
}

func validateTaskUsageBudget(budget TaskUsageBudget) error {
	if budget.TokenLimit < 0 || budget.InputTokenLimit < 0 || budget.OutputTokenLimit < 0 {
		return errors.New("Task usage budget limits cannot be negative")
	}
	if budget.InputTokenLimit == 0 && budget.OutputTokenLimit == 0 {
		if budget.TokenLimit == 0 { return errors.New("Task usage budget requires positive Token limits") }
	} else if budget.InputTokenLimit == 0 || budget.OutputTokenLimit == 0 || budget.TokenLimit != 0 {
		return errors.New("Task usage budget requires both input and output limits without a legacy total limit")
	}
	for currency, raw := range budget.MonetaryLimits {
		amount, ok := new(big.Rat).SetString(raw)
		if strings.TrimSpace(currency) == "" || currency != strings.ToUpper(strings.TrimSpace(currency)) || !ok || amount.Sign() <= 0 {
			return fmt.Errorf("invalid monetary budget for currency %q", currency)
		}
	}
	return nil
}

func validateUsageBudgetIncrease(previous, next TaskUsageBudget) error {
	if previous.TokenLimit > 0 {
		if next.TokenLimit <= previous.TokenLimit || next.InputTokenLimit != 0 || next.OutputTokenLimit != 0 {
			return errors.New("approved legacy Token limit must increase without changing budget mode")
		}
	} else if next.TokenLimit != 0 || next.InputTokenLimit <= previous.InputTokenLimit || next.OutputTokenLimit <= previous.OutputTokenLimit {
		return errors.New("approved input and output Token limits must both increase without changing budget mode")
	}
	for currency, oldRaw := range previous.MonetaryLimits {
		newRaw, ok := next.MonetaryLimits[currency]
		if !ok { return fmt.Errorf("approved budget must retain currency %s", currency) }
		oldValue, _ := new(big.Rat).SetString(oldRaw)
		newValue, _ := new(big.Rat).SetString(newRaw)
		if newValue.Cmp(oldValue) <= 0 { return fmt.Errorf("approved monetary limit for %s must increase", currency) }
	}
	return nil
}

// The first approval's previous limits are the immutable baseline for older
// ledgers too, so no migration or inferred reset is needed on restart.
func validateUsageBudgetApprovalCeiling(initial, next TaskUsageBudget) error {
	if initial.TokenLimit > 0 {
		if next.TokenLimit-initial.TokenLimit > initial.TokenLimit {
			return errors.New("approved Token limit exceeds twice the initial limit")
		}
	} else if next.InputTokenLimit-initial.InputTokenLimit > initial.InputTokenLimit ||
		next.OutputTokenLimit-initial.OutputTokenLimit > initial.OutputTokenLimit {
		return errors.New("approved input or output Token limit exceeds twice its initial limit")
	}
	for currency, nextRaw := range next.MonetaryLimits {
		initialRaw, ok := initial.MonetaryLimits[currency]
		if !ok { return fmt.Errorf("approved budget cannot add currency %s without an initial limit", currency) }
		initialValue, _ := new(big.Rat).SetString(initialRaw)
		nextValue, _ := new(big.Rat).SetString(nextRaw)
		if nextValue.Cmp(new(big.Rat).Mul(initialValue, big.NewRat(2, 1))) > 0 {
			return fmt.Errorf("approved monetary limit for %s exceeds twice its initial limit", currency)
		}
	}
	return nil
}

func defaultUsageBudgetIncrease(budget TaskUsageBudget) (TaskUsageBudget, error) {
	next := TaskUsageBudget{MonetaryLimits: make(map[string]string, len(budget.MonetaryLimits))}
	var err error
	if budget.TokenLimit > 0 {
		next.TokenLimit, err = increaseTokenLimit(budget.TokenLimit)
	} else {
		next.InputTokenLimit, err = increaseTokenLimit(budget.InputTokenLimit)
		if err == nil { next.OutputTokenLimit, err = increaseTokenLimit(budget.OutputTokenLimit) }
	}
	if err != nil { return TaskUsageBudget{}, err }
	for currency, raw := range budget.MonetaryLimits {
		value, ok := new(big.Rat).SetString(raw)
		if !ok { return TaskUsageBudget{}, fmt.Errorf("invalid existing monetary limit for %s", currency) }
		value.Mul(value, big.NewRat(13, 10))
		next.MonetaryLimits[currency] = value.RatString()
	}
	return next, nil
}

func increaseTokenLimit(limit int64) (int64, error) {
	if limit > (math.MaxInt64-99)/130 { return 0, errors.New("Token budget increase overflows int64") }
	return (limit*130 + 99)/100, nil
}

func refreshUsageBudgetGate(state *RuntimeState, ledger *TaskUsageLedger, now time.Time) error {
	if ledger.Budget == nil { return nil }
	breached, dimensions := taskUsageBudgetBreaches(ledger.Projection, *ledger.Budget)
	if !breached {
		if ledger.PendingBudgetAuthorization == nil { closeUsageBudgetGate(state) }
		return nil
	}
	digest, err := usageRecordsDigest(ledger.Records)
	if err != nil { return err }
	openedAt := now.UTC().Format(time.RFC3339Nano)
	if ledger.PendingBudgetAuthorization != nil { openedAt = ledger.PendingBudgetAuthorization.OpenedAt }
	ledger.PendingBudgetAuthorization = &UsageBudgetAuthorization{
		Reason: "known Task usage reached its configured limit: "+strings.Join(dimensions, ", "),
		SourceUsageDigest: digest, OpenedAt: openedAt,
	}
	reason := ledger.PendingBudgetAuthorization.Reason
	for i := range state.Gates {
		if state.Gates[i].ID == UsageBudgetGateID { state.Gates[i].Kind, state.Gates[i].State, state.Gates[i].Reason = GateAuthorization, GateOpen, reason; return nil }
	}
	state.Gates = append(state.Gates, Gate{ID: UsageBudgetGateID, Kind: GateAuthorization, State: GateOpen, Reason: reason})
	return nil
}

func taskUsageBudgetBreaches(projection TaskUsageProjection, budget TaskUsageBudget) (bool, []string) {
	var dimensions []string
	if budget.TokenLimit > 0 && projection.KnownTotalTokens >= budget.TokenLimit { dimensions = append(dimensions, "tokens") }
	if budget.InputTokenLimit > 0 && projection.KnownInputTokens >= budget.InputTokenLimit { dimensions = append(dimensions, "input_tokens") }
	if budget.OutputTokenLimit > 0 && projection.KnownOutputTokens >= budget.OutputTokenLimit { dimensions = append(dimensions, "output_tokens") }
	for currency, limitRaw := range budget.MonetaryLimits {
		limit, ok := new(big.Rat).SetString(limitRaw)
		if !ok { continue }
		used := new(big.Rat)
		if raw := projection.KnownCostByCurrency[currency]; raw != "" { if _, ok := used.SetString(raw); !ok { continue } }
		if used.Cmp(limit) >= 0 { dimensions = append(dimensions, "cost_"+currency) }
	}
	sort.Strings(dimensions)
	return len(dimensions) > 0, dimensions
}

func usageRecordsDigest(records []json.RawMessage) (string, error) {
	content, err := json.Marshal(records)
	if err != nil { return "", err }
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:]), nil
}

func cloneTaskUsageBudget(budget TaskUsageBudget) TaskUsageBudget {
	copy := TaskUsageBudget{TokenLimit: budget.TokenLimit, InputTokenLimit: budget.InputTokenLimit, OutputTokenLimit: budget.OutputTokenLimit, MonetaryLimits: make(map[string]string, len(budget.MonetaryLimits))}
	for currency, amount := range budget.MonetaryLimits { copy.MonetaryLimits[currency] = amount }
	return copy
}

func closeUsageBudgetGate(state *RuntimeState) {
	for i := range state.Gates { if state.Gates[i].ID == UsageBudgetGateID && state.Gates[i].State == GateOpen { state.Gates[i].State = GateResolved } }
}

func usageBudgetAuthorizationOpen(state RuntimeState) bool {
	for _, gate := range state.Gates { if gate.ID == UsageBudgetGateID && gate.State == GateOpen { return true } }
	return false
}
