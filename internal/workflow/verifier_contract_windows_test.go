//go:build windows

package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func verifierTestRow(id string, outcome VerifierOutcome) VerifierCoverage {
	applicable := true
	row := VerifierCoverage{CriterionID: id, Applicable: &applicable, Outcome: outcome, Basis: "Fixed fixture criterion and evidence.", Evidence: []string{}, Gaps: []string{}}
	if outcome == VerifierInconclusive { row.Gaps = []string{"Execution evidence is missing."} } else { row.Evidence = []string{strings.Repeat("a", 64)} }
	return row
}

func verifierTestReport() (AuxiliaryJob, VerifierReport) {
	snapshot := ActorReference{Kind: "verifier-snapshot", Path: "reports/fixed.json", SHA256: strings.Repeat("b", 64)}
	c := verifierContext{TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1", Snapshot: snapshot, Criteria: []string{"AC-one", "AC-two"}, Evidence: []string{strings.Repeat("a", 64)}}
	body, _ := json.Marshal(c)
	job := AuxiliaryJob{Kind: "verifier", Owner: c.TaskID, Prompt: "verifier-v2 "+string(body)+"\nFixed evidence only."}
	report := VerifierReport{
		SchemaVersion: 2, TaskID: c.TaskID, WorkItemID: c.WorkItemID, AttemptID: c.AttemptID,
		Snapshot: &snapshot, Outcome: VerifierPassed, Summary: "Both fixed criteria have evidence.", RecordedAt: "2026-09-25T01:00:00Z",
		Coverage: []VerifierCoverage{verifierTestRow("AC-one", VerifierPassed), verifierTestRow("AC-two", VerifierPassed)},
	}
	return job, report
}

func TestVerifierCoverageRequiresEvidenceAndConsistentOutcome(t *testing.T) {
	notApplicable := verifierTestRow("not-applicable", VerifierInconclusive)
	no := false
	notApplicable.Applicable, notApplicable.Gaps = &no, nil
	notApplicable.Evidence = []string{strings.Repeat("a", 64)}
	passWithGap := verifierTestRow("gap", VerifierPassed)
	passWithGap.Gaps = []string{"Unresolved."}
	failureWithoutEvidence := verifierTestRow("failure", VerifierFailed)
	failureWithoutEvidence.Evidence = nil
	missingApplicability := verifierTestRow("missing", VerifierPassed)
	missingApplicability.Applicable = nil
	for _, test := range []struct{name string; rows []VerifierCoverage; want VerifierOutcome; invalid bool}{
		{"all-proven", []VerifierCoverage{verifierTestRow("one", VerifierPassed)}, VerifierPassed, false},
		{"failure-before-gap", []VerifierCoverage{verifierTestRow("one", VerifierInconclusive), verifierTestRow("two", VerifierFailed)}, VerifierFailed, false},
		{"missing-proof", []VerifierCoverage{verifierTestRow("one", VerifierPassed), verifierTestRow("two", VerifierInconclusive)}, VerifierInconclusive, false},
		{"none-applicable", []VerifierCoverage{notApplicable}, VerifierInconclusive, false},
		{"explicit-exclusion", []VerifierCoverage{notApplicable, verifierTestRow("one", VerifierPassed)}, VerifierPassed, false},
		{"empty", nil, "", true},
		{"duplicate", []VerifierCoverage{verifierTestRow("one", VerifierPassed), verifierTestRow("one", VerifierPassed)}, "", true},
		{"pass-with-gap", []VerifierCoverage{passWithGap}, "", true},
		{"failure-without-evidence", []VerifierCoverage{failureWithoutEvidence}, "", true},
		{"missing-applicability", []VerifierCoverage{missingApplicability}, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := verifierCoverageOutcome(test.rows)
			if test.invalid { if err == nil { t.Fatal("invalid coverage was accepted") }; return }
			if err != nil || got != test.want { t.Fatalf("outcome=%s want=%s error=%v", got, test.want, err) }
		})
	}
}

