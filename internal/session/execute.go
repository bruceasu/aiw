package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"aiw/internal/ai"
)

func ExecuteTurn(ctx context.Context, store *Store, id, phase, prompt, backendName, model string, forceNew bool) (TurnResult, error) {
	return ExecuteTurnWithOverrides(ctx, store, id, phase, prompt, backendName, model, forceNew)
}

// ExecuteTurnWithOverrides applies provider and model overrides to this turn
// only. It deliberately leaves the stored Backend configuration unchanged.
func ExecuteTurnWithOverrides(ctx context.Context, store *Store, id, phase, prompt, providerOverride, modelOverride string, forceNew bool) (TurnResult, error) {
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
	instructions, err := store.ReadText(id, status.Instructions.File)
	if err != nil {
		return TurnResult{}, err
	}
	memory, err := store.ReadText(id, status.Instructions.MemoryFile)
	if err != nil {
		return TurnResult{}, err
	}
	if phase == "" {
		phase = status.Session.CurrentPhase
		if phase == "" {
			phase = "interactive"
		}
	}
	composed := fmt.Sprintf("[Persistent Execution Instructions]\n\n%s\n\n[Session Memory]\n\n%s\n\n[Current Phase]\n\n%s\n\n[Current Task]\n\n%s\n", strings.TrimSpace(instructions), strings.TrimSpace(memory), phase, strings.TrimSpace(prompt))
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
	result, runErr := backend.Generate(ctx, TurnRequest{SessionID: id, Prompt: composed, Workspace: status.Workspace.Path, ThreadID: status.Backend.ThreadID, Instructions: instructions, Memory: memory, Phase: phase, TurnNumber: turn, OutputDir: store.sessionDir(id) + "/outputs", ForceNewThread: forceNew})
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
	status, err := store.Load(id)
	if err != nil { return TurnResult{}, err }
	if err := RequireRunnable(status); err != nil { return TurnResult{}, err }
	if strings.TrimSpace(prompt) == "" { return TurnResult{}, fmt.Errorf("prompt is empty") }
	instructions, err := store.ReadText(id, status.Instructions.File)
	if err != nil { return TurnResult{}, err }
	memory, err := store.ReadText(id, status.Instructions.MemoryFile)
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
	result, runErr := backend.Interactive(ctx, TurnRequest{SessionID: id, Prompt: composed, Workspace: status.Workspace.Path, ThreadID: status.Backend.ThreadID, Instructions: instructions, Memory: memory, Phase: phase, TurnNumber: turn, OutputDir: store.sessionDir(id) + "/outputs", ForceNewThread: forceNew, Model: cfg.Model})
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
