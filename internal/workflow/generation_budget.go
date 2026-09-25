package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// GenerationBudget is preserved inside the same Core commit as stages and
// leases. Counters are derived from unique request facts, never reset by a
// successful compile, another Actor, a restart, Stop or a new Attempt.
type GenerationBudget struct {
	Version int `json:"version"`
	Unknown []WorkItemID `json:"unknown,omitempty"`
	Accounts []GenerationAccount `json:"accounts"`
	Requests []GenerationCharge `json:"requests"`
	Failures []GenerationFailure `json:"failures"`
	TestRepairs []WorkItemID `json:"test_repairs"`
}

type GenerationAccount struct {
	WorkItemID WorkItemID `json:"work_item_id"`
	Actor ActorKind `json:"actor"`
	Models []AISelection `json:"models"`
	Index int `json:"index"`
	Escalations int `json:"escalations"`
}

type GenerationCharge struct {
	Request StageRequest `json:"request"`
	State string `json:"state"` // reserved, spent, released
	Repair bool `json:"repair"`
	Status string `json:"status,omitempty"`
	FailureClass string `json:"failure_class,omitempty"`
}

type GenerationFailure struct {
	WorkItemID WorkItemID `json:"work_item_id"`
	Actor ActorKind `json:"actor"`
	Generation string `json:"generation"`
	ObservedBy string `json:"observed_by"`
	Model AISelection `json:"model"`
}

type GenerationBudgetService struct {
	Policies map[ActorKind]GenerationRoutingPolicy
}

func (service GenerationBudgetService) Connect(services *ExecutionServices) error {
	if services == nil { return errors.New("execution services are required") }
	// Copy host configuration; callers cannot mutate an in-flight snapshot.
	policies := map[ActorKind]GenerationRoutingPolicy{}
	for _, actor := range []ActorKind{ActorCoder, ActorTester} {
		policy, ok := service.Policies[actor]
		if !ok { return errors.New("Coder and Tester routing policies are required") }
		if _, err := distinctModels(policy.Profiles); err != nil { return err }
		found := false
		for _, model := range policy.Profiles { if model.Profile == policy.Default { found = true } }
		if !found { return errors.New("routing default is not configured") }
		policy.Profiles = append([]AISelection(nil), policy.Profiles...)
		policies[actor] = policy
	}
	services.Budget = func(state *RuntimeState, request StageRequest, operation string) error {
		if operation == "reserve" && isGeneration(request) {
			ledger, err := readGenerationBudget(*state)
			if err != nil { return err }
			if generationAccount(&ledger, request.WorkItemID, request.Actor) == nil {
				policy := policies[request.Actor]
				allowed := false
				if request.Route != nil && request.Route.Router == policy.Router {
					for i := range policy.Profiles {
						if policy.Override && policy.Profiles[i].Profile != policy.Default { continue }
						models, err := distinctModels(policy.Profiles[i:])
						if err == nil && equalJSON(models, request.Route.Models) { allowed = true; break }
					}
				}
				if !allowed { return errors.New("generation route differs from the configured host policy") }
			}
		}
		return applyGenerationBudget(state, request, operation)
	}
	services.MigrateBudget = migrateGenerationBudget
	return nil
}

// GenerationBudgetSnapshot exposes the one Task-level account and migration
// gap list for diagnostics without changing counters or requesting a model.
func GenerationBudgetSnapshot(state RuntimeState) (GenerationBudget, error) {
	return readGenerationBudget(state)
}