func TestVerifierFrozenReportRejectsMismatchedInput(t *testing.T) {
	job, report := verifierTestReport()
	valid, err := json.Marshal(report)
	if err != nil { t.Fatal(err) }
	if err := ValidateVerifierOutput(job, string(valid)); err != nil { t.Fatal(err) }
	for _, test := range []struct{name string; change func(*VerifierReport)}{
		{"task", func(r *VerifierReport) { r.TaskID = "other-task" }},
		{"work-item", func(r *VerifierReport) { r.WorkItemID = "other-item" }},
		{"attempt", func(r *VerifierReport) { r.AttemptID = "other-attempt" }},
		{"snapshot", func(r *VerifierReport) { r.Snapshot.SHA256 = strings.Repeat("c", 64) }},
		{"foreign-evidence", func(r *VerifierReport) { r.Coverage[0].Evidence = []string{strings.Repeat("c", 64)} }},
		{"omitted-criterion", func(r *VerifierReport) { r.Coverage = r.Coverage[:1] }},
		{"unknown-criterion", func(r *VerifierReport) { r.Coverage[0].CriterionID = "invented" }},
		{"contradictory-outcome", func(r *VerifierReport) { r.Outcome = VerifierFailed }},
		{"legacy-schema", func(r *VerifierReport) { r.SchemaVersion = 1 }},
		{"empty-coverage", func(r *VerifierReport) { r.Coverage = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var changed VerifierReport
			if err := json.Unmarshal(valid, &changed); err != nil { t.Fatal(err) }
			test.change(&changed)
			body, err := json.Marshal(changed)
			if err != nil { t.Fatal(err) }
			if err := ValidateVerifierOutput(job, string(body)); err == nil { t.Fatal("report escaped the frozen input contract") }
		})
	}
	if err := ValidateVerifierOutput(job, "{}"); err == nil { t.Fatal("empty report was accepted") }
}

func TestVerifierNegativePublicationPreservesDevelopmentState(t *testing.T) {
	for _, outcome := range []VerifierOutcome{VerifierFailed, VerifierInconclusive} {
		t.Run(string(outcome), func(t *testing.T) {
			store, id := auxiliaryDurableFixture(t, 128)
			store.AuxiliaryServices.Authorize = func(state RuntimeState, job AuxiliaryJob) error {
				if state.Task.ID != id || job.Owner != id || job.Kind != "verifier" { return fmt.Errorf("outside fixture verifier authorization") }
				return nil
			}
			state, _ := auxiliaryTestLatest(t, store, id)
			lock, err := store.lock(id)
			if err != nil { t.Fatal(err) }
			// Fixed test artifact, not proof that production acceptance captured Git.
			snapshot, err := store.persistProtocolArtifactLocked(id, "verifier-snapshot", map[string]string{"fixed": "old accepted input fixture"})
			unlock(lock)
			if err != nil { t.Fatal(err) }
			_, report := verifierTestReport()
			report.Snapshot, report.Outcome = &snapshot, outcome
			report.Summary = "Fixed fixture review outcome: " + string(outcome)
			report.Coverage[0] = verifierTestRow("AC-one", outcome)
			c := verifierContext{TaskID: id, WorkItemID: report.WorkItemID, AttemptID: report.AttemptID, Snapshot: snapshot, Criteria: []string{"AC-one", "AC-two"}, Evidence: []string{strings.Repeat("a", 64)}}
			body, err := json.Marshal(c)
			if err != nil { t.Fatal(err) }
			ref, err := store.EnqueueAuxiliary(id, id, state.Protocol.Auxiliary.Sources[0].ID, "verifier", "verifier-v2 "+string(body)+"\nRead the fixed fixture only.")
			if err != nil { t.Fatal(err) }
			call := auxiliaryTestClaim(t, store, ref)
			auxiliaryTestChangeTitle(t, store, id)
			before, err := store.Load(id)
			if err != nil { t.Fatal(err) }
			job, err := store.ResolveAuxiliary(ref)
			if err != nil { t.Fatal(err) }
			body, err = json.Marshal(report)
			if err != nil { t.Fatal(err) }
			if err := ValidateVerifierOutput(job, string(body)); err != nil { t.Fatal(err) }
			auxiliaryTestObserve(t, store, call, "valid", string(body))
			if err := store.PublishAuxiliaryOutput(ref); err != nil { t.Fatal(err) }
			reopened := NewStore(store.Root)
			after, err := reopened.Load(id)
			if err != nil { t.Fatal(err) }
			saved, err := reopened.ResolveAuxiliary(ref)
			if err != nil { t.Fatal(err) }
			if saved.State != "completed" || saved.Verifier == nil || saved.Verifier.Outcome != outcome || saved.Verifier.Snapshot == nil || *saved.Verifier.Snapshot != snapshot || saved.RecoveryUsed || len(saved.Calls) != 1 {
				t.Fatal("negative review lost its old snapshot, failed publication or consumed model recovery")
			}
			var output AuxiliaryOutput
			if saved.Output == nil { t.Fatal("published report lacks immutable output") }
			if err := reopened.ReadExecutionArtifact(id, *saved.Output, &output); err != nil { t.Fatal(err) }
			if output.Text != string(body) { t.Fatal("saved report was rewritten") }
			if before.Planning != after.Planning || before.Delivery != after.Delivery || !equalJSON(before.WorkItems, after.WorkItems) || !equalJSON(before.Attempts, after.Attempts) || !equalJSON(before.Gates, after.Gates) || !equalJSON(before.WriteLease, after.WriteLease) {
				t.Fatal("negative report changed development, delivery, Gate or writer state")
			}
		})
	}
}
