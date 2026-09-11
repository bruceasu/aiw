package workflow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	VerificationPlanSchemaVersion = 1
	VerificationPlanSelectionSchemaVersion = 1
	FocusedTestProfile            = "focused-test"
	NetworkPolicyDeny             = "deny"
	MaxFocusedTestTimeoutSeconds  = 3600
)

var (
	verificationCheckIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	environmentNamePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// VerificationPlan is the task-local, human-authored contract for controlled
// focused validation. Validation and parsing are deliberately separate so an
// adapter can reject an invalid plan before it reaches an Agent or a process.
type VerificationPlan struct {
	SchemaVersion int                 `json:"schema_version"`
	TaskID        TaskID              `json:"task_id"`
	Checks        []VerificationCheck `json:"checks"`
}

// VerificationCheck is one immutable candidate command. Argv is never a shell
// command string: the controlled runner receives only its individual entries.
type VerificationCheck struct {
	CheckID             string   `json:"check_id"`
	Argv                []string `json:"argv"`
	WorkingDirectory    string   `json:"working_directory"`
	TimeoutSeconds      int      `json:"timeout_seconds"`
	ExpectedExitCode    int      `json:"expected_exit_code"`
	EvidenceDestination string   `json:"evidence_destination"`
	NetworkPolicy       string   `json:"network"`
	Profile             string   `json:"profile"`
	AllowedEnvironment  []string `json:"allowed_environment,omitempty"`
}

// VerificationPlanSelection is the auditable, read-only test-agent handoff.
// It intentionally contains no command, directory, timeout, or authority.
type VerificationPlanSelection struct {
	SchemaVersion      int      `json:"schema_version"`
	PlanDigest         string   `json:"plan_digest"`
	CheckID            string   `json:"check_id"`
	Rationale          string   `json:"rationale"`
	EvidenceReferences []string `json:"evidence_references,omitempty"`
}

// ParseVerificationPlanSelection decodes a test-agent's structured response
// and binds it to one validated, immutable Verification Plan. It accepts no
// command fields: the later runner must resolve all execution details from the
// Plan using the selected check ID.
func ParseVerificationPlanSelection(data []byte, plan VerificationPlan) (VerificationPlanSelection, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var selection VerificationPlanSelection
	if err := decoder.Decode(&selection); err != nil {
		return VerificationPlanSelection{}, fmt.Errorf("decode verification plan selection: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return VerificationPlanSelection{}, fmt.Errorf("decode verification plan selection: multiple JSON values")
		}
		return VerificationPlanSelection{}, fmt.Errorf("decode verification plan selection: %w", err)
	}
	if err := plan.ValidateSelection(selection); err != nil {
		return VerificationPlanSelection{}, err
	}
	return selection, nil
}

// ValidateSelection verifies that a structured test-agent choice is covered
// by this exact Plan. Context-reference ownership is checked by the command
// adapter that constructed the read-only handoff.
func (p VerificationPlan) ValidateSelection(selection VerificationPlanSelection) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("validate verification plan selection: %w", err)
	}
	if selection.SchemaVersion != VerificationPlanSelectionSchemaVersion {
		return fmt.Errorf("verification plan selection schema_version must be %d", VerificationPlanSelectionSchemaVersion)
	}
	digest, err := p.Digest()
	if err != nil {
		return fmt.Errorf("digest verification plan: %w", err)
	}
	if selection.PlanDigest != digest {
		return fmt.Errorf("verification plan selection digest does not match the active plan")
	}
	if !verificationCheckIDPattern.MatchString(selection.CheckID) {
		return fmt.Errorf("verification plan selection has invalid check_id %q", selection.CheckID)
	}
	found := false
	for _, check := range p.Checks {
		if check.CheckID == selection.CheckID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("verification plan selection check_id %q is not declared by the active plan", selection.CheckID)
	}
	rationale := strings.TrimSpace(selection.Rationale)
	if rationale == "" || len(rationale) > 1000 {
		return fmt.Errorf("verification plan selection rationale must contain 1 to 1000 characters")
	}
	if len(selection.EvidenceReferences) == 0 || len(selection.EvidenceReferences) > 16 {
		return fmt.Errorf("verification plan selection must contain 1 to 16 evidence_references")
	}
	seenReferences := make(map[string]struct{}, len(selection.EvidenceReferences))
	for index, reference := range selection.EvidenceReferences {
		if strings.TrimSpace(reference) == "" || len(reference) > 512 {
			return fmt.Errorf("verification plan selection evidence_references[%d] must contain 1 to 512 characters", index)
		}
		if _, exists := seenReferences[reference]; exists {
			return fmt.Errorf("verification plan selection has duplicate evidence reference %q", reference)
		}
		seenReferences[reference] = struct{}{}
	}
	return nil
}

