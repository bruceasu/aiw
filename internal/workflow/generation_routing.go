package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
)

// GenerationRoutingPolicy is supplied by the configured host. Order is an
// explicit escalation order, never inferred from profile or model names.
// The router is fixed configuration and is not itself recursively routed.
type GenerationRoutingPolicy struct {
	Profiles []AISelection `json:"profiles"`
	Default string `json:"default"`
	Router AISelection `json:"router"`
	Override bool `json:"override,omitempty"`
}

type GenerationRoutingContext struct {
	TaskID TaskID `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	Actor ActorKind `json:"actor"`
	Input ActorReference `json:"input"`
	Requirements []string `json:"requirements"`
	Failures []GenerationFailure `json:"failures"`
}

// GenerationRoute travels with the immutable generation request and its input.
// A prepared but undispatched route does not spend an escalation.
type GenerationRoute struct {
	Models []AISelection `json:"models"`
	Index int `json:"index"`
	Reason string `json:"reason"`
	Router AISelection `json:"router"`
	ContextDigest string `json:"context_digest"`
}

type GenerationRecommendation struct {
	Profile string
	Reason string
}

type GenerationRecommender func(AISelection, GenerationRoutingContext) (GenerationRecommendation, error)

func validSelection(model AISelection) bool {
	return model.Profile != "" && model.Provider != "" && model.Model != "" && model.Digest != ""
}

func sameModel(a, b AISelection) bool { return a.Provider == b.Provider && a.Model == b.Model }

func jsonDigest(value any) string {
	data, _ := json.Marshal(value)
	return contentDigest(data)
}

func distinctModels(models []AISelection) ([]AISelection, error) {
	var result []AISelection
	for _, model := range models {
		if !validSelection(model) { return nil, errors.New("routing requires complete configured model snapshots") }
		duplicate := false
		for _, prior := range result { if sameModel(prior, model) { duplicate = true; break } }
		if !duplicate { result = append(result, model) }
	}
	if len(result) == 0 { return nil, errors.New("routing has no configured models") }
	return result, nil
}

// RouteGeneration is called outside Store locks, only for a genuinely needed
// new generation. Recovery always reuses the original StageRequest instead.
func RouteGeneration(state RuntimeState, context GenerationRoutingContext, policy GenerationRoutingPolicy, recommend GenerationRecommender) (GenerationRoute, error) {
	if context.TaskID != state.Task.ID || context.WorkItemID == "" || (context.Actor != ActorCoder && context.Actor != ActorTester && context.Actor != ActorVerifier) || !validProtocolReference(context.Input) { return GenerationRoute{}, errors.New("routing requires the exact Work Item, Actor and frozen context") }
	if state.Protocol == nil || state.Protocol.Stop != nil || !state.Protocol.BudgetKnown { return GenerationRoute{}, errors.New("routing is stopped or the budget is unknown") }
	if err := requireNoStageInFlight(state); err != nil { return GenerationRoute{}, err }
	ledger, err := readGenerationBudget(state)
	if err != nil { return GenerationRoute{}, err }
	context.Failures = nil
	for _, failure := range ledger.Failures { if failure.WorkItemID == context.WorkItemID && failure.Actor == context.Actor { context.Failures = append(context.Failures, failure) } }
	foundItem := context.Actor == ActorVerifier
	for _, item := range state.Protocol.Items {
		if item.WorkItemID != context.WorkItemID { continue }
		foundItem = true
		if item.Phase == PhaseAccepted && context.Actor != ActorVerifier { return GenerationRoute{}, errors.New("accepted Work Item has no new generation budget") }
		if context.Actor == ActorCoder && item.Phase != PhaseCoder || context.Actor == ActorTester && item.Phase != PhaseTester { return GenerationRoute{}, errors.New("generation is not the current required stage") }
	}
	if !foundItem { return GenerationRoute{}, errors.New("Work Item has no execution cursor") }
	if context.Actor != ActorVerifier && repairRequired(ledger, state, context.WorkItemID) && repairUsage(ledger, context.WorkItemID) >= 6 { return GenerationRoute{}, errors.New("shared repair budget exhausted") }
	if account := generationAccount(&ledger, context.WorkItemID, context.Actor); account != nil {
		index, err := nextGenerationTier(ledger, *account)
		if err != nil { return GenerationRoute{}, err }
		return GenerationRoute{Models: append([]AISelection(nil), account.Models...), Index: index, Reason: "resume the recorded Actor tier; two distinct failed generations require the next distinct model", ContextDigest: jsonDigest(context)}, nil
	}
	if _, err := distinctModels(policy.Profiles); err != nil { return GenerationRoute{}, err }
	selected := -1
	for i, profile := range policy.Profiles { if profile.Profile == policy.Default { selected = i; break } }
	if selected < 0 { return GenerationRoute{}, errors.New("routing default is not configured") }
	reason := "configured default: routing provider unavailable"
	if policy.Override { reason = "explicit one-request model override" }
	if !policy.Override && recommend != nil && validSelection(policy.Router) {
		choice, recommendationErr := recommend(policy.Router, context)
		reason = "configured default: routing recommendation failed or was invalid"
		if recommendationErr == nil && choice.Reason != "" {
			for i, profile := range policy.Profiles { if profile.Profile == choice.Profile { selected, reason = i, choice.Reason; break } }
		}
	}
	models, err := distinctModels(policy.Profiles[selected:])
	if err != nil { return GenerationRoute{}, err }
	return GenerationRoute{Models: models, Reason: reason, Router: policy.Router, ContextDigest: jsonDigest(context)}, nil
}

// ConnectBudgetedVerification composes E02/E03/E04 without activating or
// migrating a Task. The real host and platform verifier remain mandatory.
func ConnectBudgetedVerification(store *Store, host VerificationHost, policies map[ActorKind]GenerationRoutingPolicy, activation func(RuntimeState, []ActorReference) error) error {
	if store == nil || activation == nil { return errors.New("durable execution requires a Store and platform activation authority") }
	services := &ExecutionServices{VerifyActivation: activation}
	verification := &VerificationService{Store: store, Host: host}
	if err := verification.Connect(services); err != nil { return err }
	if err := (GenerationBudgetService{Policies: policies}).Connect(services); err != nil { return err }
	store.ExecutionServices = services
	return nil
}

func nextGenerationTier(ledger GenerationBudget, account GenerationAccount) (int, error) {
	index := account.Index
	if generationFailures(ledger, account.WorkItemID, account.Actor, account.Models[index]) < 2 { return index, nil }
	if account.Escalations >= 2 || index+1 >= len(account.Models) { return 0, fmt.Errorf("%s model failure budget exhausted; manual resolution required", account.Actor) }
	return index+1, nil
}
