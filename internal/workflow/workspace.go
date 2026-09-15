package workflow

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const ParentWriteFenceGateID GateID = "parent_write_fence_violated"

// RecordWorkspaceBinding persists immutable ancestry and path facts. Reuse is
// allowed only when every recorded value agrees with the existing binding.
func (s *Store) RecordWorkspaceBinding(id TaskID, binding WorkspaceBinding) (RuntimeState, error) {
	if err := validateWorkspaceBinding(binding); err != nil {
		return RuntimeState{}, err
	}
	return s.UpdateWithEvent(id, Event{Type: "workspace.bound", Detail: binding.WorktreePath}, func(state *RuntimeState) error {
		if state.Workspace == nil {
			state.Workspace = &binding
			return nil
		}
		if sameWorkspaceBinding(*state.Workspace, binding) {
			return nil
		}
		return errors.New("workspace binding conflicts with recorded parent ancestry or worktree")
	})
}

// RecordParentDrift records non-AIW parent movement without blocking work.
func (s *Store) RecordParentDrift(id TaskID, commit string, paths []string) (RuntimeState, error) {
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return RuntimeState{}, errors.New("parent drift commit is required")
	}
	return s.UpdateWithEvent(id, Event{Type: "workspace.parent-drift", Detail: commit}, func(state *RuntimeState) error {
		if state.Workspace == nil {
			return errors.New("workspace binding must be recorded before parent drift")
		}
		state.Workspace.ParentDrift = append(state.Workspace.ParentDrift, ParentDrift{ObservedAt: time.Now().UTC().Format(time.RFC3339), Commit: commit, Paths: append([]string(nil), paths...)})
		return nil
	})
}

// ConfirmParentWriteFenceViolation blocks scheduling only for a confirmed
// AIW-originated parent write; ordinary drift must not be attributed to AIW.
func (s *Store) ConfirmParentWriteFenceViolation(id TaskID, detail string) (RuntimeState, error) {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return RuntimeState{}, errors.New("parent-write fence evidence is required")
	}
	return s.UpdateWithEvent(id, Event{Type: "workspace.parent-write-fence-violated", Detail: detail}, func(state *RuntimeState) error {
		for index := range state.Gates {
			if state.Gates[index].ID == ParentWriteFenceGateID {
				state.Gates[index].State = GateOpen
				state.Gates[index].Kind = GateDependency
				state.Gates[index].Reason = detail
				return nil
			}
		}
		state.Gates = append(state.Gates, Gate{ID: ParentWriteFenceGateID, Kind: GateDependency, State: GateOpen, Reason: detail})
		return nil
	})
}

func validateWorkspaceBinding(binding WorkspaceBinding) error {
	if strings.TrimSpace(binding.ParentPath) == "" || strings.TrimSpace(binding.ParentBranch) == "" || strings.TrimSpace(binding.ParentCommit) == "" || strings.TrimSpace(binding.WorktreePath) == "" || strings.TrimSpace(binding.TaskBranch) == "" {
		return errors.New("workspace binding requires parent path, branch, commit, worktree path, and task branch")
	}
	if binding.ParentPath == binding.WorktreePath {
		return fmt.Errorf("workspace binding must use an isolated worktree")
	}
	return nil
}

func sameWorkspaceBinding(left, right WorkspaceBinding) bool {
	return left.ParentPath == right.ParentPath && left.ParentBranch == right.ParentBranch && left.ParentCommit == right.ParentCommit && left.WorktreePath == right.WorktreePath && left.TaskBranch == right.TaskBranch
}
