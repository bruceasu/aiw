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
	Finalize     bool
}

func newTask(id string, allowDirty bool) error {
	if !safeID(id) {
		return errors.New("invalid task id")
	}
	primary, primaryPath, err := gitx.IsPrimaryWorktree()
	if err != nil { return err }
	if !primary { return fmt.Errorf("ordinary Tasks must be created from the primary workspace: %s", primaryPath) }
	dirty, err := gitx.IsDirty()
	if err != nil { return err }
	if dirty && !allowDirty { return errors.New("working tree has uncommitted changes; commit or clean them, or rerun with --allow-dirty") }
	dir := taskx.TaskDir(id)
	if fsx.Exists(dir) {
		return fmt.Errorf("task already exists: %s", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	meta, err := newTaskMeta(id)
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
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath(id), meta); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(taskMD), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(notesMD), 0o644); err != nil {
		return err
	}
	return writeRegistry()
}

func newTaskMeta(id string) (taskx.TaskMeta, error) {
	parentBranch, err := gitx.CurrentBranch()
	if err != nil {
		return taskx.TaskMeta{}, fmt.Errorf("create task metadata: %w", err)
	}
	return taskMetaFor(id, parentBranch), nil
}

func taskMetaFor(id, parentBranch string) taskx.TaskMeta {
	return taskx.TaskMeta{
		ID:           id,
		Type:         "task",
		Status:       "TODO",
		Created:      taskx.Today(),
		Updated:      taskx.Today(),
		Branch:       parentBranch,
		ParentBranch: parentBranch,
		Worktree:     ".",
		WorkspaceKind: "primary",
		Delivery:     "unmanaged",
		Session:      id,
	}
}

