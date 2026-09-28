package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidationInputsTrackContentAndDiscovery(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil { t.Fatal(err) }
	}
	capture := func() ValidationInputs {
		t.Helper()
		value, err := CaptureValidationInputs(root, []string{"."}, nil, "fixed-toolchain")
		if err != nil { t.Fatal(err) }
		return value
	}
	write("code.go", "first")
	original := capture()
	write(".ai/task/reports/output.log", "new log and memory")
	if got := capture(); got.Digest() != original.Digest() { t.Fatal("runtime projection invalidated code evidence") }
	write("code.go", "other") // Same size and same dirty status still differ.
	changed := capture()
	if ok, _ := EvidenceApplicability(&original, changed, true, true); ok { t.Fatal("changed bytes reused old validation") }
	write("fixture/input.txt", "new fixture")
	added := capture()
	if added.Digest() == changed.Digest() { t.Fatal("new fixture was absent from discovery") }
	if err := os.Remove(filepath.Join(root, "fixture/input.txt")); err != nil { t.Fatal(err) }
	if restored := capture(); restored.Digest() != changed.Digest() { t.Fatal("discovery is not deterministic") }
}

func TestReadExecutionArtifactAcceptsTaskRootReachedThroughSymlink(t *testing.T) {
	base := t.TempDir()
	realRoot := filepath.Join(base, "real-runtime")
	linkedRoot := filepath.Join(base, "linked-runtime")
	if err := os.MkdirAll(realRoot, 0o755); err != nil { t.Fatal(err) }
	if err := os.Symlink(realRoot, linkedRoot); err != nil { t.Skipf("symlink unavailable: %v", err) }

	store := NewStore(linkedRoot)
	id := TaskID("task-1")
	path := store.path(id, "reports", "protocol", "evidence.json")
	content := []byte(`{"value":"kept inside Task"}`)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, content, 0o644); err != nil { t.Fatal(err) }
	ref := ActorReference{Kind: "test-evidence", Path: "reports/protocol/evidence.json", SHA256: contentDigest(content)}
	var decoded struct { Value string `json:"value"` }
	if err := store.ReadExecutionArtifact(id, ref, &decoded); err != nil { t.Fatal(err) }
	if decoded.Value != "kept inside Task" { t.Fatalf("unexpected artifact: %+v", decoded) }
}

