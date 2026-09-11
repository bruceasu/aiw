package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// FocusedTestSelectionContext is the read-only information made available to
// a test-agent. It deliberately contains no argv, directory, timeout, or
// authorization data that could grant execution authority.
type FocusedTestSelectionContext struct {
	TaskID        workflow.TaskID
	WorkItemID    workflow.WorkItemID
	WorkItemTitle string
	AttemptID     workflow.AttemptID
	ChangedFiles  []string
	PriorEvidence []workflow.Evidence
}

// BuildFocusedTestSelectionHandoff renders the constrained test-agent input.
// The response format permits only a Plan-declared check ID and references to
// the explicitly provided context.
func BuildFocusedTestSelectionHandoff(plan workflow.VerificationPlan, context FocusedTestSelectionContext) (string, error) {
	if err := validateFocusedTestSelectionContext(plan, context); err != nil {
		return "", err
	}
	digest, err := plan.Digest()
	if err != nil {
		return "", fmt.Errorf("digest verification plan: %w", err)
	}

	checkIDs := make([]string, 0, len(plan.Checks))
	for _, check := range plan.Checks {
		checkIDs = append(checkIDs, check.CheckID)
	}
	sort.Strings(checkIDs)

	var handoff strings.Builder
	fmt.Fprintln(&handoff, "Role: test-agent (read only)")
	fmt.Fprintln(&handoff, "")
	fmt.Fprintf(&handoff, "Task: %s\nWork Item: %s\nAttempt: %s\nPlan schema: %d\nPlan digest: %s\n", context.TaskID, context.WorkItemID, context.AttemptID, plan.SchemaVersion, digest)
	if title := strings.TrimSpace(context.WorkItemTitle); title != "" {
		fmt.Fprintf(&handoff, "Work Item context: %s\n", title)
	}
	fmt.Fprintln(&handoff, "")
	fmt.Fprintln(&handoff, "Choose exactly one allowed check ID:")
	for _, checkID := range checkIDs {
		fmt.Fprintf(&handoff, "- %s\n", checkID)
	}
	fmt.Fprintln(&handoff, "")
	if len(context.ChangedFiles) > 0 {
		fmt.Fprintln(&handoff, "Changed files:")
		for _, path := range context.ChangedFiles {
			if path = strings.TrimSpace(path); path != "" {
				fmt.Fprintf(&handoff, "- %s\n", path)
			}
		}
		fmt.Fprintln(&handoff, "")
	}
	if len(context.PriorEvidence) > 0 {
		fmt.Fprintln(&handoff, "Prior Evidence:")
		for _, evidence := range context.PriorEvidence {
			if strings.TrimSpace(string(evidence.ID)) != "" {
				fmt.Fprintf(&handoff, "- evidence:%s (kind=%s; state=%s; reference=%s)\n", evidence.ID, evidence.Kind, evidence.State, evidence.Reference)
			}
		}
		fmt.Fprintln(&handoff, "")
	}
	fmt.Fprintln(&handoff, "Context references you may cite:")
	for _, reference := range focusedTestContextReferences(context) {
		fmt.Fprintf(&handoff, "- %s\n", reference)
	}
	fmt.Fprintln(&handoff, "")
	fmt.Fprintln(&handoff, "Return JSON only with schema_version, plan_digest, check_id, rationale, and evidence_references.")
	fmt.Fprintln(&handoff, "Do not return a command, argv, working directory, timeout, environment, profile, network policy, or authorization decision.")
	return handoff.String(), nil
}

// ValidateFocusedTestSelection checks a test-agent response against both the
// immutable Plan and the context references exposed in its own handoff.
func ValidateFocusedTestSelection(plan workflow.VerificationPlan, context FocusedTestSelectionContext, selection workflow.VerificationPlanSelection) error {
	if err := validateFocusedTestSelectionContext(plan, context); err != nil {
		return err
	}
	if err := plan.ValidateSelection(selection); err != nil {
		return err
	}
	allowed := make(map[string]struct{})
	for _, reference := range focusedTestContextReferences(context) {
		allowed[reference] = struct{}{}
	}
	for _, reference := range selection.EvidenceReferences {
		if _, ok := allowed[reference]; !ok {
			return fmt.Errorf("verification plan selection reference %q was not provided to the test-agent", reference)
		}
	}
	return nil
}

// WriteFocusedTestSelection persists the validated structured choice in the
// Task runtime area. It never writes high-churn agent output into OpenSpec.
func WriteFocusedTestSelection(taskID string, selection workflow.VerificationPlanSelection) error {
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("task ID is required")
	}
	path := taskx.VerificationSelectionPath(taskID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create focused-test selection artifact directory: %w", err)
	}
	data, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		return fmt.Errorf("encode focused-test selection artifact: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write focused-test selection artifact: %w", err)
	}
	return nil
}

func validateFocusedTestSelectionContext(plan workflow.VerificationPlan, context FocusedTestSelectionContext) error {
	if err := plan.Validate(); err != nil {
		return err
	}
	if plan.TaskID != context.TaskID {
		return fmt.Errorf("test-agent selection context task ID does not match the active plan")
	}
	if strings.TrimSpace(string(context.WorkItemID)) == "" || strings.TrimSpace(string(context.AttemptID)) == "" {
		return fmt.Errorf("test-agent selection context requires work item and attempt IDs")
	}
	return nil
}

func focusedTestContextReferences(context FocusedTestSelectionContext) []string {
	references := []string{"work-item:" + string(context.WorkItemID)}
	for _, path := range context.ChangedFiles {
		if path = strings.TrimSpace(path); path != "" {
			references = append(references, "changed-file:"+path)
		}
	}
	for _, evidence := range context.PriorEvidence {
		if strings.TrimSpace(string(evidence.ID)) != "" {
			references = append(references, "evidence:"+string(evidence.ID))
		}
	}
	sort.Strings(references)
	return references
}
