package cli

import (
	"encoding/json"
	"fmt"
	"sort"

	"aiw/internal/workflow"
)

// printUsageStatus renders the bounded operational usage view. It deliberately
// reads only the normalized projection and selected Profile identity; usage
// envelopes and Provider response evidence remain private to the ledger.
func printUsageStatus(taskID string, state workflow.RuntimeState) {
	fmt.Printf("Task %s status\n", taskID)
	ledger := (*workflow.TaskUsageLedger)(nil)
	if state.Protocol != nil {
		ledger = state.Protocol.Usage
	}
	if ledger == nil {
		fmt.Println("BUDGET\tunconfigured; historical usage unknown")
		fmt.Println("PENDING AUTHORIZATION\tnone")
		fmt.Println("SELECTED PROFILE\tnot recorded")
		fmt.Println("PARTIAL USAGE\tledger unavailable; historical usage unknown")
	} else {
		printUsageBudgetStatus(ledger)
		printPendingUsageAuthorization(ledger)
		printSelectedUsageProfile(ledger)
		printPartialUsageStatus(ledger)
	}
	if workflow.DeriveSummary(state).Status == workflow.TaskBlocked {
		reason := "no reason recorded"
		if state.Protocol != nil && state.Protocol.Stop != nil && state.Protocol.Stop.Reason != "" {
			reason = state.Protocol.Stop.Reason
		}
		fmt.Printf("BLOCKED REASON\t%s\n", reason)
		fmt.Println("RECOVERY\treview the stop reason and resume only through the explicit Workflow recovery path")
	}
}

func printUsageBudgetStatus(ledger *workflow.TaskUsageLedger) {
	if ledger.Budget == nil {
		fmt.Println("BUDGET\tunconfigured")
		return
	}
	status := "within_limits"
	if ledger.PendingBudgetAuthorization != nil {
		status = "authorization_pending"
	}
	projection := ledger.Projection
	fmt.Printf("BUDGET\t%s; tokens=%d/%d; known_cost=%s\n", status,
		projection.KnownTotalTokens, ledger.Budget.TokenLimit,
		formatKnownCosts(projection.KnownCostByCurrency))
	for _, currency := range sortedCurrencyKeys(ledger.Budget.MonetaryLimits) {
		fmt.Printf("BUDGET LIMIT\t%s=%s\n", currency, ledger.Budget.MonetaryLimits[currency])
	}
}

func printPendingUsageAuthorization(ledger *workflow.TaskUsageLedger) {
	pending := ledger.PendingBudgetAuthorization
	if pending == nil {
		fmt.Println("PENDING AUTHORIZATION\tnone")
		return
	}
	fmt.Printf("PENDING AUTHORIZATION\topened=%s; reason=%s\n", pending.OpenedAt, pending.Reason)
}

func printSelectedUsageProfile(ledger *workflow.TaskUsageLedger) {
	var selected workflow.UsageEvent
	for _, raw := range ledger.Records {
		var event workflow.UsageEvent
		if json.Unmarshal(raw, &event) == nil {
			selected = event
		}
	}
	if selected.Profile == "" {
		fmt.Println("SELECTED PROFILE\tnot recorded")
		return
	}
	fmt.Printf("SELECTED PROFILE\t%s (level=%d; provider=%s; model=%s; reasoning=%s)\n",
		selected.Profile, selected.DifficultyLevel, selected.Provider, selected.Model, selected.ReasoningIntensity)
}

func printPartialUsageStatus(ledger *workflow.TaskUsageLedger) {
	p := ledger.Projection
	fmt.Printf("PARTIAL USAGE\tusage_unknown=%d; availability=%s; input_tokens=%s; output_tokens=%s; total_tokens=%s; cost_amount=%s; cost_currency=%s\n",
		p.UsageUnknown,
		formatUsageFieldCounts(p.AvailabilityFields),
		formatUsageFieldCounts(p.InputTokenFields),
		formatUsageFieldCounts(p.OutputTokenFields),
		formatUsageFieldCounts(p.TotalTokenFields),
		formatUsageFieldCounts(p.CostAmountFields),
		formatUsageFieldCounts(p.CostCurrencyFields))
}

func formatUsageFieldCounts(counts workflow.UsageFieldCounts) string {
	return fmt.Sprintf("known:%d,unknown:%d,invalid:%d", counts.Known, counts.Unknown, counts.Invalid)
}

func formatKnownCosts(costs map[string]string) string {
	if len(costs) == 0 {
		return "none"
	}
	keys := sortedCurrencyKeys(costs)
	values := make([]string, 0, len(keys))
	for _, currency := range keys {
		values = append(values, currency+"="+costs[currency])
	}
	return fmt.Sprint(values)
}

func sortedCurrencyKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for currency := range values {
		keys = append(keys, currency)
	}
	sort.Strings(keys)
	return keys
}
