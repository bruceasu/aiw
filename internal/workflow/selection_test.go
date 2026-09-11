package workflow

import "testing"

func TestSelectReadyMappedWorkItemSkipsDependenciesAndGates(t *testing.T) {
	state := compatibleState()
	state.WorkItems = []WorkItem{
		{ID: "wi-0001", Checklist: ChecklistReference{Item: "1.1"}, State: WorkItemReady},
		{ID: "wi-0002", Checklist: ChecklistReference{Item: "1.2"}, Dependencies: []WorkItemID{"wi-0001"}, State: WorkItemReady},
		{ID: "wi-0003", Checklist: ChecklistReference{Item: "1.3"}, State: WorkItemReady},
	}
	state.Gates = []Gate{{ID: "gate-1", WorkItemID: "wi-0001", Kind: GateDecision, State: GateOpen}}
	item, err := SelectReadyMappedWorkItem(state)
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != "wi-0003" {
		t.Fatalf("got %s, want wi-0003", item.ID)
	}
}
