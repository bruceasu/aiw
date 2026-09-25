package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ScopeTestPlan is separate from VerificationPlan v1: old fixed selections
// never acquire discovery rights by normalization or migration.
type ScopeTestPlan struct {
	SchemaVersion int `json:"schema_version"`
	TaskID TaskID `json:"task_id"`
	Permission string `json:"permission"`
	WorkspaceDigest string `json:"workspace_digest"`
	InputScope []string `json:"input_scope"`
	TestPaths []string `json:"test_paths"`
	FixturePaths []string `json:"fixture_paths"`
	Checks []VerificationCheck `json:"checks"`
	MaxFiles int `json:"max_files"`
	MaxInputBytes int64 `json:"max_input_bytes"`
	MaxOutputBytes int `json:"max_output_bytes"`
	MaxMemoryBytes int64 `json:"max_memory_bytes"`
	MaxProcesses int `json:"max_processes"`
	SideEffects []string `json:"side_effects"`
}

type TestExecutionPolicy struct {
	SchemaVersion int `json:"schema_version"`
	Version string `json:"version"`
	TaskID TaskID `json:"task_id"`
	WorkspaceDigest string `json:"workspace_digest"`
	InputScope []string `json:"input_scope"`
	TestPaths []string `json:"test_paths"`
	FixturePaths []string `json:"fixture_paths"`
	Checks []VerificationCheck `json:"checks"`
	MaxFiles int `json:"max_files"`
	MaxInputBytes int64 `json:"max_input_bytes"`
	MaxOutputBytes int `json:"max_output_bytes"`
	MaxMemoryBytes int64 `json:"max_memory_bytes"`
	MaxProcesses int `json:"max_processes"`
	// N/A is limited to exact, reviewed descriptive documents. A filename
	// extension alone never establishes that prompts/config/scripts are inert.
	DescriptiveDocuments []ActorReference `json:"descriptive_documents"`
}

type TestTarget struct {
	Path string `json:"path"`
	RealPath string `json:"real_path"`
	SHA256 string `json:"sha256"`
}

type TestInvocation struct {
	Check VerificationCheck `json:"check"`
	Directory string `json:"directory"`
	Executable TestTarget `json:"executable"`
	Environment []string `json:"environment"`
}

