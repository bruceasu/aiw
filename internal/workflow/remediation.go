package workflow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	RemediationSchemaVersion    = 1
	MaxAutomaticRemediationRounds = 3
	remediationReportsDir       = "reports/remediation"
)

type RemediationStatus string

const (
	RemediationAnalyzed      RemediationStatus = "analyzed"
	RemediationAutoResolved  RemediationStatus = "auto-resolved"
	RemediationAwaitingHuman RemediationStatus = "awaiting-human"
	RemediationHumanConsumed RemediationStatus = "human-consumed"
	RemediationFailed        RemediationStatus = "failed"
)

type RemediationAction string

const (
	RemediationActionRetryWorkItem  RemediationAction = "retry-work-item"
	RemediationActionRepairSession  RemediationAction = "repair-session"
	RemediationActionRepairProjection RemediationAction = "repair-projection"
	RemediationActionResumeSupervisor RemediationAction = "resume-supervisor"
	RemediationActionStop            RemediationAction = "stop-supervisor"
	RemediationActionHumanReview     RemediationAction = "human-review"
)

type RemediationProblem struct {
	TaskID            TaskID     `json:"task_id"`
	WorkItemID        WorkItemID `json:"work_item_id,omitempty"`
	AttemptID         AttemptID  `json:"attempt_id,omitempty"`
	EventSequence     uint64     `json:"event_sequence"`
	Category          string     `json:"category"`
	Detail            string     `json:"detail"`
	EvidenceReference string     `json:"evidence_reference,omitempty"`
	FailureDigest     string     `json:"failure_digest"`
}

type RemediationDiagnosis struct {
	Category          string             `json:"category"`
	Summary           string             `json:"summary"`
	Confidence        float64            `json:"confidence"`
	Evidence          []string           `json:"evidence"`
	RecommendedAction RemediationAction  `json:"recommended_action"`
	Reason            string             `json:"reason"`
}

func (d RemediationDiagnosis) Validate(problem RemediationProblem, options []RemediationOption) error {
	if strings.TrimSpace(d.Category) == "" || strings.TrimSpace(d.Summary) == "" || strings.TrimSpace(d.Reason) == "" || d.Confidence < 0 || d.Confidence > 1 || !validRemediationAction(d.RecommendedAction) {
		return errors.New("remediation diagnosis is incomplete")
	}
	allowed := false
	for _, option := range options {
		if option.Action == d.RecommendedAction { allowed = true; break }
	}
	if !allowed { return errors.New("remediation diagnosis recommends an action outside the option set") }
	if problem.EvidenceReference != "" {
		for _, evidence := range d.Evidence { if evidence != problem.EvidenceReference { return errors.New("remediation diagnosis cites evidence outside the frozen problem") } }
	}
	return nil
}

func NewRemediationReport(problem RemediationProblem, diagnosis *RemediationDiagnosis, status RemediationStatus, round int, options []RemediationOption) RemediationReport {
	now := time.Now().UTC().Format(time.RFC3339)
	problemID := remediationProblemID(problem)
	report := RemediationReport{SchemaVersion: RemediationSchemaVersion, ProblemID: problemID, ReportPath: filepath.ToSlash(filepath.Join(remediationReportsDir, problemID+"-r"+strconv.Itoa(round)+".json")), Problem: problem, Diagnosis: diagnosis, Status: status, Round: round, Options: append([]RemediationOption(nil), options...), CreatedAt: now, UpdatedAt: now}
	report.ReportDigest = remediationReportDigest(report)
	return report
}

type RemediationOption struct {
	ID              string             `json:"id"`
	Action          RemediationAction  `json:"action"`
	Summary         string             `json:"summary"`
	Scope           string             `json:"scope"`
	Risk            string             `json:"risk"`
	ResourceImpact  string             `json:"resource_impact"`
	ExternalEffect  string             `json:"external_effect"`
	Rollback        string             `json:"rollback"`
	RequiresApproval bool              `json:"requires_approval"`
}

