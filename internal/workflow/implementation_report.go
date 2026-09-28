package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReportSection distinguishes an actual empty set from missing, unknown and
// unverified facts. Reasons are mandatory when evidence is unavailable.
type ReportSection struct {
	State string `json:"state"`
	Items []string `json:"items"`
	Reason string `json:"reason"`
}

type ImplementationReport struct {
	SchemaVersion int `json:"schema_version"`
	RequestID string `json:"request_id"`
	TaskID TaskID `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	SessionID string `json:"session_id"`
	Turn int `json:"turn"`
	Actor ActorKind `json:"actor"`
	InputSHA256 string `json:"input_sha256"`
	Coverage ReportSection `json:"coverage"`
	Changes []ActorReference `json:"changes"`
	Interfaces ReportSection `json:"interfaces"`
	SideEffects ReportSection `json:"side_effects"`
	TestEntrypoints ReportSection `json:"test_entrypoints"`
	Decisions ReportSection `json:"decisions"`
	Limitations ReportSection `json:"limitations"`
	Risks ReportSection `json:"risks"`
	Validation ReportSection `json:"validation"`
	References []ActorReference `json:"references"`
	History ReportSection `json:"history"`
}

// ImplementationReportOutputSchema is the structured final-answer contract for
// a Coder turn. Semantic identity and file digests are still checked by
// ValidateImplementationReport against the frozen request and workspace.
func ImplementationReportOutputSchema() map[string]any {
	object := func(properties map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	stringType := map[string]any{"type": "string"}
	integerType := map[string]any{"type": "integer"}
	strings := map[string]any{"type": "array", "items": stringType}
	section := object(map[string]any{"state": stringType, "items": strings, "reason": stringType}, "state", "items", "reason")
	reference := object(map[string]any{"kind": stringType, "path": stringType, "sha256": stringType}, "kind", "path", "sha256")
	references := map[string]any{"type": "array", "items": reference}
	report := object(map[string]any{
		"schema_version": integerType, "request_id": stringType, "task_id": stringType,
		"work_item_id": stringType, "attempt_id": stringType, "session_id": stringType,
		"turn": integerType, "actor": stringType, "input_sha256": stringType,
		"coverage": section, "changes": references, "interfaces": section,
		"side_effects": section, "test_entrypoints": section, "decisions": section,
		"limitations": section, "risks": section, "validation": section,
		"references": references, "history": section,
	}, "schema_version", "request_id", "task_id", "work_item_id", "attempt_id", "session_id", "turn", "actor", "input_sha256", "coverage", "changes", "interfaces", "side_effects", "test_entrypoints", "decisions", "limitations", "risks", "validation", "references", "history")
	return object(map[string]any{"report": report}, "report")
}

func (section ReportSection) validate(name string) error {
	if section.Items == nil { return fmt.Errorf("report %s must include an explicit items array", name) }
	switch section.State {
	case "known":
		if len(section.Items) == 0 { return fmt.Errorf("report %s known facts are empty", name) }
	case "empty":
		if len(section.Items) != 0 || strings.TrimSpace(section.Reason) == "" { return fmt.Errorf("report %s empty facts require a reason", name) }
	case "unknown", "unverified":
		if strings.TrimSpace(section.Reason) == "" { return fmt.Errorf("report %s requires an uncertainty reason", name) }
	default:
		return fmt.Errorf("report %s state is missing or unsupported", name)
	}
	for _, item := range section.Items { if strings.TrimSpace(item) == "" { return fmt.Errorf("report %s contains an empty fact", name) } }
	return nil
}

func ValidateImplementationReport(output string, request PreparedAgentRequest, root string) (ImplementationReport, error) {
	var envelope struct { Report *ImplementationReport `json:"report"` }
	if err := json.Unmarshal([]byte(output), &envelope); err != nil { return ImplementationReport{}, err }
	if envelope.Report == nil { return ImplementationReport{}, errors.New("implementation report is missing") }
	report := *envelope.Report
	origin := request
	if request.ReportOrigin != nil { origin = *request.ReportOrigin }
	if report.SchemaVersion != 1 || report.RequestID != ExecutionRequestID(origin) || report.TaskID != origin.TaskID || report.WorkItemID != origin.WorkItemID || report.AttemptID != origin.AttemptID || report.SessionID != origin.SessionID || report.Turn != origin.ExpectedSessionTurn || report.Actor != ActorCoder || origin.InputReference == nil || report.InputSHA256 != origin.InputReference.SHA256 {
		return report, errors.New("implementation report does not match its generation and input version")
	}
	sections := []struct { name string; value ReportSection }{
		{"coverage", report.Coverage}, {"interfaces", report.Interfaces}, {"side_effects", report.SideEffects}, {"test_entrypoints", report.TestEntrypoints}, {"decisions", report.Decisions}, {"limitations", report.Limitations}, {"risks", report.Risks}, {"validation", report.Validation}, {"history", report.History},
	}
	for _, section := range sections { if err := section.value.validate(section.name); err != nil { return report, err } }
	if report.Changes == nil || report.References == nil { return report, errors.New("report changes and references require explicit arrays") }
	seen := make(map[string]bool)
	for _, ref := range report.Changes {
		if strings.TrimSpace(ref.Path) == "" { return report, errors.New("report change path is missing") }
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(ref.Path)))
		if seen[path] { return report, errors.New("duplicate report change path") }
		seen[path] = true
		if ref.Kind == "deleted" {
			// Resolve the parent before checking absence, including symlinks.
			parent, err := confinedFile(root, filepath.Dir(filepath.FromSlash(ref.Path)))
			if err != nil { return report, err }
			if ref.SHA256 != "" { return report, errors.New("deleted content must not claim a current digest") }
			if _, err := os.Lstat(filepath.Join(parent, filepath.Base(ref.Path))); !errors.Is(err, os.ErrNotExist) { return report, errors.New("reported deletion is not absent") }
			continue
		}
		if ref.Kind != "content" || ref.SHA256 == "" { return report, errors.New("changed content requires a digest") }
		source := ReadInputSource(root, InputSource{Path: ref.Path, ExpectedSHA256: ref.SHA256, Required: true})
		if source.Status != "loaded" { return report, fmt.Errorf("reported change %s: %s", ref.Path, source.Reason) }
	}
	for _, ref := range report.References {
		if ref.Kind == "" || ref.SHA256 == "" { return report, errors.New("report reference requires kind and exact version") }
		source := ReadInputSource(root, InputSource{Path: ref.Path, ExpectedSHA256: ref.SHA256, Required: true})
		if source.Status != "loaded" { return report, fmt.Errorf("report reference %s: %s", ref.Path, source.Reason) }
	}
	return report, nil
}

// ValidateReportChanges checks completeness against bytes captured before the
// generation, so an empty changes list cannot hide modified or deleted files.
func ValidateReportChanges(report ImplementationReport, before *ValidationInputs, root string) error {
	if before == nil { return errors.New("generation content baseline is unknown") }
	after, err := CaptureValidationInputs(root, before.Scope, before.Bindings, before.Toolchain)
	if err != nil { return err }
	old := make(map[string]string)
	for _, ref := range before.Files { old[ref.Path] = ref.SHA256 }
	declared := make(map[string]ActorReference)
	for _, ref := range report.Changes { declared[filepath.ToSlash(filepath.Clean(filepath.FromSlash(ref.Path)))] = ref }
	for _, ref := range after.Files {
		if digest, found := old[ref.Path]; !found || digest != ref.SHA256 {
			change, found := declared[ref.Path]
			if !found || change.Kind != "content" || change.SHA256 != ref.SHA256 { return fmt.Errorf("report omits current changed content: %s", ref.Path) }
		}
		delete(old, ref.Path)
	}
	for path := range old { if declared[path].Kind != "deleted" { return fmt.Errorf("report omits deleted content: %s", path) } }
	return nil
}

// PrepareReportSupplement reserves one report-only turn by original request,
// never by retry count, model or the next Attempt. Dispatch remains managed.
func (s *Store) PrepareReportSupplement(id TaskID, original PreparedAgentRequest, reason string) (RuntimeState, error) {
	key := ExecutionRequestID(original)
	return s.UpdateWithEvent(id, Event{Type: "report.supplement.prepared", WorkItemID: original.WorkItemID, AttemptID: original.AttemptID, Detail: reason}, func(state *RuntimeState) error {
		current := state.Automation.PreparedRequest
		if current == nil || ExecutionRequestID(*current) != key || original.ReportOrigin != nil || current.DispatchedAt == "" || state.WriteLease == nil || state.WriteLease.AttemptID != original.AttemptID { return errors.New("report supplement must retain the completed generation and owning Attempt") }
		for _, used := range state.Automation.ReportSupplements { if used == key { return errors.New("report supplement already reserved; manual reconciliation required") } }
		state.Automation.ReportSupplements = append(state.Automation.ReportSupplements, key)
		origin := *current
		current.ReportOrigin = &origin
		current.ExpectedSessionTurn++
		current.DispatchedAt = ""
		current.InputReference = nil
		state.Automation.Cursor = AutomationCursor{Result: string(RunnerPrepared), Detail: "report-only supplement"}
		return nil
	})
}
