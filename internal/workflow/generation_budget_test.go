package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func generationBudgetFixture(t *testing.T) RuntimeState {
	t.Helper()
	state := RuntimeState{SchemaVersion: DurableSchemaVersion, Task: TaskReference{ID: "task"}, Protocol: &ExecutionProtocol{BudgetKnown: true, Items: []ItemExecution{{WorkItemID: "item", AttemptID: "attempt", Phase: PhaseCoder}}}}
	if err := saveGenerationBudget(&state, GenerationBudget{Version: 1}); err != nil { t.Fatal(err) }
	return state
}

func generationModels() []AISelection {
	return []AISelection{{Profile: "one", Provider: "p", Model: "m1", Digest: "snapshot1"}, {Profile: "two", Provider: "p", Model: "m2", Digest: "snapshot2"}, {Profile: "three", Provider: "p", Model: "m3", Digest: "snapshot3"}, {Profile: "four", Provider: "p", Model: "m4", Digest: "snapshot4"}}
}

func budgetRequest(id string, actor ActorKind, index int) StageRequest {
	models := generationModels()
	phase := PhaseCoder
	if actor == ActorTester { phase = PhaseTester }
	return StageRequest{ActorRequest: ActorRequest{ID: id, TaskID: "task", WorkItemID: "item", AttemptID: "attempt", Actor: actor}, Phase: phase, Model: &models[index], Route: &GenerationRoute{Models: models, Index: index, Reason: "configured routing", ContextDigest: strings.Repeat("a", 64)}, GenerationSources: map[ActorKind]string{actor: id}}
}

func budgetStep(t *testing.T, state *RuntimeState, request StageRequest, operation string) {
	t.Helper()
	if err := applyGenerationBudget(state, request, operation); err != nil { t.Fatalf("%s %s: %v", request.ID, operation, err) }
	if operation == "reserve" { state.Protocol.Items[0].CurrentRequest = request.ID }
}

func budgetRoundTrip(t *testing.T, state RuntimeState) RuntimeState {
	t.Helper()
	data, err := json.Marshal(state)
	if err != nil { t.Fatal(err) }
	var recovered RuntimeState
	if err := json.Unmarshal(data, &recovered); err != nil { t.Fatal(err) }
	return recovered
}

func TestGenerationBudgetDeduplicatesDelayedDefectsAcrossRestart(t *testing.T) {
	state := generationBudgetFixture(t)
	first := budgetRequest("coder1", ActorCoder, 0)
	budgetStep(t, &state, first, "reserve")
	budgetStep(t, &state, first, "consume:passed:")
	compile := StageRequest{ActorRequest: ActorRequest{ID: "compile1", TaskID: "task", WorkItemID: "item", Actor: ActorCompiler}, Phase: PhaseCompile, GenerationSources: map[ActorKind]string{ActorCoder: first.ID}}
	budgetStep(t, &state, compile, "reserve")
	budgetStep(t, &state, compile, "consume:failed:implementation")
	state = budgetRoundTrip(t, state)
	budgetStep(t, &state, compile, "consume:failed:implementation")
	compile.ID = "compile2"
	budgetStep(t, &state, compile, "reserve")
	budgetStep(t, &state, compile, "consume:failed:implementation")
	second := budgetRequest("coder2", ActorCoder, 0)
	budgetStep(t, &state, second, "reserve")
	budgetStep(t, &state, second, "consume:passed:")
	compile.ID, compile.GenerationSources = "compile3", map[ActorKind]string{ActorCoder: second.ID}
	budgetStep(t, &state, compile, "reserve")
	budgetStep(t, &state, compile, "consume:failed:implementation")
	ledger, err := readGenerationBudget(state)
	if err != nil { t.Fatal(err) }
	if len(ledger.Failures) != 2 || ledger.Failures[0].Generation != first.ID || ledger.Failures[1].Generation != second.ID { t.Fatalf("unexpected attribution: %+v", ledger.Failures) }
	if err := applyGenerationBudget(&state, budgetRequest("wrong-tier", ActorCoder, 0), "reserve"); err == nil { t.Fatal("two failed generations did not require escalation") }
}

func TestGenerationBudgetReservationsAndEscalationLimits(t *testing.T) {
	state := generationBudgetFixture(t)
	for index := 0; index < 3; index++ {
		for failure := 0; failure < 2; failure++ {
			request := budgetRequest(fmt.Sprintf("coder-%d-%d", index, failure), ActorCoder, index)
			budgetStep(t, &state, request, "reserve")
			budgetStep(t, &state, request, "consume:failed:implementation")
		}
		if index == 2 { continue }
		request := budgetRequest(fmt.Sprintf("released-%d", index), ActorCoder, index+1)
		budgetStep(t, &state, request, "reserve")
		budgetStep(t, &state, request, "unknown")
		ledger, err := readGenerationBudget(state)
		if err != nil { t.Fatal(err) }
		if ledger.Accounts[0].Escalations != index { t.Fatal("reservation consumed an escalation") }
		budgetStep(t, &state, request, "not-dispatched")
		state = budgetRoundTrip(t, state)
	}
	if err := applyGenerationBudget(&state, budgetRequest("forbidden", ActorCoder, 3), "reserve"); err == nil { t.Fatal("third escalation was allowed") }
	// The other Actor does not inherit Coder failures or clear them.
	tester := budgetRequest("tester", ActorTester, 0)
	budgetStep(t, &state, tester, "reserve")
	budgetStep(t, &state, tester, "consume:failed:infrastructure")
	ledger, err := readGenerationBudget(state)
	if err != nil { t.Fatal(err) }
	if len(ledger.Failures) != 6 || ledger.Accounts[0].Escalations != 2 || ledger.Accounts[1].Escalations != 0 { t.Fatal("Actor budgets were reset or mixed") }
}

