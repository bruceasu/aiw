package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"aiw/internal/ai"
)

// TaskMemoryProjection is installed by the executable composition root. It
// appends Task-owned facts to this turn only; human Session memory is untouched.
var TaskMemoryProjection func(string) (string, error)

func projectTaskMemory(status Status, memory string) (string, error) {
	if TaskMemoryProjection == nil || status.Task == nil || status.Task.TaskID == "" { return memory, nil }
	projection, err := TaskMemoryProjection(status.Task.TaskID)
	if err != nil { return "", err }
	if projection != "" { memory += "\n\n"+projection }
	return memory, nil
}

func ExecuteTurn(ctx context.Context, store *Store, id, phase, prompt, backendName, model string, forceNew bool) (TurnResult, error) {
	return ExecuteTurnWithOverrides(ctx, store, id, phase, prompt, backendName, model, forceNew)
}

// ExecuteTurnWithOverrides applies provider and model overrides to this turn
// only. It deliberately leaves the stored Backend configuration unchanged.
func ExecuteTurnWithOverrides(ctx context.Context, store *Store, id, phase, prompt, providerOverride, modelOverride string, forceNew bool) (TurnResult, error) {
	return ExecuteTurnWithOverridesAndEnvironment(ctx, store, id, phase, prompt, providerOverride, modelOverride, forceNew, nil)
}

// ExecuteTurnWithOverridesAndEnvironment applies one-call provider, model,
// and process-local environment overrides without persisting them in Session
// state.
func ExecuteTurnWithOverridesAndEnvironment(ctx context.Context, store *Store, id, phase, prompt, providerOverride, modelOverride string, forceNew bool, environment []string) (TurnResult, error) {
	return executeTurn(ctx, store, id, phase, prompt, providerOverride, modelOverride, forceNew, environment, nil)
}

// FrozenTurn is the exact, Task-owned input selected before dispatch. It
// prevents a later Session memory update from changing the bound request.
type FrozenTurn struct {
	Prompt string
	OutputSchema map[string]any
	Instructions string
	Memory string
	ExpectedTurn int
	ReadOnly bool
	ReasoningIntensity string
	InvocationObserver ai.InvocationObserver
}

func ExecuteFrozenTurn(ctx context.Context, store *Store, id, phase string, frozen FrozenTurn, provider, model string, environment []string) (TurnResult, error) {
	return executeTurn(ctx, store, id, phase, frozen.Prompt, provider, model, true, environment, &frozen)
}

// RecoverFrozenTurn publishes the already completed result of the exact
// Session turn. It never invokes a Provider and refuses conflicting output.
func RecoverFrozenTurn(store *Store, id string, turn int, result TurnResult) error {
	if store == nil || id == "" || turn <= 0 || result.CompletedAt.IsZero() { return errors.New("frozen turn recovery identity is incomplete") }
	status, err := store.Load(id)
	if err != nil { return err }
	finalName, eventsName := fmt.Sprintf("outputs/%04d-final.txt", turn), fmt.Sprintf("outputs/%04d-events.jsonl", turn)
	final, finalErr := store.ReadText(id, finalName)
	events, eventsErr := store.ReadText(id, eventsName)
	if status.Session.LastTurn == turn {
		if status.Result.Status != "completed" && status.Result.Status != "failed" { return errors.New("Session turn was advanced without a terminal result") }
		if finalErr != nil || eventsErr != nil || final != result.FinalOutput || events != string(result.Events) { return errors.New("persisted Session turn conflicts with the Codex invocation receipt") }
		return nil
	}
	if status.Session.LastTurn+1 != turn || status.Session.State != StateRunning { return errors.New("Session is not awaiting this frozen turn recovery") }
	if finalErr == nil && final != result.FinalOutput || eventsErr == nil && events != string(result.Events) { return errors.New("partial Session outputs conflict with the Codex invocation receipt") }
	if finalErr != nil && !errors.Is(finalErr, os.ErrNotExist) { return finalErr }
	if eventsErr != nil && !errors.Is(eventsErr, os.ErrNotExist) { return eventsErr }
	if err := SaveTurnResult(store, status, result); err != nil { return err }
	_, err = store.Update(id, func(current *Status) error {
		if current.Session.LastTurn == turn {
			if current.Result.Status != "completed" && current.Result.Status != "failed" { return errors.New("Session turn has a conflicting terminal state") }
			return nil
		}
		if current.Session.LastTurn+1 != turn || current.Session.State != StateRunning { return errors.New("Session changed while recovering its frozen turn") }
		current.Session.LastTurn = turn
		current.Execution.LastExitCode = &result.ExitCode
		current.Execution.LastCompletedAt = result.CompletedAt.UTC().Format(time.RFC3339)
		current.Backend.ThreadID = result.ThreadID
		current.Result.FinalOutputFile = finalName
		current.Result.Status = "completed"
		current.Session.State = StateActive
		if result.ExitCode != 0 { current.Result.Status, current.Session.State, current.Result.ErrorMessage = "failed", StateFailed, result.FinalOutput }
		return nil
	})
	if err != nil { return err }
	return nil
}

