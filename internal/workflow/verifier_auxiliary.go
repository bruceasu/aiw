package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type VerifierCoverage struct {
	CriterionID string `json:"criterion_id"`
	Applicable *bool `json:"applicable"`
	Outcome VerifierOutcome `json:"outcome"`
	Basis string `json:"basis"`
	Evidence []string `json:"evidence"`
	Gaps []string `json:"gaps"`
}

func verifierCoverageOutcome(rows []VerifierCoverage) (VerifierOutcome, error) {
	if len(rows) == 0 { return "", errors.New("Verifier empty coverage is not a report") }
	seen := map[string]bool{}
	failed, incomplete, applicable := false, false, 0
	for _, row := range rows {
		if row.CriterionID == "" || seen[row.CriterionID] || row.Applicable == nil || strings.TrimSpace(row.Basis) == "" || !validVerifierOutcome(row.Outcome) { return "", errors.New("Verifier coverage identity, applicability or basis is invalid") }
		seen[row.CriterionID] = true
		for _, gap := range row.Gaps { if strings.TrimSpace(gap) == "" { return "", errors.New("Verifier gap cannot be empty") } }
		for _, evidence := range row.Evidence { if strings.TrimSpace(evidence) == "" { return "", errors.New("Verifier evidence cannot be empty") } }
		if !*row.Applicable {
			if row.Outcome != VerifierInconclusive || len(row.Evidence) == 0 || len(row.Gaps) != 0 { return "", errors.New("non-applicability needs explicit evidence and no unresolved gap") }
			continue
		}
		applicable++
		switch row.Outcome {
		case VerifierFailed:
			if len(row.Evidence) == 0 { return "", errors.New("failed criterion needs evidence") }
			failed = true
		case VerifierPassed:
			if len(row.Evidence) == 0 || len(row.Gaps) != 0 { return "", errors.New("passed criterion needs evidence and no gaps") }
		case VerifierInconclusive:
			if len(row.Gaps) == 0 { return "", errors.New("inconclusive criterion needs an explicit gap") }
			incomplete = true
		}
	}
	if failed { return VerifierFailed, nil }
	if incomplete || applicable == 0 { return VerifierInconclusive, nil }
	return VerifierPassed, nil
}

