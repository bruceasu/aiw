package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var goDiagnosticPattern = regexp.MustCompile(`^(.+):(\d+)(?::(\d+))?:\s*(.+)$`)

// MaxConsecutiveCompileFailures bounds automatic Coder repair handoffs for a
// single Work Item. It is deliberately independent from no-progress retries:
// compiler feedback is a repair loop, not a new implementation Attempt.
const MaxConsecutiveCompileFailures = 3

// CompileAdapter is a fully selected compile-only command. It intentionally
// has no shell-string field: adapters supply an executable and arguments.
type CompileAdapter struct {
	Kind string   `json:"kind"`
	Name string   `json:"name"`
	Path string   `json:"path,omitempty"`
	Command string `json:"command"`
	Args []string `json:"args,omitempty"`
}

// CompileDiagnostic is a portable, structured representation of compiler
// output. Source, line, and column are populated when a Go-style diagnostic
// can be recognized; Raw always preserves the original non-empty line.
type CompileDiagnostic struct {
	Target string `json:"target,omitempty"`
	Severity string `json:"severity"`
	Source   string `json:"source,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
	Raw      string `json:"raw"`
}

type CompileDiagnostics struct {
	Adapter     CompileAdapter      `json:"adapter"`
	ExitCode    int                 `json:"exit_code"`
	Diagnostics []CompileDiagnostic `json:"diagnostics"`
	RecordedAt  string               `json:"recorded_at"`
}

// CompileCommandRunner lets platform adapters provide process isolation while
// keeping adapter selection deterministic and independently testable.
type CompileCommandRunner interface {
	Run(context.Context, string, CompileAdapter) (string, int, error)
}

type ExecCompileCommandRunner struct{}

func (ExecCompileCommandRunner) Run(ctx context.Context, workspace string, adapter CompileAdapter) (string, int, error) {
	command, args := adapter.Command, append([]string(nil), adapter.Args...)
	if adapter.Kind == "script" {
		switch strings.ToLower(filepath.Ext(adapter.Path)) {
		case ".bat", ".cmd":
			if runtime.GOOS != "windows" { return "", -1, errors.New("Windows compile script cannot run on this platform") }
			command, args = "cmd.exe", []string{"/d", "/s", "/c", adapter.Path}
		case ".sh":
			command, args = "sh", []string{adapter.Path}
		case ".ps1":
			command, args = "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-File", adapter.Path}
		case ".py":
			command, args = "python", []string{adapter.Path}
		}
	}
	process := exec.CommandContext(ctx, command, args...)
	process.Dir = workspace
	output, err := process.CombinedOutput()
	if err == nil { return string(output), 0, nil }
	var exitError *exec.ExitError
	if errors.As(err, &exitError) { return string(output), exitError.ExitCode(), err }
	return string(output), -1, err
}

// SelectCompileAdapter prefers explicit compile scripts in a fixed order. If
// none exists, the registered Go adapter is the narrowest local compiler for
// this repository. No test or package/build script is ever selected.
func SelectCompileAdapter(workspace string) (CompileAdapter, error) {
	if strings.TrimSpace(workspace) == "" { return CompileAdapter{}, errors.New("compile workspace is required") }
	for _, relative := range []string{"scripts/compile.py", "scripts/compile.bat", "scripts/compile.cmd", "scripts/compile.ps1", "scripts/compile.sh", "compile.py", "compile.bat", "compile.cmd", "compile.ps1", "compile.sh"} {
		candidate := filepath.Join(workspace, filepath.FromSlash(relative))
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() { return CompileAdapter{Kind: "script", Name: relative, Path: candidate}, nil }
		if err != nil && !errors.Is(err, os.ErrNotExist) { return CompileAdapter{}, fmt.Errorf("inspect compile script %s: %w", relative, err) }
	}
	if info, err := os.Stat(filepath.Join(workspace, "go.mod")); err == nil && !info.IsDir() {
		return CompileAdapter{Kind: "go", Name: "go-build", Command: "go", Args: []string{"build", "./..."}}, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) { return CompileAdapter{}, fmt.Errorf("inspect Go module: %w", err) }
	return CompileAdapter{}, errors.New("no registered compile adapter or compile script found")
}

func (d CompileDiagnostics) Validate() error {
	if d.Adapter.Kind == "" || d.Adapter.Name == "" || d.RecordedAt == "" { return errors.New("compile diagnostics require adapter and recorded time") }
	for _, diagnostic := range d.Diagnostics {
		if strings.TrimSpace(diagnostic.Severity) == "" || strings.TrimSpace(diagnostic.Message) == "" || strings.TrimSpace(diagnostic.Raw) == "" { return errors.New("compile diagnostic requires severity, message, and raw output") }
	}
	return nil
}

// RunCompiler executes the persisted plan, then records diagnostics and both
// repair counters before returning. Recovery can reuse the completed result.
func (s *Store) RunCompiler(ctx context.Context, id TaskID, request CompilerRequest, runner CompileCommandRunner) (CompilerResult, CompileDiagnostics, error) {
	if runner == nil {
		return CompilerResult{}, CompileDiagnostics{}, errors.New("compiler requires a command runner")
	}
	request, err := s.loadCompilerRequest(id, request)
	if err != nil {
		return CompilerResult{}, CompileDiagnostics{}, err
	}
	compileRoot := request.CompileRoot
	if compileRoot == "" {
		compileRoot = request.Workspace
	}
	compileRoot, err = filepath.Abs(compileRoot)
	if err != nil {
		return CompilerResult{}, CompileDiagnostics{}, err
	}

	diagnostics, unavailable, runErr := runCompilePlan(ctx, compileRoot, request, runner)
	status, summary := ActorResultAccepted, "compile-only validation passed"
	if runErr != nil {
		status, summary = ActorResultFailed, "compile-only validation failed"
	}
	if unavailable {
		status = ActorResultBlocked
		if _, err := s.OpenCompilerGate(id, request.WorkItemID, "compile-target-unavailable", runErr.Error()); err != nil {
			return CompilerResult{}, diagnostics, err
		}
	}
	reference, persistErr := s.PersistCompileDiagnostics(id, request.WorkItemID, diagnostics)
	if persistErr != nil {
		return CompilerResult{}, diagnostics, persistErr
	}
	result := CompilerResult{ActorResult: ActorResult{SchemaVersion: ActorContractSchemaVersion, RequestID: request.ID, Actor: ActorCompiler, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Status: status, Outputs: []ActorReference{reference}, Summary: summary, CompletedAt: diagnostics.RecordedAt}, Diagnostics: reference}
	if _, err := s.RecordCompilerRequestOutcome(id, request, result); err != nil {
		return CompilerResult{}, diagnostics, err
	}
	return result, diagnostics, runErr
}

func (s *Store) loadCompilerRequest(id TaskID, request CompilerRequest) (CompilerRequest, error) {
	if err := request.Validate(); err != nil {
		return CompilerRequest{}, err
	}
	if request.TaskID != id {
		return CompilerRequest{}, errors.New("compiler request Task does not match runtime Task")
	}
	state, err := s.Load(id)
	if err != nil {
		return CompilerRequest{}, err
	}
	if prepared := state.Automation.PreparedRequest; prepared != nil && prepared.Compile != nil && prepared.Compile.Request != nil {
		persisted := prepared.Compile.Request
		if persisted.ID != request.ID || persisted.TaskID != request.TaskID || persisted.WorkItemID != request.WorkItemID || persisted.AttemptID != request.AttemptID || persisted.Workspace != request.Workspace {
			return CompilerRequest{}, errors.New("compiler request differs from the persisted request")
		}
		request = *persisted
	} else if request.Plan != nil {
		return CompilerRequest{}, errors.New("compiler request must be persisted before execution")
	}
	if err := request.Validate(); err != nil {
		return CompilerRequest{}, err
	}
	if state.WriteLease == nil || state.WriteLease.AttemptID != request.AttemptID || state.WriteLease.Workspace != request.Workspace {
		return CompilerRequest{}, errors.New("compiler request does not own the workspace write lease")
	}
	if request.Plan == nil {
		err := errors.New("compile-plan-missing: regenerate routing and prepare a fresh request")
		_, gateErr := s.OpenCompilerGate(id, request.WorkItemID, "compile-plan-missing", err.Error())
		return CompilerRequest{}, errors.Join(err, gateErr)
	}
	return request, nil
}

func runCompilePlan(ctx context.Context, root string, request CompilerRequest, runner CompileCommandRunner) (CompileDiagnostics, bool, error) {
	diagnostics := CompileDiagnostics{
		Adapter:    CompileAdapter{Kind: "plan", Name: "request-compile-plan"},
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	var failures []error
	targets := SelectCompileTargets(*request.Plan, request.ChangedPaths)
	unavailable := !request.Plan.Available || len(targets) == 0
	if unavailable {
		failures = append(failures, fmt.Errorf("compile plan unavailable: %s; add or repair a repository compile script", request.Plan.Reason))
	}
	for _, target := range targets {
		adapter, resolveErr := ResolveCompileTarget(root, target)
		output, exitCode, runErr := "", -1, resolveErr
		if resolveErr == nil {
			output, exitCode, runErr = runner.Run(ctx, root, adapter)
		}
		if runErr == nil && exitCode != 0 {
			runErr = fmt.Errorf("compiler exited with code %d", exitCode)
		}
		if runErr != nil {
			unavailable = unavailable || exitCode == -1
			failures = append(failures, fmt.Errorf("%s: %w", target.Name, runErr))
			if strings.TrimSpace(output) == "" {
				output = runErr.Error()
			}
		}
		entries := parseCompileDiagnostics(output, runErr != nil)
		for index := range entries {
			entries[index].Target = target.Name
		}
		diagnostics.Diagnostics = append(diagnostics.Diagnostics, entries...)
	}
	runErr := errors.Join(failures...)
	if runErr != nil {
		diagnostics.ExitCode = 1
		if len(diagnostics.Diagnostics) == 0 {
			diagnostics.Diagnostics = parseCompileDiagnostics(runErr.Error(), true)
		}
	}
	return diagnostics, unavailable, runErr
}

// RecordCompilerOutcome tracks the distinct, bounded compiler-repair loop.
// A passing compiler resets only this loop. On the third consecutive failure
// it opens a validation Gate, leaving the enclosing Attempt untouched for the
// supervisor to close through its normal structured outcome path.
func (s *Store) RecordCompilerOutcome(id TaskID, workItemID WorkItemID, diagnostics ActorReference, succeeded bool) (RuntimeState, error) {
	return s.recordCompilerOutcome(id, workItemID, diagnostics, succeeded, nil)
}

func (s *Store) recordCompilerOutcome(id TaskID, workItemID WorkItemID, diagnostics ActorReference, succeeded bool, result *CompilerResult) (RuntimeState, error) {
	if workItemID == "" {
		return RuntimeState{}, errors.New("compiler outcome work item is required")
	}
	if err := validateActorReferences([]ActorReference{diagnostics}); err != nil {
		return RuntimeState{}, fmt.Errorf("compiler outcome diagnostics: %w", err)
	}
	eventType := "compiler.failed"
	if succeeded {
		eventType = "compiler.passed"
	}
	if result != nil && result.Status == ActorResultBlocked {
		eventType = "compiler.gated"
	}
	recorded := false
	updated, err := s.UpdateWithEvent(id, Event{Type: eventType, WorkItemID: workItemID, Detail: diagnostics.Path}, func(state *RuntimeState) error {
		if result != nil {
			fresh, err := recordPreparedCompilerOutcome(state, result)
			if err != nil || !fresh {
				return err
			}
			if result.Status == ActorResultBlocked {
				return nil
			}
		}
		for index := range state.WorkItems {
			item := &state.WorkItems[index]
			if item.ID != workItemID {
				continue
			}
			recorded = true
			if succeeded {
				item.CompileFailureCount = 0
				return nil
			}
			item.CompileFailureCount++
			item.LastOutputReference = diagnostics.Path
			if item.CompileFailureCount < MaxConsecutiveCompileFailures {
				return nil
			}
			gateID := GateID("compiler-repair-limit-" + string(workItemID))
			reason := fmt.Sprintf("compiler failed %d consecutive times; inspect %s, repair manually, then resolve this gate", item.CompileFailureCount, diagnostics.Path)
			for gateIndex := range state.Gates {
				if state.Gates[gateIndex].ID == gateID {
					state.Gates[gateIndex].State = GateOpen
					state.Gates[gateIndex].Reason = reason
					return nil
				}
			}
			state.Gates = append(state.Gates, Gate{ID: gateID, WorkItemID: workItemID, Kind: GateValidation, State: GateOpen, Reason: reason})
			return nil
		}
		return fmt.Errorf("unknown work item %s", workItemID)
	})
	if err != nil || succeeded || !recorded {
		return updated, err
	}
	if _, err := s.PersistFailureReport(updated, failureReportForCompiler(updated, workItemID, diagnostics)); err != nil {
		return updated, err
	}
	return updated, nil
}

func parseCompileDiagnostics(output string, failed bool) []CompileDiagnostic {
	severity := "info"
	if failed { severity = "error" }
	var diagnostics []CompileDiagnostic
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }
		diagnostic := CompileDiagnostic{Severity: severity, Message: line, Raw: line}
		if match := goDiagnosticPattern.FindStringSubmatch(line); match != nil {
			diagnostic.Source, diagnostic.Message = match[1], match[4]
			diagnostic.Line, _ = strconv.Atoi(match[2])
			if match[3] != "" { diagnostic.Column, _ = strconv.Atoi(match[3]) }
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	if failed && len(diagnostics) == 0 { diagnostics = append(diagnostics, CompileDiagnostic{Severity: "error", Message: "compiler exited without diagnostic output", Raw: "compiler exited without diagnostic output"}) }
	return diagnostics
}

// PersistCompileDiagnostics writes an immutable, Task-local failure record
// before it becomes a Coder repair input.
func (s *Store) PersistCompileDiagnostics(id TaskID, workItemID WorkItemID, diagnostics CompileDiagnostics) (ActorReference, error) {
	if err := diagnostics.Validate(); err != nil { return ActorReference{}, err }
	if workItemID == "" { return ActorReference{}, errors.New("compile diagnostics work item is required") }
	content, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil { return ActorReference{}, fmt.Errorf("encode compile diagnostics: %w", err) }
	content = append(content, '\n')
	relative := filepath.ToSlash(filepath.Join("compile-diagnostics", string(workItemID)+"-"+diagnostics.RecordedAt+".json"))
	// RFC3339 contains colons, so use a portable filename while retaining the
	// timestamp inside the artifact.
	relative = strings.ReplaceAll(relative, ":", "-")
	if err := atomicWrite(s.path(id, filepath.FromSlash(relative)), content); err != nil { return ActorReference{}, err }
	sum := sha256.Sum256(content)
	return ActorReference{Kind: "compile-diagnostics", Path: relative, SHA256: hex.EncodeToString(sum[:])}, nil
}

// PrepareCoderRepairRequest turns a persisted compiler failure into the next
// Coder handoff within the same Attempt and compiler-only failure budget.
func PrepareCoderRepairRequest(state RuntimeState, requestID string, compiler CompilerRequest, result CompilerResult, implementationContract, diagnostics ActorReference) (CoderRequest, error) {
	if err := compiler.Validate(); err != nil { return CoderRequest{}, err }
	if err := result.Validate(); err != nil { return CoderRequest{}, err }
	if result.Status != ActorResultFailed || result.RequestID != compiler.ID { return CoderRequest{}, errors.New("coder repair requires the matching failed compiler result") }
	if err := validateActorReferences([]ActorReference{implementationContract, diagnostics}); err != nil { return CoderRequest{}, fmt.Errorf("coder repair inputs: %w", err) }
	for _, item := range state.WorkItems {
		if item.ID == compiler.WorkItemID {
			prepared := state.Automation.PreparedRequest
			if prepared == nil || prepared.AttemptID != compiler.AttemptID || prepared.Compile == nil || prepared.Compile.Failures >= CompilerFailureLimit { return CoderRequest{}, fmt.Errorf("work item %s has no compiler repair budget in this Attempt", item.ID) }
			if item.CompileFailureCount >= MaxConsecutiveCompileFailures {
				return CoderRequest{}, fmt.Errorf("work item %s exhausted its consecutive compiler repair limit; resolve the compiler-repair gate before another repair", item.ID)
			}
			return CoderRequest{ActorRequest: ActorRequest{SchemaVersion: ActorContractSchemaVersion, ID: requestID, Actor: ActorCoder, TaskID: compiler.TaskID, WorkItemID: compiler.WorkItemID, AttemptID: compiler.AttemptID, Workspace: compiler.Workspace, Inputs: []ActorReference{diagnostics}, PreparedAt: time.Now().UTC().Format(time.RFC3339)}, ImplementationContract: implementationContract, PriorEvidence: []ActorReference{diagnostics}}, nil
		}
	}
	return CoderRequest{}, fmt.Errorf("unknown work item %s", compiler.WorkItemID)
}
