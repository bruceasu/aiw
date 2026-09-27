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

// ReconcileTaskReference updates the compatibility projection of the Task's
// current workspace metadata. It is intentionally allowed only while no
// execution is entitled to write: historical Attempts and workspace ancestry
// remain untouched, while a stale primary/isolated marker cannot block later
// lifecycle decisions forever.
func (s *Store) ReconcileTaskReference(id TaskID, reference TaskReference) (RuntimeState, error) {
	if reference.ID != id {
		return RuntimeState{}, errors.New("Task reference id mismatch")
	}
	if strings.TrimSpace(reference.Workspace) == "" && reference.Kind != WorkspaceUnassigned {
		return RuntimeState{}, errors.New("Task reference workspace is required")
	}
	if reference.Kind != WorkspacePrimary && reference.Kind != WorkspaceIsolated && reference.Kind != WorkspaceUnassigned {
		return RuntimeState{}, fmt.Errorf("unsupported Task workspace kind: %s", reference.Kind)
	}
	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if state.SchemaVersion == DurableSchemaVersion {
		return RuntimeState{}, errors.New("durable Task reference reconciliation requires its protocol migration seam")
	}
	return s.UpdateWithEvent(id, Event{Type: "task.workspace-reconciled", Detail: fmt.Sprintf("%s:%s -> %s:%s", state.Task.Kind, state.Task.Workspace, reference.Kind, reference.Workspace)}, func(current *RuntimeState) error {
		if current.WriteLease != nil || current.Automation.PreparedRequest != nil {
			return errors.New("cannot reconcile Task workspace while execution is prepared or leased")
		}
		for _, attempt := range current.Attempts {
			switch attempt.State {
			case AttemptCreated, AttemptRunning, AttemptPaused:
				return fmt.Errorf("cannot reconcile Task workspace while Attempt %s is active", attempt.ID)
			}
		}
		if current.Task.Workspace == reference.Workspace && current.Task.Kind == reference.Kind {
			return errProtocolNoChange
		}
		current.Task.Workspace = reference.Workspace
		current.Task.Kind = reference.Kind
		return nil
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
