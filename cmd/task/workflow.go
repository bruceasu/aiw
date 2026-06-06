package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/taskx"
)

type ArchiveOptions struct {
	Push         bool
	CleanupWT    bool
	DeleteBranch bool
}

func newTask(id string) error {
	if !safeID(id) {
		return errors.New("invalid task id")
	}
	dir := taskx.TaskDir(id)
	if fsx.Exists(dir) {
		return fmt.Errorf("task already exists: %s", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	meta := taskx.TaskMeta{
		ID:       id,
		Type:     "task",
		Status:   "TODO",
		Created:  taskx.Today(),
		Updated:  taskx.Today(),
		Branch:   "feature/" + id,
		Worktree: filepath.ToSlash(filepath.Join(taskx.WorktreeDir, id)),
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
# TODO
- [ ] implement
- [ ] tests
- [ ] verification
# Verification
- [ ] tests pass
- [ ] no unrelated changes
# Notes
%% AI notes go here
`
	notesMD := `# Notes
Temporary findings, debugging notes, experiments.
`
	if err := taskx.WriteTaskMeta(filepath.Join(dir, "task.toml"), meta); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "task.md"), []byte(taskMD), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(notesMD), 0o644); err != nil {
		return err
	}
	return writeRegistry()
}

func createDecision(id string) error {
	dir := taskx.TaskDir(id)
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
	dir := filepath.Join(taskx.SpecsDir, id)
	if fsx.Exists(dir) {
		return fmt.Errorf("spec already exists: %s", id)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	meta := `id = "` + id + `"
type = "spec"
status = "active"
created = "` + taskx.Today() + `"
updated = "` + taskx.Today() + `"
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

func listTasks() error {
	entries, err := os.ReadDir(taskx.ChangesDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := taskx.ReadTaskMeta(filepath.Join(taskx.ChangesDir, e.Name(), "task.toml"))
		if err != nil {
			continue
		}
		fmt.Printf("%-24s %-12s %s\n",
			meta.ID,
			meta.Status,
			filepath.ToSlash(filepath.Join(taskx.ChangesDir, e.Name())),
		)
	}
	return nil
}

func showTask(id string) error {
	path := filepath.Join(taskx.ChangesDir, id, "task.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}

func updateStatus(id, status string) error {
	metaPath := taskx.TaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(metaPath)
	if err != nil {
		return err
	}
	meta.Status = strings.ToUpper(status)
	meta.Updated = taskx.Today()
	if err := taskx.WriteTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return writeRegistry()
}

func archiveTask(id string, opts ArchiveOptions) error {
	src := taskx.TaskDir(id)
	if !fsx.Exists(src) {
		return fmt.Errorf("task not found: %s", id)
	}

	metaPath := taskx.TaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(metaPath)
	if err != nil {
		return err
	}

	branch := strings.TrimSpace(meta.Branch)
	if branch == "" {
		branch = "feature/" + id
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(taskx.WorktreeDir, id))
	}

	if opts.Push {
		if err := gitx.Run("git", "push", "-u", "origin", branch); err != nil {
			return err
		}
	}
	if opts.CleanupWT {
		if err := gitx.Run("git", "worktree", "remove", wt); err != nil {
			return err
		}
	}
	if opts.DeleteBranch {
		if err := gitx.Run("git", "branch", "-d", branch); err != nil {
			return err
		}
	}

	dst := filepath.Join(taskx.ArchiveDir, taskx.Today()+"-"+id)
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return writeRegistry()
}

func printContext(id string) error {
	changeDir := taskx.TaskDir(id)
	if !fsx.Exists(changeDir) {
		return fmt.Errorf("task not found: %s", id)
	}
	fmt.Print("Read these files first:\n\n")
	files := []string{
		filepath.Join(changeDir, "task.toml"),
		filepath.Join(changeDir, "task.md"),
		filepath.Join(changeDir, "design.md"),
		filepath.Join(changeDir, "notes.md"),
	}
	for _, f := range files {
		if fsx.Exists(f) {
			fmt.Println("-", filepath.ToSlash(f))
		}
	}
	meta, err := taskx.ReadTaskMeta(filepath.Join(changeDir, "task.toml"))
	if err == nil {
		for _, spec := range meta.Specs {
			fmt.Println("-", filepath.ToSlash(filepath.Join(taskx.SpecsDir, spec, "spec.md")))
		}
	}
	fmt.Print(`
Instruction:
- implement only the scoped task
- avoid unrelated refactors
- preserve backward compatibility
- update TODO and Verification before finishing
- use %% notes instead of guessing
`)
	return nil
}

func parseArchiveOptions(args []string) (ArchiveOptions, error) {
	allowed := map[string]bool{
		"--push":          true,
		"--cleanup-wt":    true,
		"--delete-branch": true,
		"--finalize":      true,
	}
	for _, a := range args {
		if !allowed[a] {
			return ArchiveOptions{}, fmt.Errorf("unknown archive option: %s", a)
		}
	}
	opts := ArchiveOptions{
		Push:         hasFlag(args, "--push"),
		CleanupWT:    hasFlag(args, "--cleanup-wt"),
		DeleteBranch: hasFlag(args, "--delete-branch"),
	}
	if hasFlag(args, "--finalize") {
		opts.Push = true
		opts.CleanupWT = true
		opts.DeleteBranch = true
	}
	return opts, nil
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func safeID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func writeRegistry() error {
	return taskx.WriteRegistry()
}