func TestReadExecutionArtifactAcceptsOnlyTaskLocalRoutingPlan(t *testing.T) {
	store := NewStore(t.TempDir())
	id := TaskID("task-1")
	content := []byte(`{"task_id":"task-1"}`)
	path := store.path(id, "routing-plan.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, content, 0o644); err != nil { t.Fatal(err) }
	ref := ActorReference{Kind: "routing-plan", Path: "routing-plan.json", SHA256: contentDigest(content)}
	var decoded struct { TaskID string `json:"task_id"` }
	if err := store.ReadExecutionArtifact(id, ref, &decoded); err != nil { t.Fatal(err) }
	if decoded.TaskID != string(id) { t.Fatalf("unexpected Task ID: %s", decoded.TaskID) }

	other := store.path(id, "other.json")
	if err := os.WriteFile(other, content, 0o644); err != nil { t.Fatal(err) }
	ref.Path = "other.json"
	if err := store.ReadExecutionArtifact(id, ref, &decoded); err == nil { t.Fatal("accepted another Task-root file as routing plan") }
	ref.Kind, ref.Path = "other-artifact", "routing-plan.json"
	if err := store.ReadExecutionArtifact(id, ref, &decoded); err == nil { t.Fatal("accepted Task-root file as report evidence") }
	ref.Kind = "routing-plan"
	ref.SHA256 = contentDigest([]byte("different"))
	if err := store.ReadExecutionArtifact(id, ref, &decoded); err == nil { t.Fatal("accepted altered routing plan digest") }
}

func TestLoadExecutionInputReusesFrozenRequestIdentity(t *testing.T) {
	store := NewStore(t.TempDir())
	request := PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", SessionID: "session-1", ExpectedSessionTurn: 1, Workspace: "workspace"}
	if _, _, exists, err := store.LoadExecutionInput(request); err != nil || exists { t.Fatalf("unexpected absent input: exists=%v err=%v", exists, err) }
	input := ExecutionInput{SchemaVersion: 1, RequestID: ExecutionRequestID(request), TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, SessionID: request.SessionID, Turn: request.ExpectedSessionTurn, Workspace: request.Workspace, Actor: ActorCoder}
	content, err := json.MarshalIndent(input, "", "  ")
	if err != nil { t.Fatal(err) }
	content = append(content, '\n')
	path := store.path(request.TaskID, "reports", "inputs", input.RequestID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, content, 0o644); err != nil { t.Fatal(err) }
	ref, saved, exists, err := store.LoadExecutionInput(request)
	if err != nil || !exists { t.Fatalf("failed to recover frozen input: exists=%v err=%v", exists, err) }
	if ref.Path != "reports/inputs/"+input.RequestID+".json" || ref.SHA256 != contentDigest(content) || saved.RequestID != input.RequestID { t.Fatalf("wrong recovered input: ref=%+v saved=%+v", ref, saved) }
	input.Workspace = "different-workspace"
	content, err = json.MarshalIndent(input, "", "  ")
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil { t.Fatal(err) }
	if _, _, _, err := store.LoadExecutionInput(request); err == nil { t.Fatal("accepted a frozen input bound to a different workspace") }
}

func TestReportSectionsDistinguishMissingEmptyAndUnknown(t *testing.T) {
	for _, section := range []ReportSection{{}, {State: "empty", Items: []string{}}, {State: "unknown", Items: []string{}, Reason: ""}, {State: "known", Items: []string{}}} {
		if section.validate("coverage") == nil { t.Fatalf("accepted missing or ambiguous facts: %+v", section) }
	}
	for _, section := range []ReportSection{{State: "empty", Items: []string{}, Reason: "no public interface changed"}, {State: "unknown", Items: []string{}, Reason: "original reason was not recorded"}, {State: "unverified", Items: []string{}, Reason: "tests were not authorized"}} {
		if err := section.validate("coverage"); err != nil { t.Fatal(err) }
	}
}

func TestImplementationReportBindsGenerationAndChangedBytes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "code.go"), []byte("original"), 0o644); err != nil { t.Fatal(err) }
	request := PreparedAgentRequest{TaskID: "task-1", WorkItemID: "wi-1", AttemptID: "attempt-1", SessionID: "session-1", ExpectedSessionTurn: 2, InputReference: &ActorReference{Kind: "execution-input", Path: "input.json", SHA256: "input-digest"}}
	empty := ReportSection{State: "empty", Items: []string{}, Reason: "not applicable to this change"}
	report := ImplementationReport{SchemaVersion: 1, RequestID: ExecutionRequestID(request), TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, SessionID: request.SessionID, Turn: 2, Actor: ActorCoder, InputSHA256: "input-digest", Coverage: empty, Interfaces: empty, SideEffects: empty, TestEntrypoints: empty, Decisions: empty, Limitations: empty, Risks: empty, Validation: empty, History: empty, Changes: []ActorReference{{Kind: "content", Path: "code.go", SHA256: contentDigest([]byte("original"))}}, References: []ActorReference{}}
	encode := func() string { data, err := json.Marshal(map[string]any{"report": report}); if err != nil { t.Fatal(err) }; return string(data) }
	if _, err := ValidateImplementationReport(encode(), request, root); err != nil { t.Fatal(err) }
	report.Turn++
	if _, err := ValidateImplementationReport(encode(), request, root); err == nil { t.Fatal("accepted a later turn") }
	report.Turn--
	if err := os.WriteFile(filepath.Join(root, "code.go"), []byte("modified"), 0o644); err != nil { t.Fatal(err) }
	if _, err := ValidateImplementationReport(encode(), request, root); err == nil { t.Fatal("accepted an old changed-content digest") }
}

func TestLegacyEvidenceDoesNotCreateValidationOrAuthorization(t *testing.T) {
	current := ValidationInputs{SchemaVersion: 1, Scope: []string{"."}, Toolchain: "fixed"}
	if decision := AssessExecutionReuse(ExecutionEvidence{}, current, false, false); decision.Action != "reconcile" { t.Fatal(decision) }
	evidence := ExecutionEvidence{SchemaVersion: 1, RequestID: "original", Inputs: &current, Controlled: true, Passed: true, CompileRunID: "compile-1", TestsRequired: true}
	if decision := AssessExecutionReuse(evidence, current, false, false); decision.Action != "authorization-required" { t.Fatal(decision) }
	evidence.TestRunIDs = []string{"test-1"}
	if decision := AssessExecutionReuse(evidence, current, false, false); decision.Action != "report-only" { t.Fatal(decision) }
	evidence.Report = &ActorReference{Kind: "execution-report", Path: "report.json", SHA256: "digest"}
	if decision := AssessExecutionReuse(evidence, current, false, false); decision.Action != "reuse" { t.Fatal(decision) }
	if decision := AssessExecutionReuse(evidence, current, false, true); decision.Action != "wait" { t.Fatal(decision) }
	evidence.Accepted = true
	if decision := AssessExecutionReuse(evidence, current, false, true); decision.Action != "reuse" { t.Fatal(decision) }
	current.Toolchain = "changed"
	// Keep the saved version independent of the current value.
	previous := *evidence.Inputs
	previous.Toolchain = "fixed"
	evidence.Inputs = &previous
	if decision := AssessExecutionReuse(evidence, current, true, true); decision.Action != "validate" { t.Fatal(decision) }
}

func TestUnknownScriptEnvironmentCannotReusePassedEvidence(t *testing.T) {
	inputs := ValidationInputs{SchemaVersion: 1, Scope: []string{"."}, Toolchain: "incomplete:script-environment"}
	if ok, _ := EvidenceApplicability(&inputs, inputs, true, true); ok { t.Fatal("partial toolchain identity was treated as proof of applicability") }
}
