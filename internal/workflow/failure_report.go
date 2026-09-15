package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const FailureReportSchemaVersion = 1

const latestFailureReportPath = "reports/latest-failure.md"

// FailureReport is an immutable, event-scoped account of an unresolved
// workflow failure. Runtime state remains authoritative; reports are a
// read-oriented projection for operators and never participate in scheduling.
type FailureReport struct {
	SchemaVersion     int        `json:"schema_version"`
	TaskID            TaskID     `json:"task_id"`
	EventSequence     uint64     `json:"event_sequence"`
	WorkItemID        WorkItemID `json:"work_item_id,omitempty"`
	AttemptID         AttemptID  `json:"attempt_id,omitempty"`
	Category          string     `json:"category"`
	Detail            string     `json:"detail"`
	EvidenceReference string     `json:"evidence_reference,omitempty"`
	Retryable         bool       `json:"retryable"`
	Owner             string     `json:"owner"`
	RecommendedAction string     `json:"recommended_action"`
	RecordedAt        string     `json:"recorded_at"`
}

func (r FailureReport) Validate() error {
	if r.SchemaVersion != FailureReportSchemaVersion || r.TaskID == "" || r.EventSequence == 0 || strings.TrimSpace(r.Category) == "" || strings.TrimSpace(r.Detail) == "" || strings.TrimSpace(r.Owner) == "" || strings.TrimSpace(r.RecommendedAction) == "" || strings.TrimSpace(r.RecordedAt) == "" {
		return fmt.Errorf("failure report requires schema, Task, event, category, detail, owner, action, and recorded time")
	}
	return nil
}

// PersistFailureReport writes one immutable event record and atomically
// replaces the task's human-readable current failure projection.
func (s *Store) PersistFailureReport(state RuntimeState, report FailureReport) (ActorReference, error) {
	if report.TaskID != state.Task.ID {
		return ActorReference{}, fmt.Errorf("failure report Task does not match runtime Task")
	}
	if report.RecordedAt == "" {
		report.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := report.Validate(); err != nil {
		return ActorReference{}, err
	}
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ActorReference{}, fmt.Errorf("encode failure report: %w", err)
	}
	content = append(content, '\n')
	relative := filepath.ToSlash(filepath.Join("reports", "attempts", fmt.Sprintf("%06d-%s.json", report.EventSequence, strings.ReplaceAll(report.RecordedAt, ":", "-"))))
	if err := atomicWrite(s.path(state.Task.ID, filepath.FromSlash(relative)), content); err != nil {
		return ActorReference{}, err
	}
	if err := atomicWrite(s.path(state.Task.ID, filepath.FromSlash(latestFailureReportPath)), []byte(renderLatestFailureReport(state, report, relative))); err != nil {
		return ActorReference{}, err
	}
	sum := sha256.Sum256(content)
	return ActorReference{Kind: "failure-report", Path: relative, SHA256: hex.EncodeToString(sum[:])}, nil
}

// ReadLatestFailureReport reads the current failure projection without loading
// or changing RuntimeState. Callers can therefore present diagnostics safely
// while a supervisor is paused or another process owns the Task lease.
func (s *Store) ReadLatestFailureReport(id TaskID) ([]byte, error) {
	if err := validateTaskID(id); err != nil {
		return nil, err
	}
	return os.ReadFile(s.path(id, filepath.FromSlash(latestFailureReportPath)))
}

