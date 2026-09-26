package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemediationProblemIdentityIsStableAndEvidenceBound(t *testing.T) {
	base := FailureReport{
		SchemaVersion:     FailureReportSchemaVersion,
		TaskID:            "task-1",
		EventSequence:     7,
		WorkItemID:        "wi-0001",
		AttemptID:         "attempt-1",
		Category:          "runner",
		Detail:            "agent exited before producing a result",
		EvidenceReference: "sessions/session-1/output.json",
		Retryable:         true,
		Owner:             "supervisor",
		RecommendedAction: "review the recorded evidence",
		RecordedAt:        "2026-09-27T00:00:00Z",
	}

	first, firstID, err := BuildRemediationProblem(base)
	if err != nil {
		t.Fatal(err)
	}
	second, secondID, err := BuildRemediationProblem(base)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || firstID != secondID {
		t.Fatalf("same failure produced different identity: %#v/%s vs %#v/%s", first, firstID, second, secondID)
	}

	changed := base
	changed.Detail = "agent exited with a different failure"
	_, changedID, err := BuildRemediationProblem(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedID == firstID {
		t.Fatal("changed failure detail reused the same problem identity")
	}
}

func TestPersistRemediationReportIsIdempotentAndPreservesConflicts(t *testing.T) {
	store, state, report := newAwaitingRemediationFixture(t)

	first, err := store.PersistRemediationReport(state, report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.PersistRemediationReport(state, report)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("idempotent report write changed reference: %#v vs %#v", first, second)
	}

	conflict := report
	conflict.Problem.Detail = "conflicting history must not overwrite the original"
	if _, err := store.PersistRemediationReport(state, conflict); err == nil || !strings.Contains(err.Error(), "conflicts with preserved problem history") {
		t.Fatalf("expected preserved-history conflict, got %v", err)
	}
}

func TestReadRemediationResponseStrictlyValidatesDigestOptionAndJSON(t *testing.T) {
	store, _, report := newAwaitingRemediationFixture(t)
	option := report.Options[0]
	valid := RemediationResponse{
		SchemaVersion: RemediationSchemaVersion,
		ProblemID:     report.ProblemID,
		ReportDigest:  report.ReportDigest,
		OptionID:      option.ID,
		Operator:      "operator-1",
		RiskConfirmed: true,
		RecordedAt:    "2026-09-27T00:00:00Z",
	}

	writeRemediationResponse(t, store, report, marshalJSON(t, valid))
	if _, err := store.ReadRemediationResponse("task-1", report); err != nil {
		t.Fatal(err)
	}

	unknownField := marshalJSON(t, valid)
	unknownField = []byte(strings.TrimSuffix(string(unknownField), "\n"))
	unknownField = append(unknownField[:len(unknownField)-1], []byte(`,"unexpected":true}`)...)
	writeRemediationResponse(t, store, report, unknownField)
	if _, err := store.ReadRemediationResponse("task-1", report); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}

	writeRemediationResponse(t, store, report, append(marshalJSON(t, valid), []byte("{}")...))
	if _, err := store.ReadRemediationResponse("task-1", report); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("expected trailing-data rejection, got %v", err)
	}

	invalid := valid
	invalid.ReportDigest = "stale-digest"
	writeRemediationResponse(t, store, report, marshalJSON(t, invalid))
	if _, err := store.ReadRemediationResponse("task-1", report); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected stale-digest rejection, got %v", err)
	}

	invalid = valid
	invalid.OptionID = "not-an-option"
	writeRemediationResponse(t, store, report, marshalJSON(t, invalid))
	if _, err := store.ReadRemediationResponse("task-1", report); err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("expected unknown-option rejection, got %v", err)
	}
}

func TestConsumeRemediationResponseIsOneShotAndRetainsHistory(t *testing.T) {
	store, _, report := newAwaitingRemediationFixture(t)
	response := RemediationResponse{
		SchemaVersion: RemediationSchemaVersion,
		ProblemID:     report.ProblemID,
		ReportDigest:  report.ReportDigest,
		OptionID:      report.Options[0].ID,
		Operator:      "operator-1",
		RiskConfirmed: true,
		RecordedAt:    "2026-09-27T00:00:00Z",
	}
	writeRemediationResponse(t, store, report, marshalJSON(t, response))

	updated, err := store.ConsumeRemediationResponse("task-1", response)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Automation.Cursor.Result != "remediation-response-consumed" {
		t.Fatalf("cursor result = %q", updated.Automation.Cursor.Result)
	}
	if _, err := store.ConsumeRemediationResponse("task-1", response); err == nil || !strings.Contains(err.Error(), "no human remediation response is pending") {
		t.Fatalf("expected duplicate response rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(report.ReportPath))); err != nil {
		t.Fatalf("remediation report was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(report.HumanResponsePath))); err != nil {
		t.Fatalf("response history was removed: %v", err)
	}
}

func newAwaitingRemediationFixture(t *testing.T) (*Store, RuntimeState, RemediationReport) {
	t.Helper()
	store := NewStore(t.TempDir())
	state := compatibleState()
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	problem := RemediationProblem{
		TaskID:            "task-1",
		WorkItemID:        "wi-0001",
		AttemptID:         "attempt-1",
		EventSequence:     7,
		Category:          "runner",
		Detail:            "agent result is unavailable",
		EvidenceReference: "sessions/session-1/output.json",
		FailureDigest:     "frozen-failure-digest",
	}
	report := NewRemediationReport(problem, nil, RemediationAwaitingHuman, 0, DefaultRemediationOptions(problem, true))
	if _, err := store.AwaitRemediation("task-1", report); err != nil {
		t.Fatal(err)
	}
	_, report, err := store.ReadPendingRemediation("task-1")
	if err != nil {
		t.Fatal(err)
	}
	return store, state, report
}

func writeRemediationResponse(t *testing.T, store *Store, report RemediationReport, content []byte) {
	t.Helper()
	path := filepath.Join(store.Root, "tasks", "task-1", filepath.FromSlash(report.HumanResponsePath))
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func marshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