type TestManifest struct {
	SchemaVersion int `json:"schema_version"`
	RequestID string `json:"request_id"`
	TaskID TaskID `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	Workspace string `json:"workspace"`
	Plan ActorReference `json:"plan"`
	Grant ActorReference `json:"grant"`
	Policy ActorReference `json:"policy"`
	Selection []string `json:"selection"`
	Inputs ValidationInputs `json:"inputs"`
	Targets []TestTarget `json:"targets"`
	Tests []string `json:"tests"`
	Fixtures []string `json:"fixtures"`
	Invocations []TestInvocation `json:"invocations"`
	Scope GrantScope `json:"scope"`
	Limits ScopeTestPlan `json:"limits"`
}

func WorkspaceBindingDigest(state RuntimeState) string {
	data, _ := canonicalJSON(struct {
		Task TaskID `json:"task"`
		Workspace string `json:"workspace"`
		Binding any `json:"binding"`
	}{state.Task.ID, executionWorkspace(state), state.Workspace})
	return contentDigest(data)
}

func (s *Store) readStrictArtifact(id TaskID, ref ActorReference, value any) error {
	var raw json.RawMessage
	if err := s.ReadExecutionArtifact(id, ref, &raw); err != nil { return err }
	path, err := confinedFile(s.path(id, ""), ref.Path)
	if err != nil { return err }
	data, err := os.ReadFile(path)
	if err != nil { return err }
	if contentDigest(data) != ref.SHA256 { return errors.New("artifact changed while reading its complete JSON") }
	return strictJSON(data, value)
}

func equalJSON(a, b any) bool {
	x, e1 := canonicalJSON(a)
	y, e2 := canonicalJSON(b)
	return e1 == nil && e2 == nil && bytes.Equal(x, y)
}

func (s *Store) testPlan(state RuntimeState, planRef, policyRef ActorReference, selection []string) (ScopeTestPlan, error) {
	var policy TestExecutionPolicy
	if err := s.readStrictArtifact(state.Task.ID, policyRef, &policy); err != nil { return ScopeTestPlan{}, err }
	if policy.SchemaVersion != 1 || policy.Version == "" || policy.Version == "latest" || policy.TaskID != state.Task.ID || policy.WorkspaceDigest != WorkspaceBindingDigest(state) { return ScopeTestPlan{}, errors.New("stale: test policy binding") }
	var raw json.RawMessage
	if err := s.ReadExecutionArtifact(state.Task.ID, planRef, &raw); err != nil { return ScopeTestPlan{}, err }
	var version struct { SchemaVersion int `json:"schema_version"` }
	if err := json.Unmarshal(raw, &version); err != nil { return ScopeTestPlan{}, err }
	var plan ScopeTestPlan
	switch version.SchemaVersion {
	case 1:
		var fixed VerificationPlan
		if err := strictJSON(raw, &fixed); err != nil { return plan, err }
		if err := fixed.Validate(); err != nil { return plan, err }
		if fixed.TaskID != state.Task.ID || !canonicalSet(selection, true) { return plan, errors.New("fixed plan requires exact Task and check selection") }
		plan = ScopeTestPlan{SchemaVersion: 1, TaskID: fixed.TaskID, Permission: "test-run", WorkspaceDigest: policy.WorkspaceDigest, InputScope: policy.InputScope, TestPaths: policy.TestPaths, FixturePaths: policy.FixturePaths, MaxFiles: policy.MaxFiles, MaxInputBytes: policy.MaxInputBytes, MaxOutputBytes: policy.MaxOutputBytes, MaxMemoryBytes: policy.MaxMemoryBytes, MaxProcesses: policy.MaxProcesses, SideEffects: []string{"isolated-temporary-output"}}
		for _, id := range selection {
			found := false
			for _, check := range fixed.Checks { if check.CheckID == id { plan.Checks = append(plan.Checks, check); found = true } }
			if !found { return plan, errors.New("fixed plan selection is unknown") }
		}
	case 2:
		if err := strictJSON(raw, &plan); err != nil { return plan, err }
		if len(selection) != 0 { return plan, errors.New("scope plan executes its complete frozen check list") }
	default: return plan, errors.New("unsupported test plan schema")
	}
	if plan.TaskID != state.Task.ID || plan.Permission != "test-run" || plan.WorkspaceDigest != policy.WorkspaceDigest { return plan, errors.New("stale: test plan binding") }
	// Policies authorize concrete templates, not arbitrary program strings.
	if !equalJSON(plan.InputScope, policy.InputScope) || !equalJSON(plan.TestPaths, policy.TestPaths) || !equalJSON(plan.FixturePaths, policy.FixturePaths) { return plan, errors.New("denied: plan expands policy scope") }
	if !canonicalSet(plan.InputScope, true) || !canonicalSet(plan.TestPaths, true) || !canonicalSet(plan.FixturePaths, false) || !equalJSON(plan.SideEffects, []string{"isolated-temporary-output"}) { return plan, errors.New("unknown discovery or side-effect policy") }
	for _, root := range plan.InputScope { if err := validateWorktreeRelativeDirectory(root); err != nil { return plan, err } }
	for _, pattern := range append(append([]string{}, plan.TestPaths...), plan.FixturePaths...) { if err := validateRepositoryPattern(pattern); err != nil { return plan, err } }
	if plan.MaxFiles <= 0 || plan.MaxFiles > policy.MaxFiles || plan.MaxInputBytes <= 0 || plan.MaxInputBytes > policy.MaxInputBytes || plan.MaxOutputBytes <= 0 || plan.MaxOutputBytes > policy.MaxOutputBytes || plan.MaxMemoryBytes <= 0 || plan.MaxMemoryBytes > policy.MaxMemoryBytes || plan.MaxProcesses <= 0 || plan.MaxProcesses > policy.MaxProcesses { return plan, errors.New("denied: resource limits are absent or expanded") }
	if err := (VerificationPlan{SchemaVersion: 1, TaskID: plan.TaskID, Checks: plan.Checks}).Validate(); err != nil { return plan, err }
	for _, check := range plan.Checks {
		allowed := false
		for _, template := range policy.Checks { if equalJSON(check, template) { allowed = true; break } }
		if !allowed { return plan, errors.New("denied: command template differs from policy") }
		if check.ExpectedExitCode != 0 { return plan, errors.New("unit tests must succeed with exit code zero") }
		// Only registered unit-test tools can reach the host. git and arbitrary
		// interpreters cannot hide delivery/delete operations in a test plan.
		tool := strings.TrimSuffix(strings.ToLower(filepath.Base(check.Argv[0])), ".exe")
		switch tool {
		case "go": if len(check.Argv) < 2 || check.Argv[1] != "test" { return plan, errors.New("only go test is a registered test command") }
		case "python", "python3": if len(check.Argv) < 3 || check.Argv[1] != "-m" || (check.Argv[2] != "pytest" && check.Argv[2] != "unittest") { return plan, errors.New("only Python unit-test modules are registered") }
		default: return plan, errors.New("host-unavailable: test tool adapter is not registered")
		}
	}
	return plan, nil
}

func matchesTestPath(patterns []string, path string) bool {
	for _, pattern := range patterns { if repositoryPatternMatches(pattern, path) { return true } }
	return false
}

// PrepareTestManifest freezes discovery separately from the reusable grant.
// It is read-only; the caller persists it before PrepareStage reserves budget.
func (s *Store) PrepareTestManifest(state RuntimeState, request StageRequest, selection []string) (TestManifest, error) {
	m := TestManifest{SchemaVersion: 1, RequestID: request.ID, TaskID: state.Task.ID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Workspace: executionWorkspace(state), Plan: request.Plan, Grant: request.Grant, Policy: request.Policy, Selection: append([]string{}, selection...), Targets: []TestTarget{}, Tests: []string{}, Fixtures: []string{}, Invocations: []TestInvocation{}}
	plan, err := s.testPlan(state, request.Plan, request.Policy, selection)
	if err != nil { return m, err }
	m.Limits = plan
	if err := boundTestInputs(m.Workspace, plan); err != nil { return m, err }
	toolchain, err := ValidationToolchainIdentity()
	if err != nil { return m, err }
	m.Inputs, err = CaptureValidationInputs(m.Workspace, plan.InputScope, []ActorReference{request.Plan, request.Policy}, toolchain)
	if err != nil { return m, err }
	if len(m.Inputs.Files) > plan.MaxFiles { return m, errors.New("test input file budget exceeded") }
	var total int64
	for _, file := range m.Inputs.Files {
		real, err := confinedFile(m.Workspace, file.Path)
		if err != nil { return m, err }
		info, err := os.Stat(real)
		if err != nil { return m, err }
		total += info.Size()
		if total > plan.MaxInputBytes { return m, errors.New("test input byte budget exceeded") }
		m.Targets = append(m.Targets, TestTarget{Path: file.Path, RealPath: real, SHA256: file.SHA256})
		if matchesTestPath(plan.TestPaths, file.Path) { m.Tests = append(m.Tests, file.Path) }
		if matchesTestPath(plan.FixturePaths, file.Path) { m.Fixtures = append(m.Fixtures, file.Path) }
	}
	if len(m.Tests) == 0 { return m, errors.New("test discovery is empty; this is not a passed run") }
	scopeTargets := []string{}
	for _, root := range plan.InputScope {
		real, err := confinedFile(m.Workspace, root)
		if err != nil { return m, err }
		scopeTargets = append(scopeTargets, real)
	}
	for _, target := range m.Targets {
		inside := false
		for _, root := range scopeTargets {
			rel, err := filepath.Rel(root, target.RealPath)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) { inside = true }
		}
		if !inside { return m, errors.New("denied: linked input leaves its approved module roots") }
	}
	for _, check := range plan.Checks {
		directory, err := confinedFile(m.Workspace, check.WorkingDirectory)
		if err != nil { return m, err }
		info, err := os.Stat(directory)
		if err != nil || !info.IsDir() { return m, errors.New("test directory is unavailable") }
		program, err := exec.LookPath(check.Argv[0])
		if err != nil { return m, err }
		program, err = filepath.EvalSymlinks(program)
		if err != nil { return m, err }
		program, err = filepath.Abs(program)
		if err != nil { return m, err }
		data, err := os.ReadFile(program)
		if err != nil { return m, err }
		environment := []string{}
		for _, name := range check.AllowedEnvironment { if value, exists := os.LookupEnv(name); exists { environment = append(environment, name+"="+value) } }
		sort.Strings(environment)
		m.Invocations = append(m.Invocations, TestInvocation{Check: check, Directory: directory, Executable: TestTarget{Path: check.Argv[0], RealPath: program, SHA256: contentDigest(data)}, Environment: environment})
		scopeTargets = append(scopeTargets, directory, program)
	}
	sort.Strings(scopeTargets)
	unique := []string{}
	for _, target := range scopeTargets { if len(unique) == 0 || target != unique[len(unique)-1] { unique = append(unique, target) } }
	m.Scope = GrantScope{WorkspaceDigest: plan.WorkspaceDigest, Paths: plan.InputScope, Actions: []string{"test-run"}, Targets: unique}
	return m, nil
}

// Enforce file/byte ceilings before CaptureValidationInputs reads content.
func boundTestInputs(root string, plan ScopeTestPlan) error {
	seen := map[string]bool{}
	var total int64
	for _, scope := range plan.InputScope {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(scope)), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil { return walkErr }
			rel, err := filepath.Rel(root, path)
			if err != nil { return err }
			if entry.IsDir() {
				if entry.Name() == ".git" || rel == ".ai" || rel == ".wt" { return filepath.SkipDir }
				return nil
			}
			if seen[rel] { return nil }
			seen[rel] = true
			if len(seen) > plan.MaxFiles { return errors.New("test input file budget exceeded") }
			real, err := confinedFile(root, rel)
			if err != nil { return err }
			info, err := os.Stat(real)
			if err != nil { return err }
			if !info.Mode().IsRegular() || info.Size() > plan.MaxInputBytes-total { return errors.New("test input byte/type budget exceeded") }
			total += info.Size()
			return nil
		})
		if err != nil { return err }
	}
	return nil
}

func (s *Store) CheckTestManifest(state RuntimeState, request StageRequest) (TestManifest, error) {
	var frozen TestManifest
	if err := s.readStrictArtifact(state.Task.ID, request.Input, &frozen); err != nil { return frozen, err }
	current, err := s.PrepareTestManifest(state, request, frozen.Selection)
	if err != nil { return frozen, err }
	if !equalJSON(current, frozen) || request.InputDigest != frozen.Inputs.Digest() { return frozen, errors.New("stale: test manifest, real target or input content changed") }
	log, err := s.readGrantLog(state)
	if err != nil { return frozen, err }
	if err := log.authorize(request.Grant, request.Plan, request.Policy, frozen.Scope); err != nil { return frozen, err }
	return frozen, nil
}

// PersistVerificationArtifact stores immutable preparation/receipt evidence;
// it cannot advance execution, grant rights, or close a Work Item.
func (s *Store) PersistVerificationArtifact(id TaskID, kind string, value any) (ActorReference, error) {
	if err := validateTaskID(id); err != nil { return ActorReference{}, err }
	lock, err := s.lock(id)
	if err != nil { return ActorReference{}, err }
	defer unlock(lock)
	state, err := s.Load(id)
	if err != nil { return ActorReference{}, err }
	if state.SchemaVersion != DurableSchemaVersion || !isSystemLock(lock) { return ActorReference{}, errors.New("verification evidence requires the migrated Store") }
	if !strings.HasPrefix(kind, "verification-") { return ActorReference{}, fmt.Errorf("unsupported verification artifact kind %q", kind) }
	return s.persistProtocolArtifactLocked(id, kind, value)
}