func TestGenerationBudgetSharedSixRepairsAndSuccessfulSixth(t *testing.T) {
	for _, successful := range []bool{true, false} {
		t.Run(fmt.Sprint(successful), func(t *testing.T) {
			state := generationBudgetFixture(t)
			coder := budgetRequest("initial", ActorCoder, 0)
			budgetStep(t, &state, coder, "reserve")
			budgetStep(t, &state, coder, "consume:passed:")
			runner := StageRequest{ActorRequest: ActorRequest{ID: "runner", TaskID: "task", WorkItemID: "item", Actor: ActorTester}, Phase: PhaseTest, GenerationSources: map[ActorKind]string{ActorCoder: coder.ID}}
			budgetStep(t, &state, runner, "reserve")
			budgetStep(t, &state, runner, "consume:failed:unattributed")
			budgetStep(t, &state, runner, "attribute:implementation")
			counts := map[ActorKind]int{ActorCoder: 1}
			for i := 1; i <= 6; i++ {
				actor, class := ActorCoder, "implementation"
				if i%2 == 0 { actor, class = ActorTester, "test" }
				request := budgetRequest(fmt.Sprintf("repair-%d", i), actor, counts[actor]/2)
				budgetStep(t, &state, request, "reserve")
				budgetStep(t, &state, request, "dispatched")
				status := "consume:failed:"+class
				if i == 6 && successful { status = "consume:passed:" }
				budgetStep(t, &state, request, status)
				counts[actor]++
				state = budgetRoundTrip(t, state)
			}
			ledger, err := readGenerationBudget(state)
			if err != nil { t.Fatal(err) }
			if repairUsage(ledger, "item") != 6 { t.Fatal("repairs were not shared") }
			if successful {
				if generationCharge(&ledger, "repair-6").Status != "passed" { t.Fatal("sixth success was rejected") }
				if err := applyGenerationBudget(&state, budgetRequest("seventh-after-success", ActorCoder, 2), "reserve"); err == nil { t.Fatal("intermediate success enabled an uncharged seventh generation") }
				// Validation consumes no seventh generation, even at the cap.
				runner.ID = "successful-validation"
				budgetStep(t, &state, runner, "reserve")
				budgetStep(t, &state, runner, "consume:passed:")
			} else if err := applyGenerationBudget(&state, budgetRequest("repair-7", ActorCoder, 2), "reserve"); err == nil { t.Fatal("seventh repair was allowed") }
		})
	}
}

func TestGenerationRoutingFallbackAliasesAndRecoverySnapshot(t *testing.T) {
	state := generationBudgetFixture(t)
	models := generationModels()
	alias := models[0]
	alias.Profile = "alias"
	policy := GenerationRoutingPolicy{Default: "one", Profiles: []AISelection{models[0], alias, models[1]}, Router: models[2]}
	context := GenerationRoutingContext{TaskID: "task", WorkItemID: "item", Actor: ActorCoder, Input: ActorReference{Kind: "context", Path: "context.json", SHA256: strings.Repeat("a", 64)}, Requirements: []string{"Keep the approved behavior."}}
	calls := 0
	route, err := RouteGeneration(state, context, policy, func(AISelection, GenerationRoutingContext) (GenerationRecommendation, error) { calls++; return GenerationRecommendation{}, errors.New("provider unavailable") })
	if err != nil { t.Fatal(err) }
	if calls != 1 || len(route.Models) != 2 || route.Models[0] != models[0] || route.Reason == "" { t.Fatal("fallback or alias deduplication failed") }
	request := budgetRequest("routed", ActorCoder, 0)
	request.Route, request.Model = &route, &route.Models[0]
	budgetStep(t, &state, request, "reserve")
	budgetStep(t, &state, request, "consume:passed:")
	state = budgetRoundTrip(t, state)
	changed := policy
	changed.Profiles = []AISelection{models[3]}
	recovered, err := RouteGeneration(state, context, changed, func(AISelection, GenerationRoutingContext) (GenerationRecommendation, error) { t.Fatal("recovery re-routed the recorded tier"); return GenerationRecommendation{}, nil })
	if err != nil || !equalJSON(recovered.Models, route.Models) { t.Fatalf("recorded selection changed: %+v %v", recovered, err) }
	context.Actor = ActorCompiler
	if _, err := RouteGeneration(state, context, policy, nil); err == nil { t.Fatal("Compiler was routed") }
}

func TestGenerationBudgetLegacyUnknownAndFutureVersion(t *testing.T) {
	state := generationBudgetFixture(t)
	state.Protocol.Budget = json.RawMessage(`null`)
	state.WorkItems = []WorkItem{{ID: "item", CompileFailureCount: 1}}
	if err := migrateGenerationBudget(&state); err != nil { t.Fatal(err) }
	if state.Protocol.BudgetKnown { t.Fatal("legacy count became a fresh known balance") }
	if err := applyGenerationBudget(&state, budgetRequest("new", ActorCoder, 0), "reserve"); err == nil { t.Fatal("unknown legacy budget allowed dispatch") }
	state.Protocol.Budget = json.RawMessage(`{"version":2,"unknown":["item"]}`)
	if _, err := readGenerationBudget(state); err == nil { t.Fatal("future budget version was accepted") }
}