// BoundedOutputReference identifies output captured outside Workflow Core.
// The bounded size and storage policy are enforced by the later runner seam.
type BoundedOutputReference struct {
	Path          string `json:"path"`
	ByteCount     int    `json:"byte_count"`
	Truncated     bool   `json:"truncated"`
	SHA256        string `json:"sha256,omitempty"`
}

// FocusedTestResult is the durable result artifact written by the controlled
// runner. Evidence references this artifact rather than embedding process
// output in RuntimeState.
type FocusedTestResult struct {
	SchemaVersion      int                    `json:"schema_version"`
	PlanDigest         string                 `json:"plan_digest"`
	CheckID            string                 `json:"check_id"`
	Argv               []string               `json:"argv"`
	WorkingDirectory   string                 `json:"working_directory"`
	StartedAt          string                 `json:"started_at"`
	FinishedAt         string                 `json:"finished_at"`
	ExitCode           *int                   `json:"exit_code,omitempty"`
	TimedOut           bool                   `json:"timed_out"`
	RunnerError        string                 `json:"runner_error,omitempty"`
	Output             BoundedOutputReference `json:"output"`
}

// ParseVerificationPlan decodes a task-local plan without accepting unknown
// fields. Callers must use this before giving plan content to an Agent or a
// controlled runner.
func ParseVerificationPlan(data []byte) (VerificationPlan, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var plan VerificationPlan
	if err := decoder.Decode(&plan); err != nil {
		return VerificationPlan{}, fmt.Errorf("decode verification plan: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return VerificationPlan{}, fmt.Errorf("decode verification plan: multiple JSON values")
		}
		return VerificationPlan{}, fmt.Errorf("decode verification plan: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return VerificationPlan{}, err
	}
	return plan, nil
}

// Validate verifies the fixed, task-bound constraints required before a plan
// can be authorized or executed. It never expands paths or interprets argv.
func (p VerificationPlan) Validate() error {
	if p.SchemaVersion != VerificationPlanSchemaVersion {
		return fmt.Errorf("verification plan schema_version must be %d", VerificationPlanSchemaVersion)
	}
	if strings.TrimSpace(string(p.TaskID)) == "" {
		return fmt.Errorf("verification plan task_id is required")
	}
	if len(p.Checks) == 0 {
		return fmt.Errorf("verification plan must contain at least one check")
	}

	seenCheckIDs := make(map[string]struct{}, len(p.Checks))
	for index, check := range p.Checks {
		prefix := fmt.Sprintf("verification plan check %d", index)
		if !verificationCheckIDPattern.MatchString(check.CheckID) {
			return fmt.Errorf("%s has invalid check_id %q", prefix, check.CheckID)
		}
		if _, exists := seenCheckIDs[check.CheckID]; exists {
			return fmt.Errorf("verification plan has duplicate check_id %q", check.CheckID)
		}
		seenCheckIDs[check.CheckID] = struct{}{}
		if err := validateFocusedArgv(check.Argv); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
		if err := validateWorktreeRelativeDirectory(check.WorkingDirectory); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
		if check.TimeoutSeconds < 1 || check.TimeoutSeconds > MaxFocusedTestTimeoutSeconds {
			return fmt.Errorf("%s timeout_seconds must be between 1 and %d", prefix, MaxFocusedTestTimeoutSeconds)
		}
		if check.Profile != FocusedTestProfile {
			return fmt.Errorf("%s profile must be %q", prefix, FocusedTestProfile)
		}
		if check.NetworkPolicy != NetworkPolicyDeny {
			return fmt.Errorf("%s network must be %q", prefix, NetworkPolicyDeny)
		}
		if strings.TrimSpace(check.EvidenceDestination) == "" {
			return fmt.Errorf("%s evidence_destination is required", prefix)
		}
		if check.ExpectedExitCode < 0 {
			return fmt.Errorf("%s expected_exit_code must not be negative", prefix)
		}
		if err := validateAllowedEnvironment(check.AllowedEnvironment); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
	}
	return nil
}

func validateFocusedArgv(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("argv is required")
	}
	for index, arg := range argv {
		if arg == "" || strings.TrimSpace(arg) == "" {
			return fmt.Errorf("argv[%d] must not be empty", index)
		}
		if strings.ContainsAny(arg, "\x00\r\n|;&<>`$*?[]") || strings.Contains(arg, "$(") || strings.Contains(arg, "${") {
			return fmt.Errorf("argv[%d] contains shell syntax", index)
		}
		if hasShellInterpreter(arg) {
			return fmt.Errorf("argv[%d] must not invoke a shell interpreter", index)
		}
		if hasCustomScriptPath(arg) {
			return fmt.Errorf("argv[%d] must not reference a custom script", index)
		}
	}
	return nil
}

