package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

// GoStaticReviewSandbox is an optional Tester capability. The review receives
// the compiler-produced inventory and has no writable test paths.
type GoStaticReviewSandbox interface {
	ReviewGo(context.Context, TesterRequest, ObservableInterfaceInventory) (GoStaticReview, error)
}

type GoStaticReview struct {
	WorkItemID WorkItemID     `json:"work_item_id"`
	AttemptID  AttemptID      `json:"attempt_id"`
	InputSHA256 string        `json:"input_sha256"`
	Inputs     *ValidationInputs `json:"inputs,omitempty"`
	Sources    []string       `json:"sources"`
	State      EvidenceState  `json:"state"`
	Summary    string         `json:"summary"`
}

func (r GoStaticReview) validate(request TesterRequest, sources []string) error {
	if r.WorkItemID != request.WorkItemID || r.AttemptID != request.AttemptID || r.InputSHA256 == "" || r.InputSHA256 != request.InterfaceInventory.SHA256 || (r.State != EvidencePassed && r.State != EvidenceFailed) || strings.TrimSpace(r.Summary) == "" {
		return errors.New("Go static review requires matching work item, attempt, input digest, outcome, and summary")
	}
	actual := append([]string(nil), r.Sources...)
	sort.Strings(actual)
	if len(actual) != len(sources) { return errors.New("Go static review sources do not match the compiled inventory") }
	for i := range sources { if actual[i] != sources[i] { return errors.New("Go static review sources do not match the compiled inventory") } }
	return nil
}

func (s *Store) loadTesterInterfaces(id TaskID, request TesterRequest) (ObservableInterfaceInventory, error) {
	var inventory ObservableInterfaceInventory
	ref := request.InterfaceInventory
	legacy := filepath.ToSlash(filepath.Join("observable-interfaces", string(request.WorkItemID)+".json"))
	versioned := filepath.ToSlash(filepath.Join("observable-interfaces", string(request.WorkItemID)+"-"+ref.SHA256+".json"))
	if ref.Kind != "observable-interface-inventory" || ref.SHA256 == "" || (ref.Path != legacy && ref.Path != versioned) {
		return inventory, errors.New("tester requires the current work item's interface inventory reference")
	}
	target, err := confinedFile(s.path(id, ""), ref.Path)
	if err != nil { return inventory, err }
	content, err := os.ReadFile(target)
	if err != nil { return inventory, err }
	if contentDigest(content) != ref.SHA256 { return inventory, errors.New("interface inventory digest mismatch") }
	if err := json.Unmarshal(content, &inventory); err != nil { return inventory, err }
	if err := inventory.Validate(); err != nil { return inventory, err }
	if inventory.WorkItemID != request.WorkItemID { return inventory, errors.New("interface inventory work item does not match request") }
	return inventory, nil
}

func testerSources(inventory ObservableInterfaceInventory) (goSources []string, hasNonGo bool) {
	seen := map[string]struct{}{}
	for _, item := range inventory.Interfaces {
		if strings.EqualFold(filepath.Ext(item.Source), ".go") {
			if _, exists := seen[item.Source]; !exists { goSources = append(goSources, item.Source); seen[item.Source] = struct{}{} }
		} else { hasNonGo = true }
	}
	sort.Strings(goSources)
	return goSources, hasNonGo
}

func nonGoTestPaths(patterns []string) []string {
	var allowed []string
	for _, pattern := range patterns {
		// Only an explicit non-Go extension can exclude *_test.go before the
		// sandbox runs. Broad patterns are unsafe for mixed-language work.
		if ext := filepath.Ext(pattern); ext != "" && !strings.EqualFold(ext, ".go") { allowed = append(allowed, pattern) }
	}
	return allowed
}

func (s *Store) persistGoStaticReview(id TaskID, review GoStaticReview) (ActorReference, error) {
	content, err := json.MarshalIndent(review, "", "  ")
	if err != nil { return ActorReference{}, err }
	content = append(content, '\n')
	identity := sha256.Sum256([]byte(string(review.WorkItemID) + "\x00" + string(review.AttemptID) + "\x00" + review.Inputs.Digest()))
	relative := filepath.ToSlash(filepath.Join("static-reviews", hex.EncodeToString(identity[:])+".json"))
	if previous, err := os.ReadFile(s.path(id, filepath.FromSlash(relative))); err == nil {
		if string(previous) != string(content) { return ActorReference{}, errors.New("Go static review conflicts with the existing Attempt artifact") }
	} else if !os.IsNotExist(err) { return ActorReference{}, err }
	if err := atomicWrite(s.path(id, filepath.FromSlash(relative)), content); err != nil { return ActorReference{}, err }
	sum := sha256.Sum256(content)
	return ActorReference{Kind: "go-static-review", Path: relative, SHA256: hex.EncodeToString(sum[:])}, nil
}