func ensureTaskMeta(id string) error {
	path := taskx.TaskMetaPath(id)
	if fsx.Exists(path) {
		meta, err := taskx.ReadTaskMeta(path)
		if err != nil {
			return err
		}
		if meta.ID != "" && meta.ID != id {
			return fmt.Errorf("task metadata id mismatch: %s", meta.ID)
		}
		if meta.Branch == "" || meta.ParentBranch == "" || meta.Worktree == "" || meta.WorkspaceKind == "" || meta.Delivery == "" {
			defaults, err := newTaskMeta(id); if err != nil { return err }
			if meta.Branch == "" { meta.Branch = defaults.Branch }
			if meta.ParentBranch == "" { meta.ParentBranch = defaults.ParentBranch }
			if meta.Worktree == "" { meta.Worktree = defaults.Worktree }
			if meta.WorkspaceKind == "" { meta.WorkspaceKind = defaults.WorkspaceKind }
			if meta.Delivery == "" { meta.Delivery = defaults.Delivery }
			if meta.Session == "" { meta.Session = defaults.Session }
			if err := taskx.WriteTaskMeta(path, meta); err != nil { return err }
			return writeRegistry()
		}
		return nil
	}
	meta, err := newTaskMeta(id)
	if err != nil {
		return err
	}
	if err := taskx.WriteTaskMeta(path, meta); err != nil {
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
		if e.Name() == "archive" {
			continue
		}
		meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPathInDir(filepath.Join(taskx.ChangesDir, e.Name())))
		if err != nil {
			fmt.Printf("%-24s %-12s %s\n", e.Name(), "UNKNOWN", filepath.ToSlash(filepath.Join(taskx.ChangesDir, e.Name())))
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
	path := filepath.Join(taskx.ChangesDir, id, "tasks.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}

func updateStatus(id, status string) error {
	metaPath := taskx.ResolveTaskMetaPath(id)
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

func bindTaskWorkspace(args []string) error {
	if len(args) != 3 || args[0] != "bind" || args[2] != "--primary" { return errors.New("usage: task workspace bind <task-id> --primary") }
	id := args[1]
	primary, primaryPath, err := gitx.IsPrimaryWorktree(); if err != nil { return err }
	if !primary { return fmt.Errorf("primary workspace is %s", primaryPath) }
	metaPath := taskx.ResolveTaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(metaPath); if err != nil { return err }
	branch, err := gitx.CurrentBranch(); if err != nil { return err }
	if meta.ParentBranch != "" && meta.ParentBranch != branch { return fmt.Errorf("current branch %s does not match parent_branch %s", branch, meta.ParentBranch) }
	meta.Branch, meta.ParentBranch, meta.Worktree = branch, branch, "."
	meta.WorkspaceKind, meta.Delivery, meta.Updated = "primary", "unmanaged", taskx.Today()
	if err := taskx.WriteTaskMeta(metaPath, meta); err != nil { return err }
	return writeRegistry()
}

func resolvedWorkspaceKind(meta taskx.TaskMeta) string {
	if kind := strings.TrimSpace(meta.WorkspaceKind); kind != "" { return kind }
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" { return "unassigned" }
	if wt == "." { return "primary" }
	root, err := gitx.ProjectRoot(); if err != nil { return "unknown" }
	path := wt; if !filepath.IsAbs(path) { path = filepath.Join(root, filepath.FromSlash(path)) }
	if gitx.WorktreeRegistered(path) { return "isolated" }
	return "unknown"
}

func archiveTask(id string, opts ArchiveOptions) error {
	src := taskx.TaskDir(id)
	if !fsx.Exists(src) {
		return fmt.Errorf("task not found: %s", id)
	}

	metaPath := taskx.ResolveTaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(metaPath)
	if err != nil {
		return err
	}
	if meta.Status != "DONE" && meta.Status != "CANCELLED" { return fmt.Errorf("task must be DONE or CANCELLED before archive: %s", meta.Status) }
	kind := resolvedWorkspaceKind(meta)
	if kind == "unknown" { return errors.New("cannot archive Task with unknown workspace binding; repair it first") }
	if meta.Status == "CANCELLED" && meta.Delivery != "discarded" { return errors.New("cancelled Task must record discarded delivery before archive") }
	if kind == "primary" {
		if opts.CleanupWT || opts.DeleteBranch || opts.Finalize { return errors.New("primary Task has no managed worktree or branch to finalize") }
		if dirty, _ := gitx.IsDirty(); dirty { fmt.Fprintln(os.Stderr, "warning: archiving primary Task with unmanaged Git delivery and uncommitted changes") }
	}

	branch := strings.TrimSpace(meta.Branch)
	if branch == "" {
		branch = "feature/" + id
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(taskx.WorktreeDir, id))
	}

	if (kind == "isolated" || (kind == "unassigned" && meta.Delivery == "pending")) && meta.Status != "CANCELLED" {
		if !gitx.IsAncestor(branch, meta.ParentBranch) { return fmt.Errorf("task branch %s is not merged into %s", branch, meta.ParentBranch) }
		meta.Delivery = "merged"
		if kind == "isolated" && !opts.CleanupWT { return errors.New("isolated worktree must be cleaned before archive") }
		if !opts.DeleteBranch { return errors.New("merged task branch must be deleted before archive") }
	}
	if opts.Push {
		if err := gitx.Run("git", "push", "-u", "origin", branch); err != nil {
			return err
		}
	}
	if opts.CleanupWT {
		if kind != "isolated" || !gitx.WorktreeRegistered(wt) { return errors.New("refusing to clean an unverified isolated worktree") }
		if err := gitx.Run("git", "worktree", "remove", wt); err != nil {
			return err
		}
	}
	if opts.DeleteBranch {
		if err := gitx.Run("git", "branch", "-d", branch); err != nil {
			return err
		}
	}
	meta.Updated = taskx.Today()
	if err := taskx.WriteTaskMeta(metaPath, meta); err != nil { return err }

	if err := syncSpecSnapshots(src, meta.Specs); err != nil {
		return err
	}

	dst := taskx.ArchiveTaskDir(taskx.Today() + "-" + id)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
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
		filepath.Join(changeDir, taskx.TaskMetaFile),
		filepath.Join(changeDir, taskx.LegacyTaskMetaFile),
		filepath.Join(changeDir, "proposal.md"),
		filepath.Join(changeDir, "tasks.md"),
		filepath.Join(changeDir, "design.md"),
		filepath.Join(changeDir, "notes.md"),
	}
	for _, f := range files {
		if fsx.Exists(f) {
			fmt.Println("-", filepath.ToSlash(f))
		}
	}
	meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPathInDir(changeDir))
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
		fmt.Fprintln(os.Stderr, "warning: --finalize is deprecated; use explicit cleanup options")
		opts.CleanupWT = true
		opts.DeleteBranch = true
		opts.Finalize = true
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

func syncSpecSnapshots(taskDir string, specIDs []string) error {
	if len(specIDs) == 0 {
		return nil
	}

	for _, specID := range specIDs {
		sourceRoot := filepath.Join(taskDir, "specs", specID)
		if !fsx.Exists(sourceRoot) {
			continue
		}
		targetPath, err := resolveSpecTargetPath(specID, sourceRoot)
		if err != nil {
			return err
		}
		if targetPath == "" {
			continue
		}
		if err := mergeSpecFile(targetPath, filepath.Join(sourceRoot, "spec.md"), specID); err != nil {
			return err
		}
	}
	return nil
}

func resolveSpecTargetPath(specID, sourceRoot string) (string, error) {
	category, ok := specCategoryForID(specID, sourceRoot)
	if !ok {
		return "", nil
	}
	targetDir := filepath.Join(taskx.SpecsDir, category)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(targetDir, "spec.md"), nil
}

func specCategoryForID(specID, sourceRoot string) (string, bool) {
	lowered := strings.ToLower(specID)
	switch {
	case strings.Contains(lowered, "file-operation"), strings.Contains(lowered, "file-operations"):
		return "ai-support", true
	case strings.Contains(lowered, "workflow"), strings.Contains(lowered, "session"), strings.Contains(lowered, "grill"), strings.Contains(lowered, "task-agent"), strings.Contains(lowered, "handoff"):
		return "ai-support", true
	case strings.Contains(lowered, "plugin"), strings.Contains(lowered, "github"), strings.Contains(lowered, "patch"):
		return "plugins", true
	case strings.Contains(lowered, "skill"), strings.Contains(lowered, "capability"):
		return "capabilities", true
	}

	specPath := filepath.Join(sourceRoot, "spec.md")
	b, err := os.ReadFile(specPath)
	if err != nil {
		return "", false
	}
	text := strings.ToLower(string(b))
	switch {
	case strings.Contains(text, "github"), strings.Contains(text, "plugin"), strings.Contains(text, "python"):
		return "plugins", true
	case strings.Contains(text, "skill"), strings.Contains(text, "capabilit"):
		return "capabilities", true
	case strings.Contains(text, "session"), strings.Contains(text, "workflow"), strings.Contains(text, "handoff"), strings.Contains(text, "grill"):
		return "ai-support", true
	default:
		return "workflow", true
	}
}

func mergeSpecFile(targetPath, sourcePath, specID string) error {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	sourceText := strings.TrimSpace(string(source))
	if sourceText == "" {
		return nil
	}
	var targetText string
	if fsx.Exists(targetPath) {
		b, err := os.ReadFile(targetPath)
		if err != nil {
			return err
		}
		targetText = string(b)
	}
	merged := mergeSpecContent(targetText, sourceText, specID)
	if merged == targetText {
		return nil
	}
	return os.WriteFile(targetPath, []byte(merged), 0o644)
}

func mergeSpecContent(existing, incoming, specID string) string {
	marker := "<!-- archived spec: " + specID + " -->"
	if strings.Contains(existing, marker) {
		return existing
	}
	block := marker + "\n\n" + strings.TrimSpace(incoming) + "\n"
	trimmed := strings.TrimSpace(existing)
	if trimmed == "" {
		return block
	}
	if !strings.HasSuffix(trimmed, "\n") {
		trimmed += "\n"
	}
	return trimmed + "\n" + block
}