func readGenerationBudget(state RuntimeState) (GenerationBudget, error) {
	var ledger GenerationBudget
	if state.Protocol == nil { return ledger, errors.New("generation budget requires durable execution") }
	decoder := json.NewDecoder(bytes.NewReader(state.Protocol.Budget))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ledger); err != nil { return ledger, fmt.Errorf("preserve unsupported generation budget: %w", err) }
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF { return ledger, errors.New("generation budget contains trailing data") }
	if ledger.Version != 1 { return ledger, errors.New("unsupported generation budget version") }
	for i, account := range ledger.Accounts {
		models, err := distinctModels(account.Models)
		if err != nil || len(models) != len(account.Models) || account.WorkItemID == "" || (account.Actor != ActorCoder && account.Actor != ActorTester) || account.Index < 0 || account.Index >= len(models) || account.Escalations != account.Index || account.Escalations > 2 { return ledger, errors.New("invalid generation tier account") }
		for _, prior := range ledger.Accounts[:i] { if prior.WorkItemID == account.WorkItemID && prior.Actor == account.Actor { return ledger, errors.New("duplicate generation account") } }
	}
	seen := map[string]bool{}
	for _, charge := range ledger.Requests {
		if charge.Request.ID == "" || charge.Request.TaskID != state.Task.ID || seen[charge.Request.ID] { return ledger, errors.New("duplicate or cross-Task generation charge") }
		seen[charge.Request.ID] = true
		if charge.State != "reserved" && charge.State != "spent" && charge.State != "released" { return ledger, errors.New("invalid generation charge state") }
		if charge.Repair && !isGeneration(charge.Request) { return ledger, errors.New("non-generation cannot spend a repair") }
		if charge.Status != "" && charge.State != "spent" { return ledger, errors.New("terminal generation charge was not spent") }
		if repairUsage(ledger, charge.Request.WorkItemID) > 6 { return ledger, errors.New("repair ledger exceeds six dispatches") }
	}
	failed := map[string]bool{}
	for _, failure := range ledger.Failures {
		origin := generationCharge(&ledger, failure.Generation)
		observation := generationCharge(&ledger, failure.ObservedBy)
		if origin == nil || observation == nil || origin.State != "spent" || !isGeneration(origin.Request) || origin.Request.WorkItemID != failure.WorkItemID || origin.Request.Actor != failure.Actor || origin.Request.Model == nil || *origin.Request.Model != failure.Model || failed[failure.Generation] { return ledger, errors.New("invalid or duplicate generation failure provenance") }
		failed[failure.Generation] = true
	}
	if state.Protocol.BudgetKnown != (len(ledger.Unknown) == 0) { return ledger, errors.New("budget knowledge does not match preserved migration gaps") }
	return ledger, nil
}

func saveGenerationBudget(state *RuntimeState, ledger GenerationBudget) error {
	content, err := json.Marshal(ledger)
	if err != nil { return err }
	state.Protocol.Budget = content
	return nil
}

// Legacy counters do not identify distinct generations or model tiers. Keep
// one Task-level list of affected Work Items rather than guessing balances.
func migrateGenerationBudget(state *RuntimeState) error {
	if state.Protocol == nil || string(state.Protocol.Budget) != "null" { return errors.New("migration cannot replace an existing budget") }
	ledger := GenerationBudget{Version: 1}
	for _, item := range state.WorkItems {
		used := item.CompileFailureCount != 0 || item.NoProgressCount != 0 || item.LastOutputReference != ""
		for _, attempt := range state.Attempts { if attempt.WorkItemID == item.ID { used = true } }
		if used { ledger.Unknown = append(ledger.Unknown, item.ID) }
	}
	state.Protocol.BudgetKnown = len(ledger.Unknown) == 0
	return saveGenerationBudget(state, ledger)
}

func isGeneration(request StageRequest) bool { return request.Phase == PhaseCoder || request.Phase == PhaseTester }

func generationAccount(ledger *GenerationBudget, item WorkItemID, actor ActorKind) *GenerationAccount {
	for i := range ledger.Accounts { if ledger.Accounts[i].WorkItemID == item && ledger.Accounts[i].Actor == actor { return &ledger.Accounts[i] } }
	return nil
}

func generationCharge(ledger *GenerationBudget, id string) *GenerationCharge {
	for i := range ledger.Requests { if ledger.Requests[i].Request.ID == id { return &ledger.Requests[i] } }
	return nil
}