func (s *Store) readGoStaticReview(id TaskID, ref ActorReference) (GoStaticReview, error) {
	var review GoStaticReview
	if ref.Kind != "go-static-review" || ref.SHA256 == "" || !strings.HasPrefix(ref.Path, "static-reviews/") { return review, errors.New("Go static review reference is invalid") }
	target, err := confinedFile(s.path(id, ""), ref.Path)
	if err != nil { return review, err }
	content, err := os.ReadFile(target)
	if err != nil { return review, err }
	if contentDigest(content) != ref.SHA256 { return review, errors.New("Go static review digest mismatch") }
	if err := json.Unmarshal(content, &review); err != nil { return review, err }
	return review, nil
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
	sum := sha256.Sum256(content)
	relative := filepath.ToSlash(filepath.Join("observable-interfaces", string(inventory.WorkItemID)+"-"+hex.EncodeToString(sum[:])+".json"))
	if previous, err := os.ReadFile(s.path(id, filepath.FromSlash(relative))); err == nil {
		if string(previous) != string(content) { return ActorReference{}, errors.New("interface inventory conflicts with existing artifact") }
	} else if !os.IsNotExist(err) { return ActorReference{}, err }
	if err := atomicWrite(s.path(id, filepath.FromSlash(relative)), content); err != nil { return ActorReference{}, err }
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

// RunTester uses independent static review for Go interfaces and test-only
// authoring for non-Go interfaces. It does not execute tests.
func (s *Store) RunTester(ctx context.Context, id TaskID, request TesterRequest, contract ImplementationContract, sandbox TesterSandbox, checker ChangedPathChecker) (RuntimeState, TesterResult, error) {
	if err := request.Validate(); err != nil { return RuntimeState{}, TesterResult{}, err }
	if err := contract.Validate(); err != nil { return RuntimeState{}, TesterResult{}, err }
	if sandbox == nil || checker == nil { return RuntimeState{}, TesterResult{}, errors.New("tester requires sandbox and Git changed-path checker") }
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, TesterResult{}, err }
	if request.TaskID != id || state.WriteLease == nil || state.WriteLease.AttemptID != request.AttemptID || state.WriteLease.Workspace != request.Workspace { return RuntimeState{}, TesterResult{}, errors.New("tester request does not own the workspace write lease") }
	interfaces, err := s.loadTesterInterfaces(id, request)
	if err != nil { return state, TesterResult{}, err }
	goSources, hasNonGo := testerSources(interfaces)
	allowedTests := append([]string(nil), contract.AllowedTestPaths...)
	if len(goSources) != 0 && hasNonGo {
		allowedTests = nonGoTestPaths(allowedTests)
		if len(allowedTests) == 0 { return state, TesterResult{}, errors.New("mixed-language Tester requires explicit non-Go test paths") }
	}
	if len(goSources) == 0 || hasNonGo {
		if err := ValidateTesterChangedPaths(ImplementationContract{Status: ImplementationContractReady, AllowedTestPaths: allowedTests}, nil); err != nil { return state, TesterResult{}, err }
	}
	var staticReference ActorReference
	if len(goSources) != 0 {
		reviewer, ok := sandbox.(GoStaticReviewSandbox)
		if !ok { return state, TesterResult{}, errors.New("Go Tester requires an independent static-review sandbox") }
		before, err := checker.Snapshot(request.Workspace)
		if err != nil { return state, TesterResult{}, err }
		inputs, err := CaptureValidationInputs(request.Workspace, goSources, []ActorReference{request.InterfaceInventory}, "go-static-review-v1")
		if err != nil { return state, TesterResult{}, err }
		review, reviewErr := reviewer.ReviewGo(ctx, request, interfaces)
		after, snapshotErr := checker.Snapshot(request.Workspace)
		if snapshotErr != nil { return state, TesterResult{}, snapshotErr }
		if changed := ChangedPathsSince(before, after); len(changed) != 0 {
			updated, recordErr := s.RecordActorWriteScopeViolation(id, request.WorkItemID, request.AttemptID, changed, "Go static review must not write repository files")
			if recordErr != nil { return RuntimeState{}, TesterResult{}, recordErr }
			return updated, TesterResult{}, errors.New("Go static review changed repository files")
		}
		if reviewErr != nil { return state, TesterResult{}, reviewErr }
		if err := review.validate(request, goSources); err != nil { return state, TesterResult{}, err }
		currentInputs, err := CaptureValidationInputs(request.Workspace, goSources, []ActorReference{request.InterfaceInventory}, "go-static-review-v1")
		if err != nil { return state, TesterResult{}, err }
		if applicable, reason := EvidenceApplicability(&inputs, currentInputs, true, true); !applicable { return state, TesterResult{}, fmt.Errorf("Go sources changed during static review: %s", reason) }
		review.Inputs = &inputs
		staticReference, err = s.persistGoStaticReview(id, review)
		if err != nil { return state, TesterResult{}, err }
		if review.State == EvidenceFailed {
			return state, TesterResult{ActorResult: ActorResult{SchemaVersion: ActorContractSchemaVersion, RequestID: request.ID, Actor: ActorTester, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Status: ActorResultFailed, Outputs: []ActorReference{staticReference}, Summary: review.Summary, CompletedAt: time.Now().UTC().Format(time.RFC3339)}, StaticReview: staticReference}, nil
		}
		state, err = s.RecordEvidence(id, Evidence{ID: EvidenceID("go-static-review-"+string(request.AttemptID)+"-"+inputs.Digest()[:12]), WorkItemID: request.WorkItemID, Kind: EvidenceStaticReview, State: EvidencePassed, Reference: staticReference.Path + "#sha256=" + staticReference.SHA256})
		if err != nil { return state, TesterResult{}, err }
		if !hasNonGo {
			return state, TesterResult{ActorResult: ActorResult{SchemaVersion: ActorContractSchemaVersion, RequestID: request.ID, Actor: ActorTester, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Status: ActorResultAccepted, Outputs: []ActorReference{staticReference}, Summary: "Go static review recorded", CompletedAt: time.Now().UTC().Format(time.RFC3339)}, StaticReview: staticReference}, nil
		}
	}
	before, err := checker.Snapshot(request.Workspace)
	if err != nil { return RuntimeState{}, TesterResult{}, err }
	inventory, runErr := sandbox.AuthorTests(ctx, request, allowedTests)
	after, snapshotErr := checker.Snapshot(request.Workspace)
	if snapshotErr != nil { return RuntimeState{}, TesterResult{}, snapshotErr }
	changed := ChangedPathsSince(before, after)
	if len(goSources) != 0 {
		for _, candidate := range changed { if strings.HasSuffix(strings.ToLower(candidate), "_test.go") {
			updated, recordErr := s.RecordActorWriteScopeViolation(id, request.WorkItemID, request.AttemptID, changed, "mixed-language Tester wrote a Go test file")
			if recordErr != nil { return RuntimeState{}, TesterResult{}, recordErr }
			return updated, TesterResult{}, errors.New("mixed-language Tester wrote a Go test file")
		} }
	}
	if scopeErr := ValidateTesterChangedPaths(ImplementationContract{Status: ImplementationContractReady, AllowedTestPaths: allowedTests}, changed); scopeErr != nil {
		updated, recordErr := s.RecordActorWriteScopeViolation(id, request.WorkItemID, request.AttemptID, changed, scopeErr.Error())
		if recordErr != nil { return RuntimeState{}, TesterResult{}, recordErr }
		return updated, TesterResult{}, scopeErr
	}
	if runErr != nil { return state, TesterResult{}, runErr }
	if inventory.WorkItemID != request.WorkItemID { return state, TesterResult{}, errors.New("tester inventory work item does not match request") }
	for _, testCase := range inventory.Cases {
		if len(goSources) != 0 && strings.HasSuffix(strings.ToLower(testCase.TestFile), "_test.go") { return state, TesterResult{}, errors.New("mixed-language Tester declared a Go test file") }
		if err := ValidateTesterChangedPaths(ImplementationContract{Status: ImplementationContractReady, AllowedTestPaths: allowedTests}, []string{testCase.TestFile}); err != nil { return state, TesterResult{}, err }
	}
	reference, err := s.PersistTestCaseInventory(id, inventory)
	if err != nil { return RuntimeState{}, TesterResult{}, err }
	outputs := []ActorReference{reference}
	if staticReference.Kind != "" { outputs = append(outputs, staticReference) }
	result := TesterResult{ActorResult: ActorResult{SchemaVersion: ActorContractSchemaVersion, RequestID: request.ID, Actor: ActorTester, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Status: ActorResultAccepted, Outputs: outputs, Summary: "non-Go tests authored; Go static review recorded", CompletedAt: time.Now().UTC().Format(time.RFC3339)}, TestCaseInventory: reference, StaticReview: staticReference}
	if staticReference.Kind == "" { result.Summary = "unit tests authored" }
	return state, result, nil
}

// StableInterfaceInventory normalizes an adapter's discovered interfaces so
// persisted artifacts and handoffs do not vary with discovery order.
func StableInterfaceInventory(workItemID WorkItemID, interfaces []ObservableInterface) ObservableInterfaceInventory {
	copyInterfaces := append([]ObservableInterface(nil), interfaces...)
	sort.SliceStable(copyInterfaces, func(i, j int) bool { return copyInterfaces[i].Symbol < copyInterfaces[j].Symbol })
	return ObservableInterfaceInventory{SchemaVersion: TestAuthoringSchemaVersion, WorkItemID: workItemID, Interfaces: copyInterfaces}
}
