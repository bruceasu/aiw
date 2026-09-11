package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/ai"
	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/taskx"
	"aiw/internal/ui"
	"aiw/internal/workflow"
)

type ArchiveOptions struct {
	Push         bool
	CleanupWT    bool
	DeleteBranch bool
	Finalize     bool
	Force        bool
}

func newTask(id string, allowUnrelatedDirty bool) error {
	if !safeID(id) {
		return errors.New("invalid task id")
	}
	if err := authorizeTaskCreation(id, allowUnrelatedDirty); err != nil {
		return err
	}
	dir := taskx.TaskDir(id)
	if fsx.Exists(dir) {
		return fmt.Errorf("task already exists: %s", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(taskx.RuntimeTaskDir(id), 0o755); err != nil {
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
# Tasks
## 1. Implementation
- [ ] 1.1 Implement the approved change.
- [ ] 1.2 Add or update focused tests.

## 2. Verification
- [ ] 2.1 Run the authorized focused verification.

# Verification
- [ ] 3.1 Confirm the change meets the approved scope.
- [ ] 3.2 Confirm there are no unrelated changes.
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
	if err := ensureChecklistMapping(id); err != nil {
		return fmt.Errorf("create Work Item mapping: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(notesMD), 0o644); err != nil {
		return err
	}
	return nil
}

// ensureChecklistMapping creates the Workflow Core projection for a newly
// written OpenSpec checklist. It deliberately does not advance execution.
func ensureChecklistMapping(id string) error {
	_, store, err := compatibleWorkflow(id)
	if err != nil {
		return err
	}
	_, err = syncWorkflowChecklist(id, store)
	return err
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
		ID:            id,
		Type:          "task",
		Status:        "TODO",
		Created:       taskx.Today(),
		Updated:       taskx.Today(),
		Branch:        parentBranch,
		ParentBranch:  parentBranch,
		Worktree:      ".",
		WorkspaceKind: "primary",
		Delivery:      "unmanaged",
		Session:       id,
	}
}

func ensureTaskMeta(id string) error {
	path := taskx.TaskMetaPath(id)
	if err := os.MkdirAll(taskx.RuntimeTaskDir(id), 0o755); err != nil {
		return err
	}
	if fsx.Exists(path) {
		meta, err := taskx.ReadTaskMeta(path)
		if err != nil {
			return err
		}
		if meta.ID != "" && meta.ID != id {
			return fmt.Errorf("task metadata id mismatch: %s", meta.ID)
		}
		if meta.Branch == "" || meta.ParentBranch == "" || meta.Worktree == "" || meta.WorkspaceKind == "" || meta.Delivery == "" {
			defaults, err := newTaskMeta(id)
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
			if err := taskx.WriteTaskMeta(path, meta); err != nil {
				return err
			}
			return nil
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
	return nil
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
	entries, err := os.ReadDir(taskx.RuntimeTasksPath())
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(e.Name()))
		if err != nil {
			fmt.Printf("%-24s %-12s %s\n", e.Name(), "UNKNOWN", filepath.ToSlash(taskx.TaskDir(e.Name())))
			continue
		}
		summary, _, summaryErr := workflowSummaryForMeta(meta)
		if summaryErr != nil {
			fmt.Printf("%-24s %-24s %s\n", meta.ID, "RUNTIME_ERROR", filepath.ToSlash(taskx.TaskDir(e.Name())))
			continue
		}
		fmt.Printf("%-24s %-24s %s\n",
			meta.ID,
			summary.Status,
			filepath.ToSlash(taskx.TaskDir(e.Name())),
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
	if meta.ID != id {
		return fmt.Errorf("task metadata id mismatch: %s", meta.ID)
	}
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(taskx.WorkflowRuntimeFromMeta(meta)); err != nil {
		return err
	}
	state, err := store.UpdateWithEvent(workflow.TaskID(id), workflow.Event{Type: "task.status.compatibility-mapped", Detail: status}, func(state *workflow.RuntimeState) error {
		return workflow.ApplyLegacyStatus(state, status)
	})
	if err != nil {
		return err
	}
	return reportWorkflowState(meta, state)
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
	metaPath := taskx.ResolveTaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(metaPath)
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
	meta.WorkspaceKind, meta.Delivery, meta.Updated = "primary", "unmanaged", taskx.Today()
	if err := taskx.WriteTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return nil
}

func resolvedWorkspaceKind(meta taskx.TaskMeta) string {
	if kind := strings.TrimSpace(meta.WorkspaceKind); kind != "" {
		return kind
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		return "unassigned"
	}
	if wt == "." {
		return "primary"
	}
	root, err := gitx.ProjectRoot()
	if err != nil {
		return "unknown"
	}
	path := wt
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, filepath.FromSlash(path))
	}
	if gitx.WorktreeRegistered(path) {
		return "isolated"
	}
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
	summary, _, err := workflowSummaryForMeta(meta)
	if err != nil {
		return fmt.Errorf("read Workflow Core terminal state: %w", err)
	}
	forceClosed := summary.Status == workflow.TaskCancelled
	workflowDone := summary.Status == workflow.TaskDone
	legacyTerminal := meta.Status == "DONE" || meta.Status == "CANCELLED"
	if !forceClosed && !workflowDone && !legacyTerminal {
		return fmt.Errorf("task must be DONE or CANCELLED before archive: %s", meta.Status)
	}
	if workflowDone || (meta.Status == "DONE" && !forceClosed) {
		if err := syncArchiveWorkflow(id, metaPath, meta); err != nil {
			return err
		}
	}
	problems := archiveArtifactProblems(src)
	if len(problems) > 0 && !opts.Force {
		if err := repairArchiveArtifacts(src, problems); err != nil {
			fmt.Fprintf(os.Stderr, "archive repair failed: %v\n", err)
		}
		problems = archiveArtifactProblems(src)
		if len(problems) > 0 && !confirmArchiveWithWarnings(problems) {
			return errors.New("archive cancelled")
		}
	}
	kind := resolvedWorkspaceKind(meta)
	if kind == "unknown" {
		return errors.New("cannot archive Task with unknown workspace binding; repair it first")
	}
	if forceClosed && summary.Delivery != workflow.DeliveryMerged && summary.Delivery != workflow.DeliveryDiscarded {
		return errors.New("force-closed Task must record successful merged or discarded delivery before archive")
	}
	if !forceClosed && meta.Status == "CANCELLED" && meta.Delivery != "discarded" {
		return errors.New("cancelled Task must record discarded delivery before archive")
	}
	if kind == "primary" {
		if opts.CleanupWT || opts.DeleteBranch || opts.Finalize {
			return errors.New("primary Task has no managed worktree or branch to finalize")
		}
		if dirty, _ := gitx.IsDirty(); dirty {
			fmt.Fprintln(os.Stderr, "warning: archiving primary Task with unmanaged Git delivery and uncommitted changes")
		}
	}

	branch := strings.TrimSpace(meta.Branch)
	if branch == "" {
		branch = "feature/" + id
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(taskx.WorktreeDir, id))
	}

	delivery := summary.Delivery
	if !forceClosed {
		delivery = taskx.WorkflowRuntimeFromMeta(meta).Delivery
	}
	if kind == "isolated" && (forceClosed || meta.Status != "CANCELLED") {
		if !opts.CleanupWT {
			return errors.New("isolated worktree must be cleaned before archive")
		}
		if !opts.DeleteBranch {
			return errors.New("isolated task branch must be deleted before archive")
		}
	}
	if (kind == "isolated" || (kind == "unassigned" && delivery == workflow.DeliveryPending)) && !forceClosed && meta.Status != "CANCELLED" {
		if !gitx.IsAncestor(branch, meta.ParentBranch) {
			return fmt.Errorf("task branch %s is not merged into %s", branch, meta.ParentBranch)
		}
	}
	if opts.Push {
		if err := gitx.Run("git", "push", "-u", "origin", branch); err != nil {
			return err
		}
	}
	if opts.CleanupWT {
		if kind != "isolated" || !gitx.WorktreeRegistered(wt) {
			return errors.New("refusing to clean an unverified isolated worktree")
		}
		if err := gitx.Run("git", "worktree", "remove", wt); err != nil {
			return err
		}
	}
	if opts.DeleteBranch {
		deleteFlag := "-d"
		if delivery == workflow.DeliveryDiscarded {
			deleteFlag = "-D"
		}
		if err := gitx.Run("git", "branch", deleteFlag, branch); err != nil {
			return err
		}
	}
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
	return nil
}

func syncArchiveWorkflow(id, metaPath string, meta taskx.TaskMeta) error {
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(taskx.WorkflowRuntimeFromMeta(meta)); err != nil {
		return err
	}
	state, err := syncWorkflowChecklist(id, store)
	if err != nil {
		return fmt.Errorf("archive requires workflow sync: %w", err)
	}
	return reportWorkflowState(meta, state)
}

func archiveArtifactProblems(src string) []string {
	problems := []string{}
	tasksPath := filepath.Join(src, "tasks.md")
	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return []string{"tasks.md is missing or unreadable"}
	}
	_, diagnostics := taskx.ParseNumberedChecklist(string(content))
	for _, diagnostic := range diagnostics {
		problems = append(problems, "tasks.md: "+diagnostic.Message)
	}
	return problems
}

func repairArchiveArtifacts(src string, problems []string) error {
	config, err := ai.LoadConfig()
	if err != nil {
		return err
	}
	prompt := fmt.Sprintf(`Repair the OpenSpec change at %s so it can be archived.
Only modify files inside that directory. Preserve the intended requirements and completed checklist items.
Validation problems:
%s
Return JSON only in this shape: {"files":[{"path":"tasks.md","content":"..."}]}`, src, strings.Join(problems, "\n"))
	output, err := ai.RunLLMWithSystemPrompt(prompt, ai.LLMConfig{
		Provider: config.Name, Model: config.Model, APIBaseURL: config.BaseURL,
		APIKey: config.APIKey, CodexCommand: config.Command, CopilotCommand: config.Command,
	}, map[string]any{"type": "object"}, "You repair OpenSpec artifacts. Return only the requested JSON repair document.")
	if err != nil {
		return err
	}
	var response struct {
		Files []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return fmt.Errorf("LLM returned invalid repair JSON: %w", err)
	}
	for _, file := range response.Files {
		path := filepath.Clean(file.Path)
		if path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			return fmt.Errorf("LLM repair path escapes change directory: %s", file.Path)
		}
		if err := os.WriteFile(filepath.Join(src, path), []byte(file.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func confirmArchiveWithWarnings(problems []string) bool {
	fmt.Fprintln(os.Stderr, "archive validation warnings:")
	for _, problem := range problems {
		fmt.Fprintln(os.Stderr, "-", problem)
	}
	if !ui.IsInteractiveTerminal() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(ui.PromptLine("Archive anyway? [y/N] ")))
	return answer == "y" || answer == "yes"
}

func printContext(id string) error {
	changeDir := taskx.TaskDir(id)
	if !fsx.Exists(changeDir) {
		return fmt.Errorf("task not found: %s", id)
	}
	fmt.Print("Read these files first:\n\n")
	files := []string{
		taskx.TaskMetaPath(id),
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
	meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id))
	if err == nil {
		for _, spec := range meta.Specs {
			fmt.Println("-", filepath.ToSlash(filepath.Join(taskx.SpecsDir, spec, "spec.md")))
		}
		summary, local, summaryErr := workflowSummaryForMeta(meta)
		if summaryErr != nil {
			fmt.Printf("\nWorkflow runtime: unavailable (%v)\n", summaryErr)
		} else {
			source := "durable metadata compatibility projection"
			if local {
				source = "local runtime projection"
			}
			fmt.Printf("\nWorkflow (%s):\n", source)
			fmt.Printf("- Status: %s\n", summary.Status)
			fmt.Printf("- Planning: %s; execution: %s; validation: %s; delivery: %s\n", summary.Planning, summary.Execution, summary.Validation, summary.Delivery)
			if len(summary.BlockedBy) > 0 {
				fmt.Printf("- Blocking gates: %s\n", strings.Join(gateIDs(summary.BlockedBy), ", "))
			}
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

func workflowSummaryForMeta(meta taskx.TaskMeta) (workflow.TaskSummary, bool, error) {
	store := workflow.NewStore("")
	state, err := store.Load(workflow.TaskID(meta.ID))
	if err == nil {
		return workflow.DeriveSummary(state), true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return workflow.TaskSummary{}, false, err
	}
	compatible := taskx.WorkflowRuntimeFromMeta(meta)
	return workflow.DeriveSummary(compatible), false, nil
}

func gateIDs(ids []workflow.GateID) []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = string(id)
	}
	return result
}

func parseArchiveOptions(args []string) (ArchiveOptions, error) {
	allowed := map[string]bool{
		"--push":          true,
		"--cleanup-wt":    true,
		"--delete-branch": true,
		"--finalize":      true,
		"--force":         true,
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
	if hasFlag(args, "--force") {
		opts.Force = true
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
