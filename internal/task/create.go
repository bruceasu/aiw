package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"aiw/internal/fsx"
	"aiw/internal/gitx"
)

// CreateTask creates the Task source files and metadata without initializing a
// Workflow projection. The caller that owns Workflow integration performs the
// projection after the source files are durable.
func CreateTask(id string, allowUnrelatedDirty bool) error {
	if err := ValidateTaskID(id); err != nil {
		return err
	}
	if err := LegacyTaskPathError(id); err != nil {
		return err
	}
	if err := AuthorizeTaskCreation(id, allowUnrelatedDirty); err != nil {
		return err
	}
	dir := TaskDir(id)
	if fsx.Exists(dir) {
		return fmt.Errorf("task already exists: %s", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(RuntimeTaskDir(id), 0o755); err != nil {
		return err
	}
	meta, err := NewTaskMeta(id)
	if err != nil {
		return err
	}
	taskMD := `# Goal
Describe the goal.
# Scope
Included:
-
Out of scope:
-
# Constraints
- Do not refactor unrelated modules.
- Preserve backward compatibility.
# Context
Relevant modules:
-
# Tasks
## 1. Implementation
- [ ] 1.1 Implement the approved change.
- [ ] 1.2 Add or update focused tests.

# Verification
- [ ] 3.1 Confirm the change meets the approved scope.
- [ ] 3.2 Confirm there are no unrelated changes.
# Notes
%% AI notes go here
`
	notesMD := `# Notes
Temporary findings, debugging notes, experiments.
`
	if err := WriteTaskMeta(TaskMetaPath(id), meta); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(taskMD), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(notesMD), 0o644); err != nil {
		return err
	}
	return nil
}

// CreateIssueTask creates a Task and its FD without an OpenSpec change.
// Existing native Tasks keep their historical tasks.md layout.
func CreateIssueTask(id, issueID string, allowUnrelatedDirty bool) error {
	if err := ValidateTaskID(id); err != nil { return err }
	if err := LegacyTaskPathError(id); err != nil { return err }
	if err := AuthorizeTaskCreation(id, allowUnrelatedDirty); err != nil { return err }
	if fsx.Exists(RuntimeTaskDir(id)) || fsx.Exists(TaskDir(id)) || fsx.Exists(FeatureDesignPath(id)) {
		return fmt.Errorf("task or Feature Design already exists: %s", id)
	}
	if err := os.MkdirAll(RuntimeTaskDir(id), 0o755); err != nil { return err }
	if err := os.MkdirAll(FeatureDesignDir, 0o755); err != nil { return err }
	meta, err := NewTaskMeta(id)
	if err != nil { return err }
	if err := WriteTaskMeta(TaskMetaPath(id), meta); err != nil { return err }
	if issueID == "" { issueID = "(direct Task)" }
	content := "# Feature Design: " + id + "\n\n" +
		"Issue: " + issueID + "\n\n## Goal and scope\n\n" +
		"%% NEEDS_INPUT: Carry the approved Issue goal and scope into this FD.\n\n" +
		"## Decisions and constraints\n\n" +
		"%% NEEDS_INPUT: Record material design decisions and compatibility effects.\n\n" +
		"## Design Readiness\n\nBLOCKED\n\n" +
		"## Work Items\n\n" +
		"%% NEEDS_INPUT: Add ordered, numbered work items from the approved Issue.\n\n" +
		"## TODO\n\n- [ ] Complete the FD from approved Issue evidence.\n\n" +
		"## Verification\n\n%% NEEDS_INPUT: Record observable acceptance evidence.\n"
	return os.WriteFile(FeatureDesignPath(id), []byte(content), 0o644)
}

// EnsureTaskMeta repairs only missing Task metadata fields and never changes
// an existing identity or Workflow state.
func EnsureTaskMeta(id string) error {
	if err := LegacyTaskPathError(id); err != nil {
		return err
	}
	path := TaskMetaPath(id)
	if err := os.MkdirAll(RuntimeTaskDir(id), 0o755); err != nil {
		return err
	}
	if fsx.Exists(path) {
		meta, err := ReadTaskMeta(path)
		if err != nil {
			return err
		}
		if meta.ID != "" && meta.ID != id {
			return fmt.Errorf("task metadata id mismatch: %s", meta.ID)
		}
		if meta.Branch == "" || meta.ParentBranch == "" || meta.Worktree == "" || meta.WorkspaceKind == "" || meta.Delivery == "" {
			defaults, err := NewTaskMeta(id)
			if err != nil {
				return err
			}
			if meta.Branch == "" {
				meta.Branch = defaults.Branch
			}
			if meta.ParentBranch == "" {
				meta.ParentBranch = defaults.ParentBranch
			}
			if meta.Worktree == "" {
				meta.Worktree = defaults.Worktree
			}
			if meta.WorkspaceKind == "" {
				meta.WorkspaceKind = defaults.WorkspaceKind
			}
			if meta.Delivery == "" {
				meta.Delivery = defaults.Delivery
			}
			if meta.Session == "" {
				meta.Session = defaults.Session
			}
			return WriteTaskMeta(path, meta)
		}
		return nil
	}
	meta, err := NewTaskMeta(id)
	if err != nil {
		return err
	}
	return WriteTaskMeta(path, meta)
}

// NewTaskMeta creates the default lifecycle binding for a new Task.
func NewTaskMeta(id string) (TaskMeta, error) {
	parentBranch, err := gitx.CurrentBranch()
	if err != nil {
		return TaskMeta{}, fmt.Errorf("create task metadata: %w", err)
	}
	return TaskMeta{
		ID: id, Type: "task", Status: "TODO", Created: Today(), Updated: Today(),
		Branch: parentBranch, ParentBranch: parentBranch, Worktree: ".",
		WorkspaceKind: "primary", Delivery: "unmanaged", Session: id,
	}, nil
}

// ValidateTaskID is the Task-owned identifier contract used by Task and
// Workflow adapters.
func ValidateTaskID(id string) error {
	if id == "" {
		return errors.New("invalid task id")
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return errors.New("invalid task id")
	}
	return nil
}
