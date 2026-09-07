package session

import (
	"aiw/internal/ai"
	"encoding/json"
	"path/filepath"
)

const (
	StateCreated   = "created"
	StateActive    = "active"
	StateRunning   = "running"
	StatePaused    = "paused"
	StateFailed    = "failed"
	StateCompleted = "completed"
	StateArchived  = "archived"
	StateDeleted   = "deleted"
)

type Status struct {
	SchemaVersion int                    `json:"schema_version"`
	Session       SessionInfo            `json:"session"`
	Backend       BackendInfo            `json:"backend"`
	Workspace     WorkspaceInfo          `json:"workspace"`
	Instructions  InstructionsInfo       `json:"instructions"`
	Execution     ExecutionInfo          `json:"execution"`
	Result        ResultInfo             `json:"result"`
	Task          map[string]interface{} `json:"task,omitempty"`
	Extra         map[string]interface{} `json:"-"`
}

type SessionInfo struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	State        string `json:"state"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	CurrentPhase string `json:"current_phase,omitempty"`
	LastTurn     int    `json:"last_turn"`
}

type BackendInfo struct {
	Name     string `json:"name"`
	ThreadID string `json:"thread_id,omitempty"`
	Model    string `json:"model,omitempty"`
	Profile  string `json:"profile,omitempty"`
}

type WorkspaceInfo struct {
	Path   string `json:"path"`
	Branch string `json:"branch,omitempty"`
}

type InstructionsInfo struct {
	File       string `json:"file"`
	MemoryFile string `json:"memory_file"`
}

type ExecutionInfo struct {
	Phase           string `json:"phase,omitempty"`
	LastExitCode    *int   `json:"last_exit_code,omitempty"`
	LastStartedAt   string `json:"last_started_at,omitempty"`
	LastCompletedAt string `json:"last_completed_at,omitempty"`
}

type ResultInfo struct {
	Status          string `json:"status"`
	FinalOutputFile string `json:"final_output_file,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

type TurnRequest = ai.Request
type TurnResult = ai.Response
type Backend = ai.Provider

func (s Status) MarshalJSON() ([]byte, error) {
	type alias Status
	return json.Marshal(alias(s))
}

func (s Status) SessionDir(root string) string { return filepath.Join(root, "sessions", s.Session.ID) }
