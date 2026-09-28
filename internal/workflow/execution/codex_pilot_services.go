package execution

import (
	"errors"
	"fmt"
	"strings"

	"aiw/internal/workflow"
)

// ConnectCodexPilot installs the real Schema 10 Coder validators and existing
// generation-budget ledger for the trusted-local Codex pilot. Activation
// verification is mandatory. This composition intentionally grants neither
// Tester nor acceptance capability; the pilot stops after compilation.
func ConnectCodexPilot(store *workflow.Store, host *CodexStageHost, policy workflow.GenerationRoutingPolicy, verifyActivation func(workflow.RuntimeState, []workflow.ActorReference) error) error {
	if store == nil || host == nil || host.Store != store || host.Sessions == nil || strings.TrimSpace(host.ID) == "" {
		return errors.New("Codex pilot requires its real Store, Session store, and named host")
	}
	if verifyActivation == nil {
		return errors.New("Codex pilot requires a non-empty activation verifier")
	}
	if strings.TrimSpace(policy.Default) == "" || len(policy.Profiles) == 0 {
		return errors.New("Codex pilot requires an explicit Coder profile policy")
	}
	for _, profile := range policy.Profiles {
		if !strings.EqualFold(profile.Provider, "codex") {
			return fmt.Errorf("Codex pilot profile %q resolves to unsupported provider %q", profile.Profile, profile.Provider)
		}
	}
	services := &workflow.ExecutionServices{VerifyActivation: verifyActivation}
	verification := &workflow.VerificationService{Store: store, Host: host}
	if err := verification.Connect(services); err != nil {
		return err
	}
	// GenerationBudgetService requires both roles for the general Schema 10
	// ledger. Tester is deliberately assigned the same Codex-only policy while
	// the host rejects that phase; this does not authorize a Tester dispatch.
	policies := map[workflow.ActorKind]workflow.GenerationRoutingPolicy{
		workflow.ActorCoder: policy,
		workflow.ActorTester: policy,
	}
	if err := (workflow.GenerationBudgetService{Policies: policies}).Connect(services); err != nil {
		return err
	}
	services.ValidateAcceptance = func(workflow.RuntimeState, workflow.AcceptanceCandidate) error {
		return errors.New("Codex pilot stops before WorkItem acceptance")
	}
	store.ExecutionServices = services
	return nil
}