func renderLatestFailureReport(state RuntimeState, newest FailureReport, recordPath string) string {
	var content strings.Builder
	fmt.Fprintf(&content, "# Latest unresolved failure: %s\n\n", state.Task.ID)
	fmt.Fprintf(&content, "Updated: %s  \n", newest.RecordedAt)
	fmt.Fprintf(&content, "Latest record: [%s](%s)\n\n", recordPath, recordPath)
	fmt.Fprintf(&content, "## Latest event\n\n- Category: `%s`\n- Detail: %s\n- Retryable: `%t`\n- Owner: `%s`\n- Next action: %s\n", newest.Category, newest.Detail, newest.Retryable, newest.Owner, newest.RecommendedAction)
	if newest.WorkItemID != "" {
		fmt.Fprintf(&content, "- Work item: `%s`\n", newest.WorkItemID)
	}
	if newest.AttemptID != "" {
		fmt.Fprintf(&content, "- Attempt: `%s`\n", newest.AttemptID)
	}
	if newest.EvidenceReference != "" {
		fmt.Fprintf(&content, "- Evidence: `%s`\n", newest.EvidenceReference)
	}

	if len(state.Gates) > 0 {
		content.WriteString("\n## Open gates\n")
		for _, gate := range state.Gates {
			if gate.State != GateOpen {
				continue
			}
			fmt.Fprintf(&content, "\n- `%s` (%s): %s", gate.ID, gate.Kind, gate.Reason)
			if gate.WorkItemID != "" {
				fmt.Fprintf(&content, " -- work item `%s`", gate.WorkItemID)
			}
			content.WriteByte('\n')
		}
	}
	return content.String()
}

func failureReportForOutcome(state RuntimeState, attempt Attempt, outcome SupervisedOutcome) FailureReport {
	retryable := outcome.Kind == SupervisedOutcomeNoProgress && itemCanRetry(state, attempt.WorkItemID)
	return FailureReport{
		SchemaVersion: FailureReportSchemaVersion, TaskID: state.Task.ID, EventSequence: state.LastEventSequence,
		WorkItemID: attempt.WorkItemID, AttemptID: attempt.ID, Category: string(outcome.Kind), Detail: outcome.Detail,
		EvidenceReference: outcome.EvidenceReference, Retryable: retryable,
		Owner: failureOwner(outcome.Kind, retryable),
		RecommendedAction: recommendedFailureAction(outcome.Kind, outcome.BlockedCategory, retryable),
	}
}

func failureReportForCompiler(state RuntimeState, workItemID WorkItemID, diagnostics ActorReference) FailureReport {
	retryable := false
	for _, item := range state.WorkItems {
		if item.ID == workItemID {
			retryable = item.CompileFailureCount < MaxConsecutiveCompileFailures
			break
		}
	}
	return FailureReport{
		SchemaVersion: FailureReportSchemaVersion, TaskID: state.Task.ID, EventSequence: state.LastEventSequence,
		WorkItemID: workItemID, Category: "compiler", Detail: "compile-only validation failed",
		EvidenceReference: diagnostics.Path, Retryable: retryable,
		Owner: failureOwner(SupervisedOutcomeNoProgress, retryable),
		RecommendedAction: "inspect compiler diagnostics and return the same Work Item to Coder",
	}
}

func failureOwner(kind SupervisedOutcomeKind, retryable bool) string {
	if retryable || kind == SupervisedOutcomeCompleted {
		return "supervisor"
	}
	return "operator"
}

func itemCanRetry(state RuntimeState, workItemID WorkItemID) bool {
	for _, item := range state.WorkItems {
		if item.ID == workItemID {
			return item.State == WorkItemReady && item.NoProgressCount < item.RetryPolicy.MaxAttempts
		}
	}
	return false
}

func recommendedFailureAction(kind SupervisedOutcomeKind, category BlockedOutcomeCategory, retryable bool) string {
	if retryable {
		return "review the recorded evidence, then let the supervisor retry this Work Item"
	}
	if kind == SupervisedOutcomeBlocked {
		switch category {
		case BlockedOutcomeWorkspaceAccess:
			return "repair the recorded worktree binding, then resolve the workspace-access Gate"
		case BlockedOutcomeDependency:
			return "prepare the required local dependency without downloading during supervise, then resolve the Gate"
		case BlockedOutcomeAuthorization:
			return "obtain the required authorization, then resolve the Gate"
		}
	}
	return "review the evidence and resolve the reported Gate or reopen the Work Item deliberately"
}