func generationFailures(ledger GenerationBudget, item WorkItemID, actor ActorKind, model AISelection) int {
	count := 0
	for _, failure := range ledger.Failures { if failure.WorkItemID == item && failure.Actor == actor && sameModel(failure.Model, model) { count++ } }
	return count
}

func repairUsage(ledger GenerationBudget, item WorkItemID) int {
	count := 0
	for _, charge := range ledger.Requests { if charge.Request.WorkItemID == item && charge.Repair && charge.State != "released" { count++ } }
	return count
}

// Once tests hand work back, all further Coder/Tester generations belong
// to that repair loop, including reauthoring tests after a Coder success.
// Intermediate success must not create an uncharged seventh generation.
func repairRequired(ledger GenerationBudget, _ RuntimeState, item WorkItemID) bool {
	for _, key := range ledger.TestRepairs { if key == item { return true } }
	return false
}

// budgetGenerationSources freezes provenance before validation can be delayed.
// Compile/Runner never become the owners of a model-quality failure.
func budgetGenerationSources(state RuntimeState, request StageRequest) map[ActorKind]string {
	sources := map[ActorKind]string{}
	for _, record := range state.Protocol.Requests {
		r := record.Request
		if r.WorkItemID == request.WorkItemID && r.AttemptID == request.AttemptID && isGeneration(r) && record.Consumed { sources[r.Actor] = r.ID }
	}
	if isGeneration(request) { sources[request.Actor] = request.ID }
	return sources
}

func reserveGeneration(ledger *GenerationBudget, state RuntimeState, request StageRequest) error {
	if prior := generationCharge(ledger, request.ID); prior != nil {
		if !equalJSON(prior.Request, request) { return errors.New("request budget identity cannot be rewritten") }
		return nil
	}
	charge := GenerationCharge{Request: request, State: "reserved"}
	if isGeneration(request) {
		charge.Repair = repairRequired(*ledger, state, request.WorkItemID)
		if charge.Repair && repairUsage(*ledger, request.WorkItemID) >= 6 { return errors.New("shared repair budget exhausted; a seventh repair is forbidden") }
		route := request.Route
		if route == nil || route.Reason == "" || len(route.ContextDigest) != 64 || request.Model == nil || route.Index < 0 || route.Index >= len(route.Models) || *request.Model != route.Models[route.Index] { return errors.New("generation requires its frozen routing decision") }
		models, err := distinctModels(route.Models)
		if err != nil || len(models) != len(route.Models) { return errors.New("routing tiers contain invalid models or aliases") }
		account := generationAccount(ledger, request.WorkItemID, request.Actor)
		if account == nil {
			if route.Index != 0 { return errors.New("initial generation must start at its routed tier") }
		} else {
			index, err := nextGenerationTier(*ledger, *account)
			if err != nil { return err }
			if route.Index != index || !equalJSON(route.Models, account.Models) { return errors.New("new request cannot bypass recorded failure or escalation budget") }
		}
	} else if (request.Phase == PhaseCompile || request.Phase == PhaseTest) && (request.Model != nil || request.Route != nil) { return errors.New("Compiler and Runner do not route models") }
	ledger.Requests = append(ledger.Requests, charge)
	return nil
}

func spendGeneration(ledger *GenerationBudget, charge *GenerationCharge) error {
	if charge.State == "spent" { return nil }
	if charge.State != "reserved" { return errors.New("released reservation cannot later be dispatched") }
	request := charge.Request
	if isGeneration(request) {
		route := request.Route
		account := generationAccount(ledger, request.WorkItemID, request.Actor)
		if account == nil {
			ledger.Accounts = append(ledger.Accounts, GenerationAccount{WorkItemID: request.WorkItemID, Actor: request.Actor, Models: append([]AISelection(nil), route.Models...)})
		} else if route.Index != account.Index {
			if route.Index != account.Index+1 || account.Escalations >= 2 { return errors.New("invalid escalation consumption") }
			account.Index, account.Escalations = route.Index, account.Escalations+1
		}
	}
	charge.State = "spent"
	return nil
}

