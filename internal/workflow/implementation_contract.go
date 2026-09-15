package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ImplementationContract is Analysis' durable operational handoff.  It is
// deliberately narrower than OpenSpec design: it describes the one Work Item
// surface that Coder may implement and Tester may later observe.
type ImplementationContract struct {
	Status                 ImplementationContractStatus `json:"status"`
	FunctionalSurface      []string                     `json:"functional_surface"`
	PublicInterfaces       []string                     `json:"public_interfaces"`
	TestSeams              []string                     `json:"test_seams"`
	AllowedProductionPaths []string                     `json:"allowed_production_paths"`
	AllowedTestPaths       []string                     `json:"allowed_test_paths,omitempty"`
	Constraints            []string                     `json:"constraints,omitempty"`
	OpenQuestions          []string                     `json:"open_questions,omitempty"`
}

type ImplementationContractStatus string

const (
	ImplementationContractReady      ImplementationContractStatus = "READY"
	ImplementationContractNeedsInput ImplementationContractStatus = "NEEDS_INPUT"
	ImplementationContractComplete   ImplementationContractStatus = "COMPLETE"
	ImplementationContractGateID     GateID = "implementation_contract_needs_input"
	ActorWriteScopeGateID             GateID = "actor_write_scope_violated"
)

func (c ImplementationContract) Validate() error {
	if c.Status != ImplementationContractReady && c.Status != ImplementationContractNeedsInput && c.Status != ImplementationContractComplete {
		return fmt.Errorf("implementation contract has invalid status %q", c.Status)
	}
	if c.Status == ImplementationContractNeedsInput {
		if len(c.OpenQuestions) == 0 { return errors.New("needs-input implementation contract requires open questions") }
		return validateContractText(c.OpenQuestions, "open question")
	}
	if err := validateContractText(c.FunctionalSurface, "functional surface"); err != nil { return err }
	if err := validateContractText(c.PublicInterfaces, "public interface"); err != nil { return err }
	if err := validateContractText(c.TestSeams, "test seam"); err != nil { return err }
	if len(c.AllowedProductionPaths) == 0 { return errors.New("implementation contract requires allowed production paths") }
	for _, pattern := range c.AllowedProductionPaths {
		if err := validateRepositoryPattern(pattern); err != nil { return fmt.Errorf("allowed production path %q: %w", pattern, err) }
	}
	return validateContractText(c.Constraints, "constraint")
}

func validateContractText(values []string, name string) error {
	for _, value := range values { if strings.TrimSpace(value) == "" { return fmt.Errorf("%s is required", name) } }
	return nil
}

// PersistImplementationContract stores the human-readable contract below the
// Task runtime root and records a Gate for an incomplete Analysis result.
func (s *Store) PersistImplementationContract(id TaskID, workItemID WorkItemID, contract ImplementationContract) (ActorReference, RuntimeState, error) {
	if err := contract.Validate(); err != nil { return ActorReference{}, RuntimeState{}, err }
	if strings.TrimSpace(string(workItemID)) == "" { return ActorReference{}, RuntimeState{}, errors.New("implementation contract work item is required") }
	relative := filepath.ToSlash(filepath.Join("implementation-contracts", string(workItemID)+".md"))
	content := []byte(contract.Markdown())
	if err := atomicWrite(s.path(id, filepath.FromSlash(relative)), content); err != nil { return ActorReference{}, RuntimeState{}, err }
	sum := sha256.Sum256(content)
	reference := ActorReference{Kind: "implementation-contract", Path: relative, SHA256: hex.EncodeToString(sum[:])}
	state, err := s.UpdateWithEvent(id, Event{Type: "implementation-contract.persisted", WorkItemID: workItemID, Detail: relative}, func(state *RuntimeState) error {
		if !runtimeHasWorkItem(*state, workItemID) { return fmt.Errorf("unknown work item %s", workItemID) }
		if contract.Status != ImplementationContractNeedsInput { return nil }
		reason := "implementation contract needs input: " + strings.Join(contract.OpenQuestions, "; ")
		for index := range state.Gates {
			if state.Gates[index].ID == ImplementationContractGateID {
				state.Gates[index].WorkItemID, state.Gates[index].Kind, state.Gates[index].State, state.Gates[index].Reason = workItemID, GateDecision, GateOpen, reason
				return nil
			}
		}
		state.Gates = append(state.Gates, Gate{ID: ImplementationContractGateID, WorkItemID: workItemID, Kind: GateDecision, State: GateOpen, Reason: reason})
		return nil
	})
	if err != nil { return ActorReference{}, RuntimeState{}, err }
	return reference, state, nil
}

