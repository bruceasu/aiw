package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const TestAuthoringSchemaVersion = 1

// ObservableInterfaceInventory is the compiler-success handoff to Tester.
// Entries describe behavior that exists in the compiled Work Item, rather
// than a proposed shell command or a broad source-tree snapshot.
type ObservableInterfaceInventory struct {
	SchemaVersion int                   `json:"schema_version"`
	WorkItemID    WorkItemID            `json:"work_item_id"`
	Interfaces    []ObservableInterface `json:"interfaces"`
}

type ObservableInterface struct {
	Kind      string   `json:"kind"`
	Symbol    string   `json:"symbol"`
	Source    string   `json:"source"`
	Inputs    []string `json:"inputs,omitempty"`
	Outputs   []string `json:"outputs,omitempty"`
	Errors    []string `json:"errors,omitempty"`
	Scenarios []string `json:"scenarios,omitempty"`
	TestSeam  string   `json:"test_seam,omitempty"`
}

func (i ObservableInterfaceInventory) Validate() error {
	if i.SchemaVersion != TestAuthoringSchemaVersion || i.WorkItemID == "" || len(i.Interfaces) == 0 {
		return errors.New("interface inventory requires schema, work item, and at least one interface")
	}
	seen := map[string]struct{}{}
	for index, item := range i.Interfaces {
		if strings.TrimSpace(item.Kind) == "" || strings.TrimSpace(item.Symbol) == "" {
			return fmt.Errorf("interface %d requires kind and symbol", index)
		}
		if _, err := normalizeRepositoryPath(item.Source); err != nil { return fmt.Errorf("interface %d source: %w", index, err) }
		if _, exists := seen[item.Symbol]; exists { return fmt.Errorf("duplicate interface symbol %q", item.Symbol) }
		seen[item.Symbol] = struct{}{}
	}
	return nil
}

// TestCaseInventory records authored unit tests. Execution state is deliberately
// absent: the later TestExecutionPlan owns executing these cases.
type TestCaseInventory struct {
	SchemaVersion int        `json:"schema_version"`
	WorkItemID    WorkItemID `json:"work_item_id"`
	Cases         []TestCase `json:"cases"`
}

type TestCase struct {
	ID          string   `json:"id"`
	WorkItemID  WorkItemID `json:"work_item_id"`
	ScenarioIDs []string `json:"scenario_ids"`
	Interface   string   `json:"interface"`
	TestFile    string   `json:"test_file"`
	TestName    string   `json:"test_name"`
	Kind        string   `json:"kind"`
	Status      string   `json:"status"`
}

func (i TestCaseInventory) Validate() error {
	if i.SchemaVersion != TestAuthoringSchemaVersion || i.WorkItemID == "" || len(i.Cases) == 0 {
		return errors.New("test-case inventory requires schema, work item, and at least one case")
	}
	seen := map[string]struct{}{}
	for index, testCase := range i.Cases {
		if strings.TrimSpace(testCase.ID) == "" || testCase.WorkItemID != i.WorkItemID || len(testCase.ScenarioIDs) == 0 || strings.TrimSpace(testCase.Interface) == "" || strings.TrimSpace(testCase.TestName) == "" || testCase.Kind != "unit" || testCase.Status != "authored" {
			return fmt.Errorf("test case %d must be an authored unit case mapped to its work item", index)
		}
		if _, err := normalizeRepositoryPath(testCase.TestFile); err != nil { return fmt.Errorf("test case %d file: %w", index, err) }
		if _, exists := seen[testCase.ID]; exists { return fmt.Errorf("duplicate test case id %q", testCase.ID) }
		seen[testCase.ID] = struct{}{}
	}
	return nil
}

// TesterSandbox gives a platform adapter the only allowed test roots. It
// authors tests and returns structured inventory; it has no execution command.
type TesterSandbox interface {
	AuthorTests(context.Context, TesterRequest, []string) (TestCaseInventory, error)
}

func ValidateTesterChangedPaths(contract ImplementationContract, changed []string) error {
	if contract.Status != ImplementationContractReady { return errors.New("tester requires a READY implementation contract") }
	if len(contract.AllowedTestPaths) == 0 { return errors.New("tester requires explicit allowed test paths") }
	for _, pattern := range contract.AllowedTestPaths {
		if err := validateRepositoryPattern(pattern); err != nil { return fmt.Errorf("allowed test path %q: %w", pattern, err) }
	}
	for _, candidate := range changed {
		normalized, err := normalizeRepositoryPath(candidate)
		if err != nil { return err }
		allowed := false
		for _, pattern := range contract.AllowedTestPaths { if repositoryPatternMatches(pattern, normalized) { allowed = true; break } }
		if !allowed { return &ActorWriteScopeViolation{Actor: ActorTester, Paths: []string{normalized}} }
	}
	return nil
}

