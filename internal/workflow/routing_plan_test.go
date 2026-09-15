package workflow

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRoutingPlanDefaultsAndPersistence(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	state := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	plan := NewRoutingPlan("task-1", "defaults", nil, CompilePlan{Reason: "no adapter"})
	want := map[string]string{"analysis": "fast", "coder": "balanced", "tester": "balanced", "verifier": "reasoning"}
	if !reflect.DeepEqual(plan.Profiles, want) {
		t.Fatalf("profiles = %v, want %v", plan.Profiles, want)
	}
	if _, _, err := store.PersistRoutingPlan("task-1", plan); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewStore(root).LoadRoutingPlan("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, plan) {
		t.Fatalf("reloaded plan = %+v, want %+v", loaded, plan)
	}
}

func TestCompilePlanWorkspaceTargets(t *testing.T) {
	workspace := t.TempDir()
	plan := CompilePlanForWorkspace(workspace)
	if plan.Available || plan.Reason == "" || len(plan.Targets) != 0 {
		t.Fatalf("empty workspace plan = %+v", plan)
	}
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example\n"), 0600); err != nil {
		t.Fatal(err)
	}
	plan = CompilePlanForWorkspace(workspace)
	if !plan.Available || len(plan.Targets) != 1 {
		t.Fatalf("Go workspace plan = %+v", plan)
	}
	if target := plan.Targets[0]; target.Kind != "go" || target.Command != "go" || !reflect.DeepEqual(target.PathOwners, []string{"."}) {
		t.Fatalf("Go target = %+v", target)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "scripts"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "scripts", "compile.sh"), []byte("#!/bin/sh\n"), 0600); err != nil {
		t.Fatal(err)
	}
	plan = CompilePlanForWorkspace(workspace)
	if !plan.Available || len(plan.Targets) != 1 {
		t.Fatalf("script workspace plan = %+v", plan)
	}
	if target := plan.Targets[0]; target.Kind != "script" || target.Script != "scripts/compile.sh" || target.Command != "" || len(target.Args) != 0 {
		t.Fatalf("script must supersede Go and remain repository-relative: %+v", target)
	}
}

func TestPreparedAISelectionSurvivesStoreRecovery(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	state := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	want := AISelection{Profile: "balanced", Provider: "provider-a", Model: "model-a", Digest: "secret-free-digest"}
	request := PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-1", AttemptID: "attempt-1", Workspace: ".", AISelection: &want}
	if _, err := store.RecordAutomation("task-1", "fingerprint", AutomationCursor{Result: "prepared"}, &request); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewStore(root).Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	recovered := loaded.Automation.PreparedRequest
	if recovered == nil || recovered.AISelection == nil || *recovered.AISelection != want {
		t.Fatalf("recovered request = %+v", recovered)
	}
}