func (c ImplementationContract) Markdown() string {
	section := func(name string, values []string) string {
		var lines []string
		for _, value := range values { lines = append(lines, "- "+value) }
		if len(lines) == 0 { lines = append(lines, "- None") }
		return "## " + name + "\n" + strings.Join(lines, "\n") + "\n\n"
	}
	return "# Implementation Contract\n\nStatus: " + string(c.Status) + "\n\n" +
		section("Functional Surface", c.FunctionalSurface) + section("Public Interfaces", c.PublicInterfaces) +
		section("Test Seams", c.TestSeams) + section("Allowed Production Paths", c.AllowedProductionPaths) +
		section("Allowed Test Paths", c.AllowedTestPaths) +
		section("Constraints", c.Constraints) + section("Open Questions", c.OpenQuestions)
}

// ActorSandbox receives the approved writable roots when the platform can
// prevent writes.  A nil sandbox is supported for backends without this
// capability; post-Actor Git validation remains mandatory in both cases.
type ActorSandbox interface {
	RunCoder(context.Context, CoderRequest, []string) (CoderResult, error)
}

// ChangedPathSnapshot captures tracked, deleted, renamed, and untracked
// entries obtained from `git status --porcelain=v1 -z`.
type ChangedPathSnapshot map[string]string

type ChangedPathChecker interface { Snapshot(workspace string) (ChangedPathSnapshot, error) }

type GitChangedPathChecker struct {
	// Environment optionally carries the supervisor's scoped Git preflight.
	// Nil preserves the inherited environment for existing callers.
	Environment []string
}

func (checker GitChangedPathChecker) Snapshot(workspace string) (ChangedPathSnapshot, error) {
	command := exec.Command("git", "-C", workspace, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	command.Env = checker.Environment
	output, err := command.Output()
	if err != nil { return nil, fmt.Errorf("snapshot Git changed paths: %w", err) }
	return parseGitPorcelain(output)
}

func parseGitPorcelain(output []byte) (ChangedPathSnapshot, error) {
	result := ChangedPathSnapshot{}
	records := strings.Split(string(output), "\x00")
	for index := 0; index < len(records); index++ {
		record := records[index]
		if record == "" { continue }
		if len(record) < 4 || record[2] != ' ' { return nil, fmt.Errorf("invalid Git porcelain record") }
		status, candidate := record[:2], record[3:]
		if err := addChangedPath(result, candidate, status); err != nil { return nil, err }
		if strings.ContainsAny(status, "RC") {
			index++
			if index >= len(records) || records[index] == "" { return nil, fmt.Errorf("rename or copy record is missing original path") }
			if err := addChangedPath(result, records[index], status); err != nil { return nil, err }
		}
	}
	return result, nil
}

func addChangedPath(snapshot ChangedPathSnapshot, candidate, status string) error {
	normalized, err := normalizeRepositoryPath(candidate)
	if err != nil { return err }
	snapshot[normalized] = status
	return nil
}

// ChangedPathsSince returns the complete changed-path delta made after a
// pre-Actor snapshot. Existing human changes are not attributed to the Actor.
func ChangedPathsSince(before, after ChangedPathSnapshot) []string {
	changed := make([]string, 0)
	for candidate, status := range after { if before[candidate] != status { changed = append(changed, candidate) } }
	for candidate := range before { if _, exists := after[candidate]; !exists { changed = append(changed, candidate) } }
	sort.Strings(changed)
	return changed
}

func ValidateCoderChangedPaths(contract ImplementationContract, changed []string) error {
	if contract.Status != ImplementationContractReady { return errors.New("coder requires a READY implementation contract") }
	for _, candidate := range changed {
		normalized, err := normalizeRepositoryPath(candidate)
		if err != nil { return err }
		allowed := false
		for _, pattern := range contract.AllowedProductionPaths { if repositoryPatternMatches(pattern, normalized) { allowed = true; break } }
		if !allowed { return &ActorWriteScopeViolation{Actor: ActorCoder, Paths: []string{normalized}} }
	}
	return nil
}

// RunCoder enforces both available platform isolation and a mandatory Git
// delta check. Violating files stay in the worktree for diagnosis, while a
// durable Gate and failed Evidence stop later scheduling.
func (s *Store) RunCoder(ctx context.Context, id TaskID, request CoderRequest, contract ImplementationContract, sandbox ActorSandbox, checker ChangedPathChecker) (RuntimeState, CoderResult, error) {
	if err := request.Validate(); err != nil { return RuntimeState{}, CoderResult{}, err }
	if err := contract.Validate(); err != nil { return RuntimeState{}, CoderResult{}, err }
	if sandbox == nil || checker == nil { return RuntimeState{}, CoderResult{}, errors.New("coder requires sandbox and Git changed-path checker") }
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, CoderResult{}, err }
	if request.TaskID != id || state.WriteLease == nil || state.WriteLease.AttemptID != request.AttemptID || state.WriteLease.Workspace != request.Workspace { return RuntimeState{}, CoderResult{}, errors.New("coder request does not own the workspace write lease") }
	before, err := checker.Snapshot(request.Workspace)
	if err != nil { return RuntimeState{}, CoderResult{}, err }
	result, runErr := sandbox.RunCoder(ctx, request, append([]string(nil), contract.AllowedProductionPaths...))
	after, snapshotErr := checker.Snapshot(request.Workspace)
	if snapshotErr != nil { return RuntimeState{}, result, snapshotErr }
	changed := ChangedPathsSince(before, after)
	if scopeErr := ValidateCoderChangedPaths(contract, changed); scopeErr != nil {
		updated, recordErr := s.RecordActorWriteScopeViolation(id, request.WorkItemID, request.AttemptID, changed, scopeErr.Error())
		if recordErr != nil { return RuntimeState{}, result, recordErr }
		return updated, result, scopeErr
	}
	if runErr != nil { return state, result, runErr }
	return state, result, nil
}