// PersistObservableInterfaceInventory makes the compiled public surface
// recoverable before a Tester request is prepared.
func (s *Store) PersistObservableInterfaceInventory(id TaskID, inventory ObservableInterfaceInventory) (ActorReference, error) {
	if err := inventory.Validate(); err != nil { return ActorReference{}, err }
	content, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil { return ActorReference{}, fmt.Errorf("encode interface inventory: %w", err) }
	content = append(content, '\n')
	relative := filepath.ToSlash(filepath.Join("observable-interfaces", string(inventory.WorkItemID)+".json"))
	if err := atomicWrite(s.path(id, filepath.FromSlash(relative)), content); err != nil { return ActorReference{}, err }
	sum := sha256.Sum256(content)
	reference := ActorReference{Kind: "observable-interface-inventory", Path: relative, SHA256: hex.EncodeToString(sum[:])}
	_, err = s.UpdateWithEvent(id, Event{Type: "observable-interface-inventory.persisted", WorkItemID: inventory.WorkItemID, Detail: relative}, func(state *RuntimeState) error {
		if !runtimeHasWorkItem(*state, inventory.WorkItemID) { return fmt.Errorf("unknown work item %s", inventory.WorkItemID) }
		return nil
	})
	return reference, err
}

func (s *Store) PersistTestCaseInventory(id TaskID, inventory TestCaseInventory) (ActorReference, error) {
	if err := inventory.Validate(); err != nil { return ActorReference{}, err }
	content, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil { return ActorReference{}, fmt.Errorf("encode test-case inventory: %w", err) }
	content = append(content, '\n')
	relative := filepath.ToSlash(filepath.Join("test-case-inventories", string(inventory.WorkItemID)+".json"))
	if err := atomicWrite(s.path(id, filepath.FromSlash(relative)), content); err != nil { return ActorReference{}, err }
	sum := sha256.Sum256(content)
	reference := ActorReference{Kind: "test-case-inventory", Path: relative, SHA256: hex.EncodeToString(sum[:])}
	_, err = s.UpdateWithEvent(id, Event{Type: "test-case-inventory.persisted", WorkItemID: inventory.WorkItemID, Detail: relative}, func(state *RuntimeState) error {
		if !runtimeHasWorkItem(*state, inventory.WorkItemID) { return fmt.Errorf("unknown work item %s", inventory.WorkItemID) }
		return nil
	})
	return reference, err
}

// RunTester enforces test-only authoring with the same portable Git boundary
// used by Coder. It intentionally does not execute the authored tests.
func (s *Store) RunTester(ctx context.Context, id TaskID, request TesterRequest, contract ImplementationContract, sandbox TesterSandbox, checker ChangedPathChecker) (RuntimeState, TesterResult, error) {
	if err := request.Validate(); err != nil { return RuntimeState{}, TesterResult{}, err }
	if err := contract.Validate(); err != nil { return RuntimeState{}, TesterResult{}, err }
	if sandbox == nil || checker == nil { return RuntimeState{}, TesterResult{}, errors.New("tester requires sandbox and Git changed-path checker") }
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, TesterResult{}, err }
	if request.TaskID != id || state.WriteLease == nil || state.WriteLease.AttemptID != request.AttemptID || state.WriteLease.Workspace != request.Workspace { return RuntimeState{}, TesterResult{}, errors.New("tester request does not own the workspace write lease") }
	before, err := checker.Snapshot(request.Workspace)
	if err != nil { return RuntimeState{}, TesterResult{}, err }
	inventory, runErr := sandbox.AuthorTests(ctx, request, append([]string(nil), contract.AllowedTestPaths...))
	after, snapshotErr := checker.Snapshot(request.Workspace)
	if snapshotErr != nil { return RuntimeState{}, TesterResult{}, snapshotErr }
	changed := ChangedPathsSince(before, after)
	if scopeErr := ValidateTesterChangedPaths(contract, changed); scopeErr != nil {
		updated, recordErr := s.RecordActorWriteScopeViolation(id, request.WorkItemID, request.AttemptID, changed, scopeErr.Error())
		if recordErr != nil { return RuntimeState{}, TesterResult{}, recordErr }
		return updated, TesterResult{}, scopeErr
	}
	if runErr != nil { return state, TesterResult{}, runErr }
	if inventory.WorkItemID != request.WorkItemID { return state, TesterResult{}, errors.New("tester inventory work item does not match request") }
	for _, testCase := range inventory.Cases {
		if err := ValidateTesterChangedPaths(contract, []string{testCase.TestFile}); err != nil { return state, TesterResult{}, err }
	}
	reference, err := s.PersistTestCaseInventory(id, inventory)
	if err != nil { return RuntimeState{}, TesterResult{}, err }
	result := TesterResult{ActorResult: ActorResult{SchemaVersion: ActorContractSchemaVersion, RequestID: request.ID, Actor: ActorTester, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Status: ActorResultAccepted, Outputs: []ActorReference{reference}, Summary: "unit tests authored", CompletedAt: time.Now().UTC().Format(time.RFC3339)}, TestCaseInventory: reference}
	return state, result, nil
}

// StableInterfaceInventory normalizes an adapter's discovered interfaces so
// persisted artifacts and handoffs do not vary with discovery order.
func StableInterfaceInventory(workItemID WorkItemID, interfaces []ObservableInterface) ObservableInterfaceInventory {
	copyInterfaces := append([]ObservableInterface(nil), interfaces...)
	sort.SliceStable(copyInterfaces, func(i, j int) bool { return copyInterfaces[i].Symbol < copyInterfaces[j].Symbol })
	return ObservableInterfaceInventory{SchemaVersion: TestAuthoringSchemaVersion, WorkItemID: workItemID, Interfaces: copyInterfaces}
}
