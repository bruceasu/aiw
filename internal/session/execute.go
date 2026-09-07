package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func ExecuteTurn(ctx context.Context, store *Store, id, phase, prompt, backendName, model string, forceNew bool) (TurnResult, error) {
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
	if backendName == "" {
		backendName = status.Backend.Name
		if backendName == "" {
			backendName = "codex"
		}
	}
	backend, err := BackendFor(backendName, model)
	if err != nil {
		return TurnResult{}, err
	}
	if _, err := store.Update(id, func(current *Status) error {
		current.Session.State = StateRunning
		current.Session.CurrentPhase = phase
		current.Execution.Phase = phase
		current.Execution.LastStartedAt = time.Now().UTC().Format(time.RFC3339)
		current.Backend.Name = backendName
		if model != "" {
			current.Backend.Model = model
		}
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
	_ = store.AppendEvent(id, map[string]interface{}{"type": "turn.completed", "turn": turn, "phase": phase, "backend": backendName, "exit_code": result.ExitCode, "at": time.Now().UTC().Format(time.RFC3339)})
	if runErr != nil {
		return result, runErr
	}
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
