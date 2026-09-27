//go:build windows

package workflow

import (
	"encoding/json"
	"strconv"
	"sync"
	"testing"
)

func usageBudgetEvent(task, workItem, attempt, session string, turn int, tokens int64, cost string) UsageEvent {
	return UsageEvent{
		TaskID: TaskID(task), WorkItemID: WorkItemID(workItem), AttemptID: AttemptID(attempt),
		SessionID: session, SessionTurn: turn, Provider: "test-provider", Model: "test-model",
		Profile: "balanced", Outcome: "completed",
		Usage: json.RawMessage(`{"availability":"known","input_tokens":{"state":"known","value":4},"output_tokens":{"state":"known","value":6},"total_tokens":{"state":"known","value":` + strconv.FormatInt(tokens, 10) + `},"cost_amount":{"state":"known","value":` + cost + `},"cost_currency":{"state":"known","value":"USD"}}`),
	}
}

func usageBudgetFixture(t *testing.T) *Store {
	t.Helper()
	store, _, _ := durableFixture(t)
	return store
}

func TestTaskUsageBudgetAggregationReplayAndConcurrentApproval(t *testing.T) {
	store := usageBudgetFixture(t)
	if _, err := store.ConfigureTaskUsageBudget("task-1", TaskUsageBudget{TokenLimit: 10, MonetaryLimits: map[string]string{"USD": "1"}}); err != nil {
		t.Fatal(err)
	}

	first := usageBudgetEvent("task-1", "wi-0001", "attempt-1", "session-1", 1, 10, "1")
	state, err := store.RecordUsageEvent("task-1", first)
	if err != nil { t.Fatal(err) }
	if !usageBudgetAuthorizationOpen(state) { t.Fatal("Task-wide budget breach did not open authorization") }
	state, err = store.RecordUsageEvent("task-1", first)
	if err != nil { t.Fatal(err) }
	if state.Protocol.Usage.Projection.Calls != 1 || len(state.Protocol.Usage.Records) != 1 {
		t.Fatalf("replayed Provider call was counted more than once: %+v", state.Protocol.Usage.Projection)
	}
	if _, err := store.RecordUsageEvent("task-1", usageBudgetEvent("task-1", "wi-0002", "attempt-2", "session-2", 1, 10, "1")); err != nil {
		t.Fatal(err)
	}
	state, err = store.Load("task-1")
	if err != nil { t.Fatal(err) }
	projection := state.Protocol.Usage.Projection
	if projection.Calls != 2 || projection.KnownTotalTokens != 20 || projection.KnownCostByCurrency["USD"] != "2" {
		t.Fatalf("Task-wide aggregation omitted a WorkItem: %+v", projection)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, approveErr := store.ApproveTaskUsageBudget("task-1", "operator", "concurrent approval", nil)
			errs <- approveErr
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs { if err != nil { t.Fatalf("serialized repeated approval: %v", err) } }
	state, err = store.Load("task-1")
	if err != nil { t.Fatal(err) }
	if len(state.Protocol.Usage.BudgetApprovals) != 2 || state.Protocol.Usage.Budget.TokenLimit != 17 || state.Protocol.Usage.Budget.MonetaryLimits["USD"] != "169/100" {
		t.Fatalf("concurrent approvals lost an update or did not apply two 30%% increases: %+v", state.Protocol.Usage)
	}
}

func TestTaskUsageBudgetOverrideLimitsAndBlockedRecovery(t *testing.T) {
	store := usageBudgetFixture(t)
	if _, err := store.ConfigureTaskUsageBudget("task-1", TaskUsageBudget{TokenLimit: 10, MonetaryLimits: map[string]string{"USD": "1"}}); err != nil { t.Fatal(err) }
	if _, err := store.RecordUsageEvent("task-1", usageBudgetEvent("task-1", "wi-0001", "attempt-1", "session-1", 1, 20, "2")); err != nil { t.Fatal(err) }
	if _, err := store.ApproveTaskUsageBudget("task-1", "operator", "invalid override", &TaskUsageBudget{TokenLimit: 30, MonetaryLimits: map[string]string{"USD": "1"}}); err == nil {
		t.Fatal("override that failed to increase every existing dimension was accepted")
	}
	state, err := store.ApproveTaskUsageBudget("task-1", "operator", "approved override", &TaskUsageBudget{TokenLimit: 30, MonetaryLimits: map[string]string{"USD": "3"}})
	if err != nil { t.Fatal(err) }
	if len(state.Protocol.Usage.BudgetApprovals) != 1 || state.Protocol.Usage.Budget.TokenLimit != 30 || state.Protocol.Usage.Budget.MonetaryLimits["USD"] != "3" {
		t.Fatalf("explicit budget override was not recorded: %+v", state.Protocol.Usage)
	}

	// A second Task gives termination a pending authorization to preserve.
	store2 := usageBudgetFixture(t)
	if _, err := store2.ConfigureTaskUsageBudget("task-1", TaskUsageBudget{TokenLimit: 10, MonetaryLimits: map[string]string{"USD": "1"}}); err != nil { t.Fatal(err) }
	if _, err := store2.RecordUsageEvent("task-1", usageBudgetEvent("task-1", "wi-0001", "attempt-1", "session-1", 1, 10, "1")); err != nil { t.Fatal(err) }
	blocked, err := store2.TerminateTaskUsageBudget("task-1", "operator", "stop at budget")
	if err != nil { t.Fatal(err) }
	if blocked.Protocol.Stop == nil || blocked.Protocol.Stop.Kind != "usage-budget-termination" || len(blocked.Protocol.Usage.BudgetTerminations) != 1 {
		t.Fatalf("human termination did not preserve its BLOCKED decision: %+v", blocked.Protocol)
	}
	if got := DeriveSummary(blocked).Execution; got != ExecutionBlocked {
		t.Fatalf("human budget termination derived execution %q, want blocked", got)
	}

	restarted := NewStore(store2.Root)
	loaded, err := restarted.Load("task-1")
	if err != nil { t.Fatal(err) }
	decision := *loaded.Protocol.Stop
	recovered, err := restarted.StopExecution("task-1", loaded.StateRevision, decision, true)
	if err != nil { t.Fatal(err) }
	if recovered.Protocol.Stop != nil || len(recovered.Protocol.Usage.Records) != 1 || len(recovered.Protocol.Usage.BudgetTerminations) != 1 || recovered.Protocol.Usage.PendingBudgetAuthorization == nil {
		t.Fatalf("recovery discarded accounting or termination history: %+v", recovered.Protocol.Usage)
	}
}
