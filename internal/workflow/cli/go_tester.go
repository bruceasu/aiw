package cli

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

	"aiw/internal/session"
	"aiw/internal/task"
	"aiw/internal/workflow"
)

type supervisedGoTester struct { provider, model string }

func runSupervisedGoTester(id string, store *workflow.Store, prepared *workflow.PreparedAgentRequest, provider, model string) (workflow.TesterResult, error) {
	if prepared == nil || prepared.Compile == nil || prepared.Compile.Request == nil { return workflow.TesterResult{}, errors.New("Go Tester requires a frozen compiler request") }
	state, err := store.Load(prepared.TaskID)
	if err != nil { return workflow.TesterResult{}, err }
	mixed := false
	for _, item := range state.WorkItems { if item.ID == prepared.WorkItemID { mixed = item.Verification == "mixed"; break } }
	paths := append([]string(nil), prepared.Compile.Request.ChangedPaths...)
	sort.Strings(paths)
	var interfaces []workflow.ObservableInterface
	var allowedTests []string
	seen := map[string]bool{}
	for _, changed := range paths {
		if strings.HasSuffix(strings.ToLower(changed), ".go") {
			if strings.HasSuffix(strings.ToLower(changed), "_test.go") || seen[changed] { continue }
			interfaces = append(interfaces, workflow.ObservableInterface{Kind: "source", Symbol: changed, Source: changed})
			seen[changed] = true
		} else if mixed {
			ext := strings.ToLower(filepath.Ext(changed))
			if ext == ".py" || ext == ".java" || ext == ".js" || ext == ".ts" {
				interfaces = append(interfaces, workflow.ObservableInterface{Kind: "source", Symbol: changed, Source: changed})
				allowedTests = append(allowedTests, filepath.ToSlash(filepath.Join(filepath.Dir(changed), "*test*"+ext)))
			}
		}
	}
	if len(interfaces) == 0 { return workflow.TesterResult{}, errors.New("Go Tester cannot review a Work Item without changed Go sources") }
	if mixed && len(allowedTests) == 0 { return workflow.TesterResult{}, errors.New("mixed-language Go Tester requires observed non-Go sources") }
	inventory := workflow.StableInterfaceInventory(prepared.WorkItemID, interfaces)
	reference, err := store.PersistObservableInterfaceInventory(prepared.TaskID, inventory)
	if err != nil { return workflow.TesterResult{}, err }
	if prepared.InputReference == nil { return workflow.TesterResult{}, errors.New("Go Tester requires the frozen input reference") }
	request := workflow.TesterRequest{ActorRequest: workflow.ActorRequest{SchemaVersion: workflow.ActorContractSchemaVersion, ID: "tester-"+string(prepared.AttemptID), Actor: workflow.ActorTester, TaskID: prepared.TaskID, WorkItemID: prepared.WorkItemID, AttemptID: prepared.AttemptID, Workspace: prepared.Workspace, Inputs: []workflow.ActorReference{*prepared.InputReference, prepared.Compile.Request.ImplementationReport}, PreparedAt: time.Now().UTC().Format(time.RFC3339Nano)}, InterfaceInventory: reference}
	contract := workflow.ImplementationContract{Status: workflow.ImplementationContractReady, AllowedProductionPaths: []string{interfaces[0].Source}, AllowedTestPaths: allowedTests}
	_, result, err := store.RunTester(context.Background(), prepared.TaskID, request, contract, supervisedGoTester{provider: provider, model: model}, workflow.GitChangedPathChecker{})
	if err != nil { return result, err }
	if result.Status != workflow.ActorResultAccepted { return result, errors.New("independent Go Tester review did not pass") }
	return result, nil
}

