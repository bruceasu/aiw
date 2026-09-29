package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow"
)

func newTask(id string, allowUnrelatedDirty bool) error {
	if err := task.CreateIssueTask(id, "", allowUnrelatedDirty); err != nil {
		return err
	}
	if err := ensureChecklistMapping(id); err != nil {
		return fmt.Errorf("create Work Item mapping: %w", err)
	}
	return nil
}

// ensureChecklistMapping creates the Workflow Core projection for a newly
// written OpenSpec checklist. It deliberately does not advance execution.
func ensureChecklistMapping(id string) error {
	_, store, err := taskworkflow.CompatibleWorkflow(id)
	if err != nil {
		return err
	}
	_, err = taskworkflow.SyncWorkflowChecklist(id, store)
	return err
}

func newTaskMeta(id string) (task.TaskMeta, error) {
	return task.NewTaskMeta(id)
}

func taskMetaFor(id, parentBranch string) task.TaskMeta {
	return task.TaskMeta{ID: id, Type: "task", Status: "TODO", Created: task.Today(), Updated: task.Today(), Branch: parentBranch, ParentBranch: parentBranch, Worktree: ".", WorkspaceKind: "primary", Delivery: "unmanaged", Session: id}
}

func ensureTaskMeta(id string) error {
	return task.EnsureTaskMeta(id)
}

func createDecision(id string) error {
	fmt.Fprintln(os.Stderr, "legacy command: aiw decision creates an OpenSpec change design.md; write new Task decisions in docs/features/<task-id>.md")
	dir := task.TaskDir(id)
	if !fsx.Exists(dir) {
		return fmt.Errorf("task not found: %s", id)
	}
	design := filepath.Join(dir, "design.md")
	if fsx.Exists(design) {
		fmt.Println("design.md already exists")
		return nil
	}
	content := fmt.Sprintf(`# %s Design
## Decision
...
## Why
...
## Risks
...
## Future Notes
...
`, id)
	return os.WriteFile(design, []byte(content), 0o644)
}

func createSpec(id string) error {
	dir := filepath.Join(task.SpecsDir, id)
	if fsx.Exists(dir) {
		return fmt.Errorf("spec already exists: %s", id)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	meta := `id = "` + id + `"
type = "spec"
status = "active"
created = "` + task.Today() + `"
updated = "` + task.Today() + `"
`
	spec := fmt.Sprintf(`# %s Spec
## Purpose
...
## Invariants
-
## APIs
-
## Notes
...
`, strings.Title(id))
	if err := os.WriteFile(filepath.Join(dir, "spec.toml"), []byte(meta), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "spec.md"), []byte(spec), 0o644)
}

func listTaskIDs() ([]string, error) {
	locations, err := task.DiscoverTaskLocations()
	ids := make([]string, 0, len(locations))
	for _, location := range locations {
		ids = append(ids, location.ID)
	}
	return ids, err
}

// listTaskMetaPath preserves the existing path precedence without hiding stat errors.
// An empty path means that neither supported metadata file exists.
func listTaskMetaPath(id string) (string, error) {
	dir := filepath.Join(task.RuntimeTasksPath(), id)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		if legacyErr := task.LegacyTaskPathError(id); legacyErr != nil {
			return "", legacyErr
		}
	} else if err != nil {
		return "", fmt.Errorf("check task directory %s: %w", dir, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("check task directory %s: not a directory", dir)
	}
	for _, name := range []string{task.TaskMetaFile, task.LegacyTaskMetaFile} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("check task metadata %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("check task metadata %s: not a regular file", path)
		}
		return path, nil
	}
	return "", nil
}

type taskListRow struct {
	ID       string
	Status   string
	Path     string
	Archived bool
}

