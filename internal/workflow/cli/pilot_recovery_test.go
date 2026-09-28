package cli

import (
	"encoding/json"
	"reflect"
	"testing"

	"aiw/internal/workflow"
)

func TestPilotReplacesOnlyProvedNotDispatchedCoderRequest(t *testing.T) {
	state := workflow.RuntimeState{Protocol: &workflow.ExecutionProtocol{Requests: []workflow.StageRecord{{Request: workflow.StageRequest{ActorRequest: workflow.ActorRequest{ID: "request-1"}, Phase: workflow.PhaseCoder}, Dispatch: "not-dispatched"}}}}
	if !pilotCoderRequestNotDispatched(state, "request-1") { t.Fatal("proved not-dispatched Coder request was not replaced") }
	state.Protocol.Requests[0].Dispatch = "unknown"
	if pilotCoderRequestNotDispatched(state, "request-1") { t.Fatal("unknown request was replaced") }
	state.Protocol.Requests[0].Dispatch = "not-dispatched"
	state.Protocol.Requests[0].Request.Phase = workflow.PhaseCompile
	if pilotCoderRequestNotDispatched(state, "request-1") { t.Fatal("compile request was replaced as Coder") }
}

func TestPilotCompileBindsCurrentTerminalCoderAfterRetry(t *testing.T) {
	report := workflow.ActorReference{Kind: "execution-report", Path: "reports/executions/current.json", SHA256: "current"}
	result := workflow.ActorReference{Kind: "stage-result", Path: "reports/stage-results/current.json", SHA256: "result"}
	item := workflow.ItemExecution{WorkItemID: "wi-0001", AttemptID: "attempt-1", Phase: workflow.PhaseCompile, CurrentRequest: "current", ValidatedReport: &report}
	state := workflow.RuntimeState{Protocol: &workflow.ExecutionProtocol{Requests: []workflow.StageRecord{
		{Request: workflow.StageRequest{ActorRequest: workflow.ActorRequest{ID: "old", WorkItemID: item.WorkItemID, AttemptID: item.AttemptID}, Phase: workflow.PhaseCoder}, Dispatch: "not-dispatched"},
		{Request: workflow.StageRequest{ActorRequest: workflow.ActorRequest{ID: "current", WorkItemID: item.WorkItemID, AttemptID: item.AttemptID}, Phase: workflow.PhaseCoder}, Dispatch: "terminal", Consumed: true, Result: &result},
	}}}
	record, err := pilotTerminalCoderForCompile(state, item)
	if err != nil || record.Request.ID != "current" { t.Fatalf("compile did not bind the current Coder retry: record=%+v err=%v", record, err) }
	state.Protocol.Requests[1].Request.AttemptID = "other-attempt"
	if _, err := pilotTerminalCoderForCompile(state, item); err == nil { t.Fatal("compile accepted a Coder result from another Attempt") }
	state.Protocol.Requests[1].Request.AttemptID = item.AttemptID
	state.Protocol.Requests[1].Consumed = false
	if _, err := pilotTerminalCoderForCompile(state, item); err == nil { t.Fatal("compile accepted an unconsumed Coder result") }
	state.Protocol.Requests[1].Consumed = true
	item.ValidatedReport = nil
	if _, err := pilotTerminalCoderForCompile(state, item); err == nil { t.Fatal("compile accepted a missing validated report") }
}

func TestPilotToolchainChangesIgnoresOnlyLiteralPath(t *testing.T) {
	frozen := "PATH=old-shell\nGOFLAGS=\ngo=original-binary\npython=original-binary\nwindows"
	current := "PATH=new-shell\nGOFLAGS=\ngo=original-binary\npython=original-binary\nwindows"
	if changes := pilotToolchainChanges(frozen, current); len(changes) != 0 { t.Fatalf("PATH alone blocked frozen input: %v", changes) }
	changed := "PATH=new-shell\nGOFLAGS=\ngo=different-binary\npython=original-binary\nwindows"
	if changes := pilotToolchainChanges(frozen, changed); !reflect.DeepEqual(changes, []string{"go"}) { t.Fatalf("tool binary drift was not reported: %v", changes) }
}

func TestPilotReportProblemRejectsUnstructuredOutput(t *testing.T) {
	if problem := pilotReportOutputProblem("**No changes**", "reports/executions/original.json"); problem == "" { t.Fatal("Markdown was treated as a valid report") }
	if problem := pilotReportOutputProblem(`{"other":true}`, "reports/executions/original.json"); problem == "" { t.Fatal("missing report was accepted") }
	if problem := pilotReportOutputProblem(`{"report":{"schema_version":1}}`, "reports/executions/original.json"); problem != "" { t.Fatalf("valid envelope was rejected: %s", problem) }
}

func TestPilotSchemaEqualAfterFrozenJSONRoundTrip(t *testing.T) {
	want := workflow.ImplementationReportOutputSchema()
	encoded, err := json.Marshal(want)
	if err != nil { t.Fatal(err) }
	var frozen map[string]any
	if err := json.Unmarshal(encoded, &frozen); err != nil { t.Fatal(err) }
	if !pilotSchemaEqual(frozen, want) { t.Fatal("equivalent persisted schema changed the frozen request") }
}