func attributeGeneration(ledger *GenerationBudget, charge *GenerationCharge, class string) error {
	if class != "implementation" && class != "test" { return nil }
	if charge.Request.Phase == PhaseReport { return errors.New("report-only defects cannot obtain implementation repair allowances") }
	actor := ActorCoder
	if class == "test" { actor = ActorTester }
	origin := generationCharge(ledger, charge.Request.GenerationSources[actor])
	if origin == nil || !isGeneration(origin.Request) || origin.State != "spent" || origin.Request.Actor != actor || origin.Request.WorkItemID != charge.Request.WorkItemID || origin.Request.Model == nil { return errors.New("defect has no exact generating request; retain unknown attribution") }
	if charge.Request.Phase == PhaseTest {
		found := false
		for _, key := range ledger.TestRepairs { if key == charge.Request.WorkItemID { found = true } }
		if !found { ledger.TestRepairs = append(ledger.TestRepairs, charge.Request.WorkItemID) }
	}
	for _, failure := range ledger.Failures { if failure.Generation == origin.Request.ID { return nil } }
	ledger.Failures = append(ledger.Failures, GenerationFailure{WorkItemID: origin.Request.WorkItemID, Actor: actor, Generation: origin.Request.ID, ObservedBy: charge.Request.ID, Model: *origin.Request.Model})
	return nil
}

// applyGenerationBudget runs only inside an E02 conditional state transition.
// It checks limits at reserve, never before consuming a sixth successful result.
func applyGenerationBudget(state *RuntimeState, request StageRequest, operation string) error {
	ledger, err := readGenerationBudget(*state)
	if err != nil { return err }
	if !state.Protocol.BudgetKnown { return fmt.Errorf("legacy generation budget is unknown for %v", ledger.Unknown) }
	for _, item := range state.Protocol.Items { if item.WorkItemID == request.WorkItemID && item.Phase == PhaseAccepted { return errors.New("accepted Work Item accounting is closed") } }
	if operation == "reserve" {
		if err := reserveGeneration(&ledger, *state, request); err != nil { return err }
		return saveGenerationBudget(state, ledger)
	}
	charge := generationCharge(&ledger, request.ID)
	if charge == nil || !equalJSON(charge.Request, request) { return errors.New("budget observation differs from the reserved request") }
	switch {
	case operation == "unknown": // Keep the reservation; do not infer failure.
	case operation == "not-dispatched":
		if charge.State == "spent" { return errors.New("confirmed dispatch cannot be refunded") }
		charge.State = "released"
	case operation == "dispatched":
		if err := spendGeneration(&ledger, charge); err != nil { return err }
	case strings.HasPrefix(operation, "consume:"):
		parts := strings.Split(operation, ":")
		if len(parts) != 3 || (parts[1] != "passed" && parts[1] != "failed" && parts[1] != "blocked") { return errors.New("unsupported budget terminal status") }
		if charge.Status != "" {
			if charge.Status != parts[1] || charge.FailureClass != parts[2] { return errors.New("conflicting generation terminal fact") }
			return nil
		}
		if err := spendGeneration(&ledger, charge); err != nil { return err }
		charge.Status, charge.FailureClass = parts[1], parts[2]
		if charge.Status != "passed" { if err := attributeGeneration(&ledger, charge, charge.FailureClass); err != nil { return err } }
	case strings.HasPrefix(operation, "attribute:"):
		class := strings.TrimPrefix(operation, "attribute:")
		if charge.Status != "failed" || charge.FailureClass != "unattributed" { return errors.New("diagnosis requires the original unattributed failure") }
		if err := attributeGeneration(&ledger, charge, class); err != nil { return err }
		charge.FailureClass = class
	default: return errors.New("unsupported generation budget operation")
	}
	return saveGenerationBudget(state, ledger)
}