// collectTaskListRows filters before repair. Explicit archive listings can reuse
// the same discovery without redirecting archived state into the active store.
func collectTaskListRows(includeArchived bool) ([]taskListRow, error) {
	locations, err := task.DiscoverTaskLocations()
	if err != nil {
		// An unreadable root could hide a conflicting identity. Do not repair.
		return nil, err
	}
	var rows []taskListRow
	var problems []error
	for _, location := range locations {
		if len(location.Problems) > 0 {
			problem := fmt.Errorf("task %s: %w", location.ID, errors.Join(location.Problems...))
			fmt.Fprintln(os.Stderr, problem)
			problems = append(problems, problem)
			continue
		}
		if location.Archived && !includeArchived {
			continue
		}
		path := "规格已删除"
		if location.ChangeDir != "" {
			path = filepath.ToSlash(location.ChangeDir)
		} else if fsx.Exists(task.FeatureDesignPath(location.ID)) {
			path = filepath.ToSlash(task.FeatureDesignPath(location.ID))
		} else if location.Archived && fsx.Exists(task.FeatureDesignArchivePath(location.ArchiveName)) {
			path = filepath.ToSlash(task.FeatureDesignArchivePath(location.ArchiveName))
		} else if location.RuntimeDir != "" {
			path = filepath.ToSlash(location.RuntimeDir)
		}
		state, rebuilt, runtimeErr := location.EnsureRuntime()
		status := "RUNTIME_ERROR"
		if rebuilt {
			fmt.Fprintf(os.Stderr, "task %s: 已补建缺失文件（最小记录），历史状态可能丢失 (%s)\n", location.ID, location.RuntimeDir)
		}
		if runtimeErr != nil {
			problem := fmt.Errorf("task %s: %w", location.ID, runtimeErr)
			fmt.Fprintln(os.Stderr, problem)
			problems = append(problems, problem)
		} else {
			status = string(workflow.DeriveSummary(state).Status)
		}
		rows = append(rows, taskListRow{ID: location.ID, Status: status, Path: path, Archived: location.Archived})
	}
	return rows, errors.Join(problems...)
}

func showTask(id string) error {
	if fd := task.FeatureDesignPath(id); fsx.Exists(fd) {
		b, err := os.ReadFile(fd)
		if err != nil { return err }
		fmt.Print(string(b))
		return nil
	}
	path := filepath.Join(task.ChangesDir, id, "tasks.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}

func updateStatus(id, status string) error {
	metaPath := task.ResolveTaskMetaPath(id)
	meta, err := task.ReadTaskMeta(metaPath)
	if err != nil {
		return err
	}
	if meta.ID != id {
		return fmt.Errorf("task metadata id mismatch: %s", meta.ID)
	}
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(taskworkflow.WorkflowRuntimeFromMeta(meta)); err != nil {
		return err
	}
	state, err := store.UpdateWithEvent(workflow.TaskID(id), workflow.Event{Type: "task.status.compatibility-mapped", Detail: status}, func(state *workflow.RuntimeState) error {
		return workflow.ApplyLegacyStatus(state, status)
	})
	if err != nil {
		return err
	}
	return taskworkflow.ProjectWorkflowState(id, state)
}

func bindTaskWorkspace(args []string) error {
	if len(args) != 3 || args[0] != "bind" || args[2] != "--primary" {
		return errors.New("usage: task workspace bind <task-id> --primary")
	}
	id := args[1]
	primary, primaryPath, err := gitx.IsPrimaryWorktree()
	if err != nil {
		return err
	}
	if !primary {
		return fmt.Errorf("primary workspace is %s", primaryPath)
	}
	metaPath := task.ResolveTaskMetaPath(id)
	meta, err := task.ReadTaskMeta(metaPath)
	if err != nil {
		return err
	}
	branch, err := gitx.CurrentBranch()
	if err != nil {
		return err
	}
	if meta.ParentBranch != "" && meta.ParentBranch != branch {
		return fmt.Errorf("current branch %s does not match parent_branch %s", branch, meta.ParentBranch)
	}
	meta.Branch, meta.ParentBranch, meta.Worktree = branch, branch, "."
	meta.WorkspaceKind, meta.Delivery, meta.Updated = "primary", "unmanaged", task.Today()
	if err := task.WriteTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return nil
}

func resolvedWorkspaceKind(meta task.TaskMeta) string {
	return task.ResolvedWorkspaceKind(meta)
}