type RemediationAttempt struct {
	Action      RemediationAction `json:"action"`
	OptionID    string             `json:"option_id,omitempty"`
	Outcome     string             `json:"outcome"`
	Detail      string             `json:"detail"`
	RecordedAt  string             `json:"recorded_at"`
}

type RemediationReport struct {
	SchemaVersion      int                 `json:"schema_version"`
	ProblemID          string              `json:"problem_id"`
	ReportPath         string              `json:"report_path"`
	ReportDigest       string              `json:"report_digest"`
	Problem            RemediationProblem  `json:"problem"`
	Diagnosis          *RemediationDiagnosis `json:"diagnosis,omitempty"`
	Status             RemediationStatus   `json:"status"`
	Round              int                 `json:"round"`
	Options            []RemediationOption `json:"options"`
	Attempts           []RemediationAttempt `json:"attempts,omitempty"`
	HumanResponsePath  string              `json:"human_response_path,omitempty"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
}

type RemediationResponse struct {
	SchemaVersion int    `json:"schema_version"`
	ProblemID     string `json:"problem_id"`
	ReportDigest  string `json:"report_digest"`
	OptionID      string `json:"option_id"`
	Operator      string `json:"operator"`
	RiskConfirmed bool   `json:"risk_confirmed"`
	Note          string `json:"note,omitempty"`
	RecordedAt    string `json:"recorded_at"`
}

func BuildRemediationProblem(report FailureReport) (RemediationProblem, string, error) {
	if err := report.Validate(); err != nil { return RemediationProblem{}, "", err }
	problem := RemediationProblem{TaskID: report.TaskID, WorkItemID: report.WorkItemID, AttemptID: report.AttemptID, EventSequence: report.EventSequence, Category: report.Category, Detail: report.Detail, EvidenceReference: report.EvidenceReference}
	identity, err := json.Marshal(problem)
	if err != nil { return RemediationProblem{}, "", err }
	problem.FailureDigest = contentDigest(identity)
	identity, err = json.Marshal(problem)
	if err != nil { return RemediationProblem{}, "", err }
	return problem, contentDigest(identity), nil
}

func BuildRemediationReportForOutcome(state RuntimeState, attemptID AttemptID, outcome SupervisedOutcome, round int) (RemediationReport, error) {
	var attempt *Attempt
	for index := range state.Attempts {
		if state.Attempts[index].ID == attemptID { attempt = &state.Attempts[index]; break }
	}
	if attempt == nil { return RemediationReport{}, fmt.Errorf("unknown Attempt %s", attemptID) }
	failure := failureReportForOutcome(state, *attempt, outcome)
	problem, _, err := BuildRemediationProblem(failure)
	if err != nil { return RemediationReport{}, err }
	return NewRemediationReport(problem, nil, RemediationAnalyzed, round, DefaultRemediationOptions(problem, failure.Retryable)), nil
}

func (r RemediationReport) Validate() error {
	if r.SchemaVersion != RemediationSchemaVersion || strings.TrimSpace(r.ProblemID) == "" || strings.TrimSpace(r.ReportPath) == "" || strings.TrimSpace(r.ReportDigest) == "" || r.Status == "" || r.Round < 0 || len(r.Options) == 0 || r.CreatedAt == "" || r.UpdatedAt == "" {
		return errors.New("remediation report is incomplete")
	}
	if r.Problem.TaskID == "" || r.Problem.EventSequence == 0 || r.Problem.FailureDigest == "" { return errors.New("remediation report has no stable problem binding") }
	if got := remediationReportDigest(r); got != r.ReportDigest { return errors.New("remediation report digest does not match its content") }
	seen := map[string]bool{}
	for _, option := range r.Options {
		if option.ID == "" || seen[option.ID] || option.Action == "" || option.Summary == "" || option.Scope == "" || option.Risk == "" || option.ResourceImpact == "" || option.ExternalEffect == "" || option.Rollback == "" { return errors.New("remediation option is incomplete or duplicated") }
		if !validRemediationAction(option.Action) { return fmt.Errorf("unsupported remediation action %q", option.Action) }
		seen[option.ID] = true
	}
	return nil
}

func (r RemediationResponse) Validate(report RemediationReport) error {
	if r.SchemaVersion != RemediationSchemaVersion || r.ProblemID != report.ProblemID || r.ReportDigest != report.ReportDigest || strings.TrimSpace(r.OptionID) == "" || strings.TrimSpace(r.Operator) == "" || !r.RiskConfirmed || r.RecordedAt == "" { return errors.New("remediation response is incomplete or does not match the pending report") }
	for _, option := range report.Options { if option.ID == r.OptionID { if option.RequiresApproval && strings.TrimSpace(r.Note) == "" { return errors.New("approved remediation option requires an operator note") }; return nil } }
	return fmt.Errorf("remediation response selects an unknown option %q", r.OptionID)
}

func (s *Store) PersistRemediationReport(state RuntimeState, report RemediationReport) (ActorReference, error) {
	if report.Problem.TaskID != state.Task.ID { return ActorReference{}, errors.New("remediation report Task does not match runtime Task") }
	if report.CreatedAt == "" { report.CreatedAt = time.Now().UTC().Format(time.RFC3339) }
	report.UpdatedAt = report.CreatedAt
	if report.ProblemID == "" { report.ProblemID = remediationProblemID(report.Problem) }
	if report.ReportPath == "" { report.ReportPath = filepath.ToSlash(filepath.Join(remediationReportsDir, report.ProblemID+"-r"+strconv.Itoa(report.Round)+".json")) }
	report.ReportDigest = remediationReportDigest(report)
	if err := report.Validate(); err != nil { return ActorReference{}, err }
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil { return ActorReference{}, err }
	content = append(content, '\n')
	relative := report.ReportPath
	target := s.path(state.Task.ID, filepath.FromSlash(relative))
	previous, readErr := os.ReadFile(target)
	if readErr == nil {
		if !bytes.Equal(previous, content) { return ActorReference{}, errors.New("remediation report conflicts with preserved problem history") }
	} else if errors.Is(readErr, os.ErrNotExist) {
		if err := atomicWrite(target, content); err != nil { return ActorReference{}, err }
	} else { return ActorReference{}, readErr }
	digest := sha256.Sum256(content)
	return ActorReference{Kind: "remediation-report", Path: relative, SHA256: hex.EncodeToString(digest[:])}, nil
}

func (s *Store) AwaitRemediation(id TaskID, report RemediationReport) (RuntimeState, error) {
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, err }
	if report.Problem.TaskID != id { return RuntimeState{}, errors.New("remediation report Task does not match requested Task") }
	if report.Status != RemediationAwaitingHuman { return RuntimeState{}, errors.New("only an awaiting-human remediation can pause Workflow") }
	if report.HumanResponsePath == "" { report.HumanResponsePath = strings.TrimSuffix(report.ReportPath, ".json") + ".response.json" }
	if report.ReportPath == "" { report.ReportPath = filepath.ToSlash(filepath.Join(remediationReportsDir, report.ProblemID+"-r"+strconv.Itoa(report.Round)+".json")) }
	report.ReportDigest = remediationReportDigest(report)
	if _, err := s.PersistRemediationReport(state, report); err != nil { return RuntimeState{}, err }
	content, err := json.MarshalIndent(RemediationResponse{SchemaVersion: RemediationSchemaVersion, ProblemID: report.ProblemID, ReportDigest: report.ReportDigest, RecordedAt: "<RFC3339>", OptionID: "<choose-option-id>", Operator: "<operator>", RiskConfirmed: false, Note: "<required when the selected option requires approval>"}, "", "  ")
	if err != nil { return RuntimeState{}, err }
	content = append(content, '\n')
	if err := atomicWrite(s.path(id, filepath.FromSlash(report.HumanResponsePath)), content); err != nil { return RuntimeState{}, err }
	return s.UpdateWithEvent(id, Event{Type: "remediation.awaiting-human", Detail: remediationCursorKey(report)}, func(current *RuntimeState) error {
		current.Automation.Cursor = AutomationCursor{Result: "awaiting-human", Detail: remediationCursorKey(report), RecordedAt: time.Now().UTC().Format(time.RFC3339)}
		current.Automation.Supervisor.Result = "awaiting-human"
		current.Automation.Supervisor.Detail = report.ProblemID
		return nil
	})
}

func (s *Store) ReadPendingRemediation(id TaskID) (RuntimeState, RemediationReport, error) {
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, RemediationReport{}, err }
	if state.Automation.Cursor.Result != "awaiting-human" || state.Automation.Cursor.Detail == "" { return state, RemediationReport{}, errors.New("no human remediation response is pending") }
	problemID, roundText, ok := strings.Cut(state.Automation.Cursor.Detail, "@")
	if !ok || problemID == "" { return state, RemediationReport{}, errors.New("pending remediation cursor is malformed") }
	round, parseErr := strconv.Atoi(roundText)
	if parseErr != nil || round < 0 { return state, RemediationReport{}, errors.New("pending remediation cursor has an invalid round") }
	path := s.path(id, filepath.FromSlash(filepath.Join(remediationReportsDir, problemID+"-r"+strconv.Itoa(round)+".json")))
	report, err := readRemediationReport(path)
	return state, report, err
}

func (s *Store) ReadRemediationResponse(id TaskID, report RemediationReport) (RemediationResponse, error) {
	path := report.HumanResponsePath
	if path == "" { path = strings.TrimSuffix(report.ReportPath, ".json") + ".response.json" }
	content, err := os.ReadFile(s.path(id, filepath.FromSlash(path)))
	if err != nil { return RemediationResponse{}, err }
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var response RemediationResponse
	if err := decoder.Decode(&response); err != nil { return response, fmt.Errorf("decode remediation response: %w", err) }
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF { return response, errors.New("remediation response contains trailing data") }
	return response, response.Validate(report)
}

func (s *Store) ConsumeRemediationResponse(id TaskID, response RemediationResponse) (RuntimeState, error) {
	_, report, err := s.ReadPendingRemediation(id)
	if err != nil { return RuntimeState{}, err }
	if err := response.Validate(report); err != nil { return RuntimeState{}, err }
	return s.UpdateWithEvent(id, Event{Type: "remediation.response-consumed", Detail: response.OptionID}, func(current *RuntimeState) error {
		if current.Automation.Cursor.Result != "awaiting-human" || current.Automation.Cursor.Detail != remediationCursorKey(report) { return errors.New("remediation response is no longer pending") }
		current.Automation.Cursor = AutomationCursor{Result: "remediation-response-consumed", Detail: remediationCursorKey(report), RecordedAt: time.Now().UTC().Format(time.RFC3339)}
		current.Automation.Supervisor.Result = "remediation-response-consumed"
		current.Automation.Supervisor.Detail = response.OptionID
		return nil
	})
}

func (s *Store) RecordRemediationAutoRetry(id TaskID, problemID string, detail string) (RuntimeState, error) {
	if strings.TrimSpace(problemID) == "" { return RuntimeState{}, errors.New("remediation problem id is required") }
	return s.UpdateWithEvent(id, Event{Type: "remediation.auto-retry", Detail: problemID}, func(current *RuntimeState) error {
		if current.Automation.Supervisor.Detail != problemID { current.Automation.Supervisor.RemediationRounds = 0 }
		current.Automation.Supervisor.RemediationRounds++
		current.Automation.Cursor = AutomationCursor{Result: "remediation-auto-retry", Detail: detail, RecordedAt: time.Now().UTC().Format(time.RFC3339)}
		current.Automation.Supervisor.Result = "remediation-auto-retry"
		current.Automation.Supervisor.Detail = problemID
		return nil
	})
}

func readRemediationReport(path string) (RemediationReport, error) {
	content, err := os.ReadFile(path)
	if err != nil { return RemediationReport{}, err }
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var report RemediationReport
	if err := decoder.Decode(&report); err != nil { return report, fmt.Errorf("decode remediation report: %w", err) }
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF { return report, errors.New("remediation report contains trailing data") }
	return report, report.Validate()
}

func remediationProblemID(problem RemediationProblem) string {
	content, _ := json.Marshal(problem)
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:24]
}

func remediationCursorKey(report RemediationReport) string {
	return report.ProblemID + "@" + strconv.Itoa(report.Round)
}

func remediationReportDigest(report RemediationReport) string {
	copy := report
	copy.ReportDigest = ""
	content, _ := json.Marshal(copy)
	return contentDigest(content)
}

func validRemediationAction(action RemediationAction) bool {
	switch action {
	case RemediationActionRetryWorkItem, RemediationActionRepairSession, RemediationActionRepairProjection, RemediationActionResumeSupervisor, RemediationActionStop, RemediationActionHumanReview:
		return true
	default:
		return false
	}
}

func DefaultRemediationOptions(problem RemediationProblem, retryable bool) []RemediationOption {
	options := []RemediationOption{}
	if strings.Contains(strings.ToLower(problem.Category+" "+problem.Detail), "session") {
		options = append(options, RemediationOption{ID: "repair-session", Action: RemediationActionRepairSession, Summary: "按既有条件修复缺失 Session Attempt", Scope: "仅当前 Attempt、Session 绑定和 WorkItem", Risk: "如果 Session 并非真正缺失，Core 会拒绝修复", ResourceImpact: "不调用 AI；不创建第二个 Attempt", ExternalEffect: "不修改源代码或工作区内容", Rollback: "保留原 Attempt 历史，失败时转人工检查", RequiresApproval: false})
	}
	if strings.Contains(strings.ToLower(problem.Category+" "+problem.Detail), "projection") {
		options = append(options, RemediationOption{ID: "repair-projection", Action: RemediationActionRepairProjection, Summary: "重试已提交事件的投影修复", Scope: "仅当前已提交事件和记录的投影目标", Risk: "投影仍可能因文件或租约问题失败", ResourceImpact: "不调用 AI；不重放原始业务转换", ExternalEffect: "只更新受管状态投影", Rollback: "保留原事件和修复报告，失败后人工检查", RequiresApproval: false})
	}
	if retryable {
		options = append(options, RemediationOption{ID: "retry-work-item", Action: RemediationActionRetryWorkItem, Summary: "重试当前 WorkItem", Scope: "仅当前 WorkItem 和其既有 RetryPolicy", Risk: "可能再次消耗一次 Attempt", ResourceImpact: "使用现有重试额度；不增加额外预算", ExternalEffect: "可能再次启动一次受管 Agent 操作", Rollback: "失败后保留新的 Attempt 和报告，不回写旧证据", RequiresApproval: false})
	}
	options = append(options, RemediationOption{ID: "resume-supervisor", Action: RemediationActionResumeSupervisor, Summary: "恢复 Supervisor 继续处理", Scope: "当前 Task 的已记录 Workflow 状态", Risk: "如果问题仍未解决，会再次进入受限诊断", ResourceImpact: "可能消耗一轮 Supervisor 处理", ExternalEffect: "只执行已记录且通过前置校验的动作", Rollback: "Stop Supervisor 并保留当前报告", RequiresApproval: false})
	options = append(options, RemediationOption{ID: "human-review", Action: RemediationActionHumanReview, Summary: "停止并人工检查", Scope: "不修改当前执行事实", Risk: "Task 保持暂停", ResourceImpact: "不消耗 AI 调用", ExternalEffect: "不产生外部副作用", Rollback: "检查完成后重新提交答复", RequiresApproval: false})
	return options
}
