package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// VerifierReportSchemaVersion versions the report-only verifier payload.
const VerifierReportSchemaVersion = 1

type VerifierOutcome string

const (
	VerifierPassed       VerifierOutcome = "passed"
	VerifierFailed       VerifierOutcome = "failed"
	VerifierInconclusive VerifierOutcome = "inconclusive"
)

// VerifierReport is immutable task-local evidence from the optional project
// verification stage. Its outcome is deliberately separate from ActorResult
// status: recording a failed report succeeds and must not block delivery.
type VerifierReport struct {
	SchemaVersion int             `json:"schema_version"`
	TaskID        TaskID          `json:"task_id"`
	WorkItemID    WorkItemID      `json:"work_item_id"`
	AttemptID     AttemptID       `json:"attempt_id"`
	Outcome       VerifierOutcome `json:"outcome"`
	Summary       string          `json:"summary"`
	Findings      []string        `json:"findings,omitempty"`
	RecordedAt    string          `json:"recorded_at"`
}

func (r VerifierReport) Validate() error {
	if r.SchemaVersion != VerifierReportSchemaVersion || r.TaskID == "" || r.WorkItemID == "" || r.AttemptID == "" || !validVerifierOutcome(r.Outcome) || strings.TrimSpace(r.Summary) == "" || strings.TrimSpace(r.RecordedAt) == "" {
		return fmt.Errorf("verifier report requires schema, Task, Work Item, Attempt, outcome, summary, and recorded time")
	}
	for index, finding := range r.Findings {
		if strings.TrimSpace(finding) == "" {
			return fmt.Errorf("verifier finding %d is empty", index)
		}
	}
	return nil
}

// PersistVerifierReport writes a durable report artifact. It has no Runtime
// state transition, Gate, or delivery side effect; callers record the normal
// Actor handoff separately after this artifact has been written.
func (s *Store) PersistVerifierReport(id TaskID, report VerifierReport) (ActorReference, error) {
	if report.TaskID != id { return ActorReference{}, fmt.Errorf("verifier report Task does not match runtime Task") }
	if err := report.Validate(); err != nil { return ActorReference{}, err }
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil { return ActorReference{}, fmt.Errorf("encode verifier report: %w", err) }
	content = append(content, '\n')
	relative := filepath.ToSlash(filepath.Join("verifier-reports", string(report.WorkItemID)+"-"+strings.ReplaceAll(report.RecordedAt, ":", "-")+".json"))
	if err := atomicWrite(s.path(id, filepath.FromSlash(relative)), content); err != nil { return ActorReference{}, err }
	sum := sha256.Sum256(content)
	return ActorReference{Kind: "verifier-report", Path: relative, SHA256: hex.EncodeToString(sum[:])}, nil
}

// CompleteVerifierReport turns a persisted optional report into a valid Actor
// response. Its accepted status means only that the report was recorded; the
// report outcome remains informational until a future policy promotes it.
func (s *Store) CompleteVerifierReport(id TaskID, request VerifierRequest, report VerifierReport) (VerifierResult, error) {
	if err := request.Validate(); err != nil { return VerifierResult{}, err }
	if report.RecordedAt == "" { report.RecordedAt = time.Now().UTC().Format(time.RFC3339) }
	if report.TaskID != request.TaskID || report.WorkItemID != request.WorkItemID || report.AttemptID != request.AttemptID {
		return VerifierResult{}, fmt.Errorf("verifier report does not match request")
	}
	reference, err := s.PersistVerifierReport(id, report)
	if err != nil { return VerifierResult{}, err }
	return VerifierResult{ActorResult: ActorResult{SchemaVersion: ActorContractSchemaVersion, RequestID: request.ID, Actor: ActorVerifier, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Status: ActorResultAccepted, Outputs: []ActorReference{reference}, Summary: "optional verifier report recorded", CompletedAt: report.RecordedAt}, Report: reference}, nil
}

func validVerifierOutcome(outcome VerifierOutcome) bool {
	switch outcome {
	case VerifierPassed, VerifierFailed, VerifierInconclusive:
		return true
	}
	return false
}
