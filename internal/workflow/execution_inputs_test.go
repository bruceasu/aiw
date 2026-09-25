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