func ComposePrompt(instructions, memory, phase, prompt string) string {
	return fmt.Sprintf("[Persistent Execution Instructions]\n\n%s\n\n[Session Memory]\n\n%s\n\n[Current Phase]\n\n%s\n\n[Current Task]\n\n%s\n", strings.TrimSpace(instructions), strings.TrimSpace(memory), phase, strings.TrimSpace(prompt))
}

func executeTurn(ctx context.Context, store *Store, id, phase, prompt, providerOverride, modelOverride string, forceNew bool, environment []string, frozen *FrozenTurn) (TurnResult, error) {
	status, err := store.Load(id)
	if err != nil {
		return TurnResult{}, err
	}
	if err := RequireRunnable(status); err != nil {
		return TurnResult{}, err
	}
	if strings.TrimSpace(prompt) == "" {
		return TurnResult{}, fmt.Errorf("prompt is empty")
	}
	var instructions, memory string
	if frozen != nil {
		if frozen.ExpectedTurn != status.Session.LastTurn+1 { return TurnResult{}, fmt.Errorf("frozen input does not match the next Session turn") }
		instructions, memory = frozen.Instructions, frozen.Memory
	} else {
		instructions, err = store.ReadText(id, status.Instructions.File)
		if err != nil { return TurnResult{}, err }
		memory, err = store.ReadText(id, status.Instructions.MemoryFile)
		if err != nil { return TurnResult{}, err }
		memory, err = projectTaskMemory(status, memory)
		if err != nil { return TurnResult{}, err }
	}
	if phase == "" {
		phase = status.Session.CurrentPhase
		if phase == "" {
			phase = "interactive"
		}
	}
	composed := ComposePrompt(instructions, memory, phase, prompt)
	if frozen != nil { composed = frozen.Prompt }
	turn := status.Session.LastTurn + 1
	if err := store.SavePrompt(id, turn, phase, composed); err != nil {
		return TurnResult{}, err
	}
	cfg, err := ai.ResolveConfig(status.Backend.Name, status.Backend.Model, providerOverride, modelOverride)
	if err != nil {
		return TurnResult{}, err
	}
	if cfg.Name == "" || cfg.Name == "auto" {
		cfg.Name = "codex"
	}
	backend, err := ai.NewProvider(cfg)
	if err != nil {
		return TurnResult{}, err
	}
	if _, err := store.Update(id, func(current *Status) error {
		current.Session.State = StateRunning
		current.Session.CurrentPhase = phase
		current.Execution.Phase = phase
		current.Execution.LastStartedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	}); err != nil {
		return TurnResult{}, err
	}
	var observer ai.InvocationObserver
	if frozen != nil { observer = frozen.InvocationObserver }
	reasoningIntensity := ""
	if frozen != nil { reasoningIntensity = frozen.ReasoningIntensity }
	var outputSchema map[string]any
	if frozen != nil { outputSchema = frozen.OutputSchema }
	result, runErr := backend.Generate(ctx, TurnRequest{SessionID: id, Prompt: composed, OutputSchema: outputSchema, Workspace: status.Workspace.Path, ThreadID: status.Backend.ThreadID, Model: modelOverride, ReasoningIntensity: reasoningIntensity, Instructions: instructions, Memory: memory, Phase: phase, TurnNumber: turn, OutputDir: store.sessionDir(id) + "/outputs", ForceNewThread: forceNew, Environment: environment, ReadOnly: frozen != nil && frozen.ReadOnly, InvocationObserver: observer})
	ensureTurnUsage(&result, cfg.Name, cfg.Model)
	if runErr != nil && result.ExitCode == 0 {
		result.ExitCode = 1
	}
	if err := SaveTurnResult(store, status, result); err != nil {
		return result, err
	}
	_, updateErr := store.Update(id, func(current *Status) error {
		current.Session.LastTurn = turn
		current.Execution.LastExitCode = &result.ExitCode
		current.Execution.LastCompletedAt = time.Now().UTC().Format(time.RFC3339)
		current.Backend.ThreadID = result.ThreadID
		current.Result.FinalOutputFile = fmt.Sprintf("outputs/%04d-final.txt", turn)
		current.Result.Status = "completed"
		if result.ExitCode != 0 {
			current.Session.State = StateFailed
			current.Result.Status = "failed"
			current.Result.ErrorMessage = result.FinalOutput
		} else {
			current.Session.State = StateActive
			current.Result.ErrorMessage = ""
		}
		return nil
	})
	if updateErr != nil {
		return result, updateErr
	}
	_ = store.AppendEvent(id, map[string]interface{}{"type": "turn.completed", "turn": turn, "phase": phase, "backend": cfg.Name, "exit_code": result.ExitCode, "at": time.Now().UTC().Format(time.RFC3339)})
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}