type verifierContext struct {
	TaskID TaskID `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	Snapshot ActorReference `json:"snapshot"`
	Criteria []string `json:"criteria"`
	Evidence []string `json:"evidence"`
}

const verifierInstruction = `Review this fixed accepted snapshot as an independent read-only Verifier. The Git baseline is explicit; do not assume the diff began with this Work Item. Use original requirements and acceptance text, the actual diff, and controlled compile/test evidence. Do not run tools, tests, or repairs. Return one JSON object without Markdown: schema_version=2, task_id, work_item_id, attempt_id, snapshot (copy the exact reference), outcome, summary, recorded_at (RFC3339), and coverage. Cover every listed criterion exactly once. Each coverage row has criterion_id, applicable (boolean), outcome, basis, evidence (an array of exact hashes from the allowed evidence list), and gaps (an array of strings). Cite concrete locations and explain what the evidence proves in basis. A failed applicable item needs evidence; it makes the report failed even if other items have gaps. If no item fails but any applicable item lacks proof, return inconclusive. Return passed only when at least one item is applicable and every applicable item has evidence, passes, and has no gaps. For a non-applicable item use outcome=inconclusive, explain why, cite evidence, and use gaps=[]. An empty report or empty coverage is invalid. A valid failed or inconclusive report is a successful review, never a request for regeneration. Report recording does not accept work, open Gates, or block delivery.`

func verifierJobContext(job AuxiliaryJob) (verifierContext, error) {
	var c verifierContext
	line, _, _ := strings.Cut(job.Prompt, "\n")
	if !strings.HasPrefix(line, "verifier-v2 ") { return c, errors.New("Verifier lacks a fixed context") }
	if err := strictJSON([]byte(strings.TrimPrefix(line, "verifier-v2 ")), &c); err != nil { return c, err }
	if c.TaskID != job.Owner || c.WorkItemID == "" || c.AttemptID == "" || c.Snapshot.Kind != "verifier-snapshot" || !validProtocolReference(c.Snapshot) || len(c.Criteria) == 0 { return c, errors.New("Verifier fixed identity is invalid") }
	return c, nil
}

func parseVerifierOutput(job AuxiliaryJob, text string) (VerifierReport, error) {
	var report VerifierReport
	c, err := verifierJobContext(job)
	if err != nil { return report, err }
	if len(text) > AuxiliaryOutputBytes { return report, errors.New("Verifier output exceeds byte limit") }
	if err := strictJSON([]byte(text), &report); err != nil { return report, err }
	if report.SchemaVersion != 2 || report.TaskID != c.TaskID || report.WorkItemID != c.WorkItemID || report.AttemptID != c.AttemptID || report.Snapshot == nil || *report.Snapshot != c.Snapshot { return report, errors.New("Verifier report does not match its reviewed version") }
	if err := report.Validate(); err != nil { return report, err }
	criteria, citations := map[string]bool{}, map[string]bool{}
	for _, id := range c.Criteria { criteria[id] = true }
	for _, hash := range c.Evidence { citations[hash] = true }
	for _, row := range report.Coverage {
		if !criteria[row.CriterionID] { return report, errors.New("Verifier has unknown or repeated criterion") }
		delete(criteria, row.CriterionID)
		for _, hash := range row.Evidence { if !citations[hash] { return report, errors.New("Verifier cites evidence outside its frozen input") } }
	}
	if len(criteria) != 0 { return report, errors.New("Verifier omitted acceptance coverage") }
	return report, nil
}

// Called before journaling a terminal response. Negative review conclusions
// return nil and consume no recovery; malformed or empty output is distinct.
func ValidateVerifierOutput(job AuxiliaryJob, text string) error {
	if job.Kind != "verifier" { return nil }
	_, err := parseVerifierOutput(job, text)
	return err
}

// Accepted sources are the durable outbox. Only the first source containing
// this exact acceptance is used, so restarts and later progress reuse the same
// source-owned recovery account. Historical work without a snapshot is never
// reconstructed from today's workspace or promoted to a passed review.
func (s *Store) RegisterTaskVerifier(id TaskID) error {
	state, err := s.Load(id)
	if err != nil { return err }
	if state.Protocol == nil || state.Protocol.Auxiliary == nil { return nil }
	var failures []error
	for _, item := range state.Protocol.Items {
		if item.Phase != PhaseAccepted || item.Accepted == nil { continue }
		if item.VerifierSnapshot == nil {
			if item.VerifierGap != "" { failures = append(failures, fmt.Errorf("Verifier %s: %s", item.WorkItemID, item.VerifierGap)) } else { failures = append(failures, fmt.Errorf("Verifier %s unavailable: historical acceptance has no fixed snapshot", item.WorkItemID)) }
			continue
		}
		already := false
		for _, job := range state.Protocol.Auxiliary.Jobs {
			if job.Kind != "verifier" { continue }
			c, err := verifierJobContext(job)
			if err != nil { return err }
			if c.Snapshot == *item.VerifierSnapshot { already = true; break }
		}
		if already { continue }
		var v VerifierSnapshot
		if err := s.ReadExecutionArtifact(id, *item.VerifierSnapshot, &v); err != nil { failures = append(failures, err); continue }
		if v.Version != 1 || v.TaskID != id || v.WorkItemID != item.WorkItemID || v.AttemptID != item.AttemptID { failures = append(failures, errors.New("Verifier snapshot identity mismatch")); continue }
		c := verifierContext{TaskID: id, WorkItemID: item.WorkItemID, AttemptID: item.AttemptID, Snapshot: *item.VerifierSnapshot}
		for _, criterion := range v.Criteria { c.Criteria = append(c.Criteria, criterion.ID) }
		for _, source := range v.Sources { c.Evidence = append(c.Evidence, source.SHA256) }
		c.Evidence = append(c.Evidence, v.Diff.SHA256)
		for _, source := range v.Untracked { c.Evidence = append(c.Evidence, source.SHA256) }
		for _, artifact := range v.Artifacts { c.Evidence = append(c.Evidence, artifact.Reference.SHA256) }
		contextJSON, _ := json.Marshal(c)
		body, _ := json.Marshal(v)
		instruction := "verifier-v2 "+string(contextJSON)+"\n"+verifierInstruction+"\n[Fixed accepted snapshot]\n"+string(body)
		found := false
		for _, source := range state.Protocol.Auxiliary.Sources {
			for _, accepted := range source.Items {
				if accepted.ID != item.WorkItemID || accepted.Accepted == nil || *accepted.Accepted != *item.Accepted { continue }
				_, err := s.EnqueueAuxiliary(id, id, source.ID, "verifier", instruction)
				if err != nil { failures = append(failures, err) }
				found = true
				break
			}
			if found { break }
		}
		if !found { failures = append(failures, errors.New("Verifier accepted source has not been committed")) }
	}
	return errors.Join(failures...)
}