func hasShellInterpreter(arg string) bool {
	name := strings.ToLower(filepath.Base(strings.ReplaceAll(arg, "\\", "/")))
	switch strings.TrimSuffix(name, ".exe") {
	case "sh", "bash", "zsh", "fish", "cmd", "powershell", "pwsh":
		return true
	default:
		return false
	}
}

func hasCustomScriptPath(arg string) bool {
	lower := strings.ToLower(arg)
	return strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash") ||
		strings.HasSuffix(lower, ".zsh") || strings.HasSuffix(lower, ".fish") ||
		strings.HasSuffix(lower, ".ps1") || strings.HasSuffix(lower, ".cmd") ||
		strings.HasSuffix(lower, ".bat")
}

func validateWorktreeRelativeDirectory(directory string) error {
	if directory == "" || filepath.IsAbs(directory) || filepath.VolumeName(directory) != "" {
		return fmt.Errorf("working_directory must be a relative worktree path")
	}
	cleaned := filepath.Clean(directory)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("working_directory must not escape the worktree")
	}
	return nil
}

func validateAllowedEnvironment(names []string) error {
	seenNames := make(map[string]struct{}, len(names))
	for index, name := range names {
		if !environmentNamePattern.MatchString(name) {
			return fmt.Errorf("allowed_environment[%d] has invalid name %q", index, name)
		}
		if _, exists := seenNames[name]; exists {
			return fmt.Errorf("allowed_environment has duplicate name %q", name)
		}
		seenNames[name] = struct{}{}
	}
	return nil
}

// Normalized returns the canonical semantic representation used for digesting.
// It does not validate input; callers must validate a plan before authorizing
// or executing it. Sorting removes irrelevant authored ordering differences.
func (p VerificationPlan) Normalized() VerificationPlan {
	normalized := p
	normalized.Checks = append([]VerificationCheck(nil), p.Checks...)
	for i := range normalized.Checks {
		check := &normalized.Checks[i]
		check.Argv = append([]string(nil), check.Argv...)
		check.AllowedEnvironment = append([]string(nil), check.AllowedEnvironment...)
		sort.Strings(check.AllowedEnvironment)
	}
	sort.SliceStable(normalized.Checks, func(i, j int) bool {
		return normalized.Checks[i].CheckID < normalized.Checks[j].CheckID
	})
	return normalized
}

// Digest returns the SHA-256 digest of the canonical JSON representation.
// JSON encoding of these structs is stable because the model contains no maps.
func (p VerificationPlan) Digest() (string, error) {
	b, err := json.Marshal(p.Normalized())
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