// ExecuteInteractive starts one provider-owned interactive session while
// preserving the same Session bookkeeping as a bounded turn.
func ExecuteInteractive(ctx context.Context, store *Store, id, phase, prompt, backendName, model string, forceNew bool) (TurnResult, error) {
	return ExecuteInteractiveWithOverrides(ctx, store, id, phase, prompt, backendName, model, forceNew)
}

// ExecuteInteractiveWithOverrides applies one-call overrides without changing
// the persisted Session provider or model.
func ExecuteInteractiveWithOverrides(ctx context.Context, store *Store, id, phase, prompt, providerOverride, modelOverride string, forceNew bool) (TurnResult, error) {
	return ExecuteInteractiveWithOverridesAndEnvironment(ctx, store, id, phase, prompt, providerOverride, modelOverride, forceNew, nil)
}

// ExecuteInteractiveWithOverridesAndEnvironment applies one-call provider,
// model, and process-local environment overrides without persisting them in
// Session state.
func ExecuteInteractiveWithOverridesAndEnvironment(ctx context.Context, store *Store, id, phase, prompt, providerOverride, modelOverride string, forceNew bool, environment []string) (TurnResult, error) {
	status, err := store.Load(id)
	if err != nil { return TurnResult{}, err }
	if err := RequireRunnable(status); err != nil { return TurnResult{}, err }
	if strings.TrimSpace(prompt) == "" { return TurnResult{}, fmt.Errorf("prompt is empty") }
	instructions, err := store.ReadText(id, status.Instructions.File)
	if err != nil { return TurnResult{}, err }
	memory, err := store.ReadText(id, status.Instructions.MemoryFile)
	if err != nil { return TurnResult{}, err }
	memory, err = projectTaskMemory(status, memory)
	if err != nil { return TurnResult{}, err }
	if phase == "" { phase = status.Session.CurrentPhase; if phase == "" { phase = "interactive" } }
	composed := fmt.Sprintf("[Persistent Execution Instructions]\n\n%s\n\n[Session Memory]\n\n%s\n\n[Current Phase]\n\n%s\n\n[Current Task]\n\n%s\n", strings.TrimSpace(instructions), strings.TrimSpace(memory), phase, strings.TrimSpace(prompt))
	turn := status.Session.LastTurn + 1
	if err := store.SavePrompt(id, turn, phase, composed); err != nil { return TurnResult{}, err }
	cfg, err := ai.ResolveConfig(status.Backend.Name, status.Backend.Model, providerOverride, modelOverride)
	if err != nil { return TurnResult{}, err }
	if cfg.Name == "" || cfg.Name == "auto" { cfg.Name = "codex" }
	backend, err := ai.NewProvider(cfg)
	if err != nil { return TurnResult{}, err }
	if _, err := store.Update(id, func(current *Status) error {
		current.Session.State = StateRunning
		current.Session.CurrentPhase = phase
		current.Execution.Phase = phase
		current.Execution.LastStartedAt = time.Now().UTC().Format(time.RFC3339)
		return nil
	}); err != nil { return TurnResult{}, err }
	result, runErr := backend.Interactive(ctx, TurnRequest{SessionID: id, Prompt: composed, Workspace: status.Workspace.Path, ThreadID: status.Backend.ThreadID, Instructions: instructions, Memory: memory, Phase: phase, TurnNumber: turn, OutputDir: store.sessionDir(id) + "/outputs", ForceNewThread: forceNew, Model: cfg.Model, Environment: environment})
	ensureTurnUsage(&result, cfg.Name, cfg.Model)
	saveErr := SaveTurnResult(store, status, result)
	executionErr := runErr
	if saveErr != nil {
		executionErr = errors.Join(executionErr, saveErr)
	}
	_, updateErr := store.Update(id, func(current *Status) error {
		current.Session.LastTurn = turn
		current.Execution.LastExitCode = &result.ExitCode
		current.Execution.LastCompletedAt = time.Now().UTC().Format(time.RFC3339)
		current.Backend.ThreadID = result.ThreadID
		current.Result.Status = "completed"
		if result.ExitCode != 0 || executionErr != nil { current.Session.State = StateFailed; current.Result.Status = "failed" } else { current.Session.State = StateActive }
		return nil
	})
	_ = store.AppendEvent(id, map[string]interface{}{"type": "interactive.completed", "turn": turn, "phase": phase, "backend": cfg.Name, "exit_code": result.ExitCode, "at": time.Now().UTC().Format(time.RFC3339)})
	if updateErr != nil { return result, updateErr }
	if executionErr != nil { return result, executionErr }
	return result, nil
}

