//go:build windows

package cli

import (
	"encoding/json"
	"os"
	"testing"

	"aiw/internal/ai"
	"aiw/internal/workflow"
)

func TestProfileAtOrAboveUsesLevelThenNameAndFallsForward(t *testing.T) {
	profiles := map[string]ai.Profile{
		"zeta":  {Name: "zeta", Level: 2},
		"alpha": {Name: "alpha", Level: 2},
		"expert": {Name: "expert", Level: 4},
	}

	selected, ok := profileAtOrAbove(profiles, 2)
	if !ok || selected.Name != "alpha" || selected.Level != 2 {
		t.Fatalf("same-level selection = %+v, %t; want alpha at level 2", selected, ok)
	}
	selected, ok = profileAtOrAbove(profiles, 3)
	if !ok || selected.Name != "expert" || selected.Level != 4 {
		t.Fatalf("next-higher selection = %+v, %t; want expert at level 4", selected, ok)
	}
	if _, ok := profileAtOrAbove(profiles, 5); ok {
		t.Fatal("selection unexpectedly found a Profile above the highest configured level")
	}
}

func TestResolveSupervisedAISelectionEscalatesAndPreservesAudit(t *testing.T) {
	_, store := routingTestTask(t)
	config := `[ai]
provider = "openai"
model = "base-model"
[ai.profiles.balanced]
provider = "openai"
model = "balanced-model"
level = 1
reasoning_intensity = "standard"
[ai.profiles.zeta]
provider = "openai"
model = "zeta-model"
level = 2
reasoning_intensity = "high"
[ai.profiles.advanced]
provider = "openai"
model = "advanced-model"
level = 2
reasoning_intensity = "extended"
`
	if err := os.WriteFile("aiw.toml", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	workItemID := workflow.WorkItemID("wi-0001")
	prior, err := json.Marshal(workflow.UsageEvent{
		WorkItemID: workItemID, Profile: "balanced", Provider: "openai", Model: "balanced-model",
		DifficultyLevel: 1, ReasoningIntensity: "standard",
	})
	if err != nil {
		t.Fatal(err)
	}
	state := workflow.RuntimeState{
		WorkItems: []workflow.WorkItem{{ID: workItemID, State: workflow.WorkItemReady}},
		Attempts: []workflow.Attempt{{
			ID: "attempt-1", WorkItemID: workItemID, SessionID: "session-1",
			State: workflow.AttemptCompleted, Outcome: &workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeNoProgress},
		}},
		Protocol: &workflow.ExecutionProtocol{Usage: &workflow.TaskUsageLedger{Records: []json.RawMessage{prior}}},
	}

	selection, err := resolveSupervisedAISelection(store, "task-1", state, workItemID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Profile != "advanced" || selection.Model != "advanced-model" || selection.Level != 2 || selection.ReasoningIntensity != "extended" {
		t.Fatalf("escalated selection = %+v", selection)
	}
	if selection.RequestedLevel != 2 || selection.AdjustmentReason != "unresolved_agent_round" {
		t.Fatalf("difficulty adjustment = %+v", selection)
	}
	if selection.PreviousProfile != "balanced" || selection.PreviousModel != "balanced-model" || selection.PreviousLevel != 1 || selection.PreviousReasoningIntensity != "standard" {
		t.Fatalf("difficulty audit provenance = %+v", selection)
	}
}

func TestResolveSupervisedAISelectionRetainsProfileWhenNoHigherLevelExists(t *testing.T) {
	_, store := routingTestTask(t)
	config := `[ai]
provider = "openai"
model = "base-model"
[ai.profiles.balanced]
provider = "openai"
model = "balanced-model"
level = 1
reasoning_intensity = "standard"
`
	if err := os.WriteFile("aiw.toml", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	workItemID := workflow.WorkItemID("wi-0001")
	state := workflow.RuntimeState{
		WorkItems: []workflow.WorkItem{{ID: workItemID, State: workflow.WorkItemReady}},
		Attempts: []workflow.Attempt{{
			ID: "attempt-1", WorkItemID: workItemID, SessionID: "session-1",
			State: workflow.AttemptCompleted, Outcome: &workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeNoProgress},
		}},
	}

	selection, err := resolveSupervisedAISelection(store, "task-1", state, workItemID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Profile != "balanced" || selection.Level != 1 || selection.ReasoningIntensity != "standard" {
		t.Fatalf("fallback selection = %+v", selection)
	}
	if selection.RequestedLevel != 2 || selection.AdjustmentReason != "no_higher_profile_available" {
		t.Fatalf("fallback audit reason = %+v", selection)
	}
}
