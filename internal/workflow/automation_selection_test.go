package workflow

import "testing"

func TestRecordAutomationRejectsRequestForAnotherTask(t *testing.T) {
	store := NewStore(t.TempDir())
	state := RuntimeState{SchemaVersion: SchemaVersion, Task: TaskReference{ID: "task-1", Workspace: ".", Kind: WorkspacePrimary}, Planning: PlanningReady, Delivery: DeliveryUnmanaged}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	_, err := store.RecordAutomation("task-1", "plan", AutomationCursor{Result: "agent-request-prepared"}, &PreparedAgentRequest{TaskID: "task-2", WorkItemID: "wi-0001", AttemptID: "attempt-1", Workspace: "."})
	if err == nil {
		t.Fatal("expected request Task mismatch to fail")
	}
}