func (s supervisedGoTester) ReviewGo(ctx context.Context, request workflow.TesterRequest, inventory workflow.ObservableInterfaceInventory) (workflow.GoStaticReview, error) {
	sources := make([]string, 0, len(inventory.Interfaces))
	for _, item := range inventory.Interfaces { if strings.HasSuffix(strings.ToLower(item.Source), ".go") { sources = append(sources, item.Source) } }
	sort.Strings(sources)
	prompt := fmt.Sprintf("Review the Go work independently. Read the approved Issue handoff at .ai/tasks/%s/artifacts/requirement-handoff.md, the Feature Design at docs/features/%s.md, the relevant stable specs, the implementation report at %s, and these Go sources: %s. Compare the code with the approved behavior and scope. Do not write files or run commands. Return one JSON object with state (passed or failed) and summary. A passed review must describe the checks performed; report a failed review when evidence is missing or behavior differs.", request.TaskID, request.TaskID, request.Inputs[len(request.Inputs)-1].Path, strings.Join(sources, ", "))
	output, err := s.turn(ctx, request, "review", prompt, true)
	if err != nil { return workflow.GoStaticReview{}, err }
	var answer struct { State workflow.EvidenceState `json:"state"`; Summary string `json:"summary"` }
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &answer); err != nil { return workflow.GoStaticReview{}, fmt.Errorf("decode Go Tester review: %w", err) }
	return workflow.GoStaticReview{WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, InputSHA256: request.InterfaceInventory.SHA256, Sources: sources, State: answer.State, Summary: answer.Summary}, nil
}

func (s supervisedGoTester) AuthorTests(ctx context.Context, request workflow.TesterRequest, allowed []string) (workflow.TestCaseInventory, error) {
	if len(allowed) == 0 { return workflow.TestCaseInventory{}, errors.New("non-Go test authoring requires explicit paths") }
	prompt := fmt.Sprintf("Write only non-Go unit tests for Work Item %s. Read the approved Issue handoff and Feature Design for Task %s and the implementation report at %s. You may write only these test paths: %s. Never create or change a .go file. Do not run tests or other commands. Return one TestCaseInventory JSON object with schema_version=1, work_item_id, and cases.", request.WorkItemID, request.TaskID, request.Inputs[len(request.Inputs)-1].Path, strings.Join(allowed, ", "))
	output, err := s.turn(ctx, request, "tests", prompt, false)
	if err != nil { return workflow.TestCaseInventory{}, err }
	var inventory workflow.TestCaseInventory
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &inventory); err != nil { return inventory, fmt.Errorf("decode non-Go test inventory: %w", err) }
	return inventory, inventory.Validate()
}

func (s supervisedGoTester) turn(ctx context.Context, request workflow.TesterRequest, phase, prompt string, readOnly bool) (string, error) {
	root, err := task.ExecutionWorkspace(request.Workspace)
	if err != nil { return "", err }
	inputs, err := workflow.CaptureValidationInputs(root, []string{"."}, nil, "tester-v1")
	if err != nil { return "", err }
	identity := sha256.Sum256([]byte(string(request.TaskID)+"\x00"+string(request.AttemptID)+"\x00"+phase+"\x00"+inputs.Digest()))
	sessionID := "tester-" + hex.EncodeToString(identity[:12])
	store := session.NewStore("")
	status, err := store.Load(sessionID)
	if errors.Is(err, session.ErrSessionNotFound) {
		status, err = store.Create(sessionID, "Independent Tester", root, "codex", s.model, "Review only the bound Task and Attempt. Follow the current approved requirements.")
		if err != nil { return "", err }
		status, err = store.Update(sessionID, func(current *session.Status) error {
			current.Task = &session.ManagedExecutionRef{SchemaVersion: 1, TaskID: string(request.TaskID), WorkItemID: string(request.WorkItemID), AttemptID: string(request.AttemptID)}
			return nil
		})
	}
	if err != nil { return "", err }
	if status.Task == nil || status.Task.TaskID != string(request.TaskID) || status.Task.WorkItemID != string(request.WorkItemID) || status.Task.AttemptID != string(request.AttemptID) { return "", errors.New("Tester Session identity does not match the Attempt") }
	if status.Result.Status == "completed" && status.Result.FinalOutputFile != "" { return store.ReadText(sessionID, status.Result.FinalOutputFile) }
	if status.Session.LastTurn != 0 { return "", errors.New("Tester Session has an incomplete prior turn") }
	result, err := session.ExecuteFrozenTurn(ctx, store, sessionID, "tester", session.FrozenTurn{Prompt: prompt, ExpectedTurn: 1, ReadOnly: readOnly}, s.provider, s.model, nil)
	if err != nil { return "", err }
	return result.FinalOutput, nil
}
