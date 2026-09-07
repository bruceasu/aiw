package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var transitions = map[string]map[string]bool{
	StateCreated: {StateRunning: true, StateActive: true}, StateActive: {StateRunning: true, StatePaused: true, StateCompleted: true}, StateRunning: {StateActive: true, StateFailed: true}, StatePaused: {StateRunning: true}, StateFailed: {StateRunning: true}, StateCompleted: {StateArchived: true}, StateArchived: {StateDeleted: true},
}

func (s *Store) Transition(id, target string) (Status, error) {
	return s.Update(id, func(status *Status) error {
		if status.Session.State == target {
			return nil
		}
		if !transitions[status.Session.State][target] {
			return fmt.Errorf("invalid session transition: %s -> %s", status.Session.State, target)
		}
		status.Session.State = target
		return nil
	})
}
func (s *Store) Archive(id string) error {
	_, err := s.Transition(id, StateArchived)
	if err != nil {
		return err
	}
	source := s.sessionDir(id)
	target := filepath.Join(s.Root, "archive", id)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Rename(source, target)
}
func (s *Store) Handoff(id, focus string) error {
	status, err := s.Load(id)
	if err != nil {
		return err
	}
	memory, _ := s.ReadText(id, "memory.md")
	output := ""
	if status.Result.FinalOutputFile != "" {
		output, _ = s.ReadText(id, filepath.ToSlash(status.Result.FinalOutputFile))
	}
	text := fmt.Sprintf("# Session Handoff\n\n- Session: %s\n- State: %s\n- Backend: %s\n- Workspace: %s\n- Phase: %s\n- Focus: %s\n\n## Memory\n\n%s\n\n## Latest Output\n\n%s\n\n## Next Action\n\nContinue the Session in the declared workspace and report validation.\n", status.Session.ID, status.Session.State, status.Backend.Name, status.Workspace.Path, status.Execution.Phase, strings.TrimSpace(focus), memory, output)
	return s.WriteArtifact(id, "handoff.md", []byte(text))
}
func (s *Store) Validate(id string) error {
	if _, err := s.Load(id); err != nil {
		return err
	}
	return nil
}
func RequireRunnable(status Status) error {
	if status.Session.State == StateCompleted || status.Session.State == StateArchived || status.Session.State == StateDeleted {
		return errors.New("session is not runnable")
	}
	return nil
}