func ensureTurnUsage(result *TurnResult, provider, model string) {
	if result.Usage == nil {
		unknownInt := func() ai.UsageField[int64] { return ai.UsageField[int64]{State: ai.UsageFieldUnknown} }
		unknownNumber := ai.UsageField[json.Number]{State: ai.UsageFieldUnknown}
		unknownString := ai.UsageField[string]{State: ai.UsageFieldUnknown}
		result.Usage = &ai.UsageEnvelope{
			Version: ai.UsageEnvelopeVersion, Availability: ai.UsageFieldUnknown,
			InputTokens: unknownInt(), CachedInputTokens: unknownInt(),
			OutputTokens: unknownInt(), ReasoningOutputTokens: unknownInt(), TotalTokens: unknownInt(),
			CostAmount: unknownNumber, CostCurrency: unknownString,
		}
	}
	if result.Usage.CachedInputTokens.State == "" { result.Usage.CachedInputTokens.State = ai.UsageFieldUnknown }
	if result.Usage.ReasoningOutputTokens.State == "" { result.Usage.ReasoningOutputTokens.State = ai.UsageFieldUnknown }
	if result.Usage.Provider == "" {
		result.Usage.Provider = provider
	}
	if result.Usage.Model == "" {
		result.Usage.Model = model
	}
	if result.Usage.StartedAt.IsZero() {
		result.Usage.StartedAt = result.StartedAt
	}
	if result.Usage.CompletedAt.IsZero() {
		result.Usage.CompletedAt = result.CompletedAt
	}
}

func DecodeThreadID(events []byte) string {
	var id string
	for _, line := range strings.Split(string(events), "\n") {
		var event map[string]interface{}
		if json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		if value, ok := event["thread_id"].(string); ok && value != "" {
			id = value
		}
		if value, ok := event["session_id"].(string); ok && value != "" {
			id = value
		}
	}
	return id
}