type ActorWriteScopeViolation struct { Actor ActorKind; Paths []string }
func (e *ActorWriteScopeViolation) Error() string { return fmt.Sprintf("%s wrote outside its approved paths: %s", e.Actor, strings.Join(e.Paths, ", ")) }

func (s *Store) RecordActorWriteScopeViolation(id TaskID, workItemID WorkItemID, attemptID AttemptID, paths []string, detail string) (RuntimeState, error) {
	paths = append([]string(nil), paths...); sort.Strings(paths)
	return s.UpdateWithEvent(id, Event{Type: "actor.write-scope-violated", WorkItemID: workItemID, AttemptID: attemptID, Detail: strings.Join(paths, ",")}, func(state *RuntimeState) error {
		if !runtimeHasWorkItem(*state, workItemID) { return fmt.Errorf("unknown work item %s", workItemID) }
		evidenceID := EvidenceID("actor-write-scope-" + string(attemptID))
		for _, evidence := range state.Evidence { if evidence.ID == evidenceID { return fmt.Errorf("actor write-scope violation already recorded for %s", attemptID) } }
		state.Evidence = append(state.Evidence, Evidence{ID: evidenceID, WorkItemID: workItemID, Kind: EvidenceManual, State: EvidenceFailed, Reference: detail + "; paths=" + strings.Join(paths, ",")})
		for index := range state.Gates {
			if state.Gates[index].ID == ActorWriteScopeGateID { state.Gates[index].WorkItemID, state.Gates[index].Kind, state.Gates[index].State, state.Gates[index].Reason = workItemID, GateDependency, GateOpen, detail; return nil }
		}
		state.Gates = append(state.Gates, Gate{ID: ActorWriteScopeGateID, WorkItemID: workItemID, Kind: GateDependency, State: GateOpen, Reason: detail})
		return nil
	})
}

func runtimeHasWorkItem(state RuntimeState, wanted WorkItemID) bool { for _, item := range state.WorkItems { if item.ID == wanted { return true } }; return false }

func validateRepositoryPattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" || strings.HasPrefix(pattern, "/") || strings.Contains(pattern, "\\") || strings.Contains(pattern, "..") { return errors.New("must be a relative slash-separated repository pattern") }
	return nil
}

func normalizeRepositoryPath(candidate string) (string, error) {
	candidate = strings.TrimSpace(strings.ReplaceAll(candidate, "\\", "/"))
	if candidate == "" || strings.HasPrefix(candidate, "/") || filepath.IsAbs(candidate) { return "", errors.New("changed path must be relative") }
	normalized := path.Clean(candidate)
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") { return "", errors.New("changed path escapes the repository") }
	return normalized, nil
}

func repositoryPatternMatches(pattern, candidate string) bool {
	if pattern == candidate { return true }
	if strings.HasSuffix(pattern, "/**") { return strings.HasPrefix(candidate, strings.TrimSuffix(pattern, "**")) }
	matched, err := path.Match(pattern, candidate)
	return err == nil && matched
}
