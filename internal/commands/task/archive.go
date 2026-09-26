package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/ai"
	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/session"
	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
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

func archiveTask(id string, opts ArchiveOptions) error {
	return archiveWithBackend(id, opts, "")
}

// A single seam keeps fault injection at the existing CLI boundary.
var archiveRename = os.Rename

type archiveMove struct{ source, target string }

type taskArchivePlan struct {
	location task.TaskLocation
	meta     task.TaskMeta
	metaPath string
	summary  workflow.TaskSummary
	name     string
	moves    []archiveMove
}

func sameArchivePath(a, b string) bool {
	a, errA := filepath.Abs(a)
	b, errB := filepath.Abs(b)
	return errA == nil && errB == nil && filepath.Clean(a) == filepath.Clean(b)
}

func requireAbsent(path string) error {
	_, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("archive target or writer lock already exists: %s", path)
}

func prepareTaskArchive(id string) (taskArchivePlan, error) {
	plan := taskArchivePlan{}
	if !taskworkflow.SafeID(id) {
		return plan, errors.New("invalid task id")
	}
	locations, err := task.DiscoverTaskLocations()
	if err != nil {
		return plan, err
	}
	found := false
	for _, location := range locations {
		if location.ID == id {
			plan.location, found = location, true
			break
		}
	}
	if !found {
		return plan, fmt.Errorf("task not found: %s", id)
	}
	if err := errors.Join(plan.location.Problems...); err != nil {
		return plan, err
	}
	plan.metaPath, err = task.MetadataPathInDirectory(plan.location.RuntimeDir)
	if err != nil {
		return plan, err
	}
	if plan.metaPath == "" {
		return plan, errors.New("运行记录缺失: terminal eligibility cannot be established; inspect or minimally reconstruct with list first")
	}
	plan.meta, err = task.ReadTaskMeta(plan.metaPath)
	if err != nil {
		return plan, err
	}
	if plan.meta.ID != id {
		return plan, fmt.Errorf("task identity conflict: %s", plan.metaPath)
	}
	state, err := workflow.LoadFromDirectory(plan.location.RuntimeDir, workflow.TaskID(id))
	if errors.Is(err, os.ErrNotExist) {
		state, err = taskworkflow.WorkflowRuntimeFromMeta(plan.meta), nil
	}
	if err != nil {
		return plan, err
	}
	if state.WriteLease != nil {
		return plan, errors.New("Task has an unreleased write lease")
	}
	for _, attempt := range state.Attempts {
		if attempt.State == workflow.AttemptCreated || attempt.State == workflow.AttemptRunning || attempt.State == workflow.AttemptPaused {
			return plan, fmt.Errorf("Task has unfinished attempt %s (%s)", attempt.ID, attempt.State)
		}
	}
	for _, item := range state.WorkItems {
		if string(item.State) == "running" || string(item.State) == "leased" {
			return plan, fmt.Errorf("Task work item %s is occupied", item.ID)
		}
	}
	if err := requireAbsent(filepath.Join(task.RuntimeTasksPath(), "locks", id+".lock")); err != nil {
		return plan, err
	}
	plan.summary = workflow.DeriveSummary(state)
	plan.name = plan.location.ArchiveName
	if plan.name == "" {
		plan.name = task.Today() + "-" + id
	}
	if plan.location.ChangeDir == "" {
		fmt.Fprintln(os.Stderr, "规格已删除:", id)
	} else if !plan.location.Archived {
		plan.moves = append(plan.moves, archiveMove{plan.location.ChangeDir, task.ArchiveTaskDir(plan.name)})
	}
	runtimeTarget := filepath.Join(task.RuntimeRoot(), ".ai", "archive", plan.name)
	if !sameArchivePath(plan.location.RuntimeDir, runtimeTarget) {
		plan.moves = append(plan.moves, archiveMove{plan.location.RuntimeDir, runtimeTarget})
	}
	if plan.meta.Session != "" {
		for _, location := range locations {
			if err := errors.Join(location.Problems...); err != nil {
				return plan, err
			}
			path, err := task.MetadataPathInDirectory(location.RuntimeDir)
			if err != nil {
				return plan, err
			}
			if path == "" {
				continue
			}
			meta, err := task.ReadTaskMeta(path)
			if err != nil {
				return plan, err
			}
			if meta.ID != location.ID {
				return plan, fmt.Errorf("task identity conflict: %s", path)
			}
			if meta.ID != id && meta.Session == plan.meta.Session {
				return plan, fmt.Errorf("Session %s is also bound to Task %s", meta.Session, meta.ID)
			}
		}
		store := session.NewStore(os.Getenv("AIW_SESSION_ROOT"))
		if err := requireAbsent(filepath.Join(store.Root, "locks", plan.meta.Session+".lock")); err != nil {
			return plan, err
		}
		location, err := store.Resolve(plan.meta.Session)
		if errors.Is(err, session.ErrSessionNotFound) {
			// Session is an optional Task attachment. A missing record does not
			// require creating a placeholder or block Task archival.
		} else if err != nil {
			return plan, err
		} else {
			if (location.TaskID != "" && location.TaskID != id) || (location.Status.Task != nil && location.Status.Task.TaskID != "" && location.Status.Task.TaskID != id) {
				return plan, fmt.Errorf("Session %s belongs to another Task", plan.meta.Session)
			}
			if location.TaskID != "" && filepath.Base(filepath.Dir(location.Dir)) != plan.name {
				return plan, fmt.Errorf("Session archive name conflicts with Task archive %s: %s", plan.name, location.Dir)
			}
			if location.Status.Session.State == session.StateRunning {
				return plan, fmt.Errorf("Session %s has an unfinished execution", plan.meta.Session)
			}
			if started := location.Status.Execution.LastStartedAt; started != "" {
				start, startErr := time.Parse(time.RFC3339, started)
				end, endErr := time.Parse(time.RFC3339, location.Status.Execution.LastCompletedAt)
				if startErr != nil || endErr != nil || end.Before(start) {
					return plan, fmt.Errorf("Session %s has unfinished or invalid execution timestamps", plan.meta.Session)
				}
			}
			target := filepath.Join(store.Root, "sessions", "archive", plan.name, plan.meta.Session)
			if !sameArchivePath(location.Dir, target) {
				plan.moves = append(plan.moves, archiveMove{location.Dir, target})
			}
		}
	}
	for _, move := range plan.moves {
		if err := requireAbsent(move.target); err != nil {
			return plan, err
		}
	}
	return plan, nil
}

func moveArchiveDirectory(move archiveMove) error {
	if err := requireAbsent(move.target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(move.target), 0o755); err != nil {
		return err
	}
	return archiveRename(move.source, move.target)
}

func rollbackTaskArchive(cause error, moved []archiveMove) error {
	problems := []error{cause}
	for i := len(moved) - 1; i >= 0; i-- {
		move := moved[i]
		if err := moveArchiveDirectory(archiveMove{move.target, move.source}); err != nil {
			problems = append(problems, fmt.Errorf("restore %s -> %s failed: %w; inspect both paths, remove the conflict manually, then retry archive", move.target, move.source, err))
		} else {
			problems = append(problems, fmt.Errorf("restored %s -> %s", move.target, move.source))
		}
	}
	return errors.Join(problems...)
}

func delegatedArchiveLocation(id string) (string, error) {
	var found string
	for _, root := range []string{task.ArchiveDir, task.LegacyArchiveDir} {
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() || len(name) < 12 || name[10] != '-' || name[11:] != id {
				continue
			}
			if _, err := time.Parse("2006-01-02", name[:10]); err != nil {
				continue
			}
			path := filepath.Join(root, name)
			if found != "" {
				return "", fmt.Errorf("ambiguous delegated archives: %s and %s; inspect before retrying", found, path)
			}
			found = path
		}
	}
	if found == "" {
		return "", fmt.Errorf("OpenSpec did not produce a supported archive for %s; inspect %s and %s", id, task.ArchiveDir, task.LegacyArchiveDir)
	}
	return found, nil
}

func executeTaskArchive(plan taskArchivePlan, bin string) error {
	var moved []archiveMove
	for _, move := range plan.moves {
		if bin != "" && sameArchivePath(move.source, plan.location.ChangeDir) {
			cmd := exec.Command(bin, "archive", "--yes", plan.meta.ID)
			cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
			runErr := cmd.Run()
			actual, locateErr := delegatedArchiveLocation(plan.meta.ID)
			if locateErr != nil {
				return rollbackTaskArchive(errors.Join(runErr, locateErr), moved)
			}
			if err := requireAbsent(move.source); err != nil {
				return rollbackTaskArchive(fmt.Errorf("OpenSpec left source %s and archive %s: %w; inspect both paths before retrying", move.source, actual, errors.Join(runErr, err)), moved)
			}
			moved = append(moved, archiveMove{move.source, actual})
			if runErr != nil || filepath.Base(actual) != plan.name {
				return rollbackTaskArchive(fmt.Errorf("OpenSpec archive expected %s, got %s: %v", plan.name, actual, runErr), moved)
			}
		} else {
			if err := moveArchiveDirectory(move); err != nil {
				return rollbackTaskArchive(fmt.Errorf("archive %s -> %s: %w", move.source, move.target, err), moved)
			}
			moved = append(moved, move)
		}
	}
	verified, err := prepareTaskArchive(plan.meta.ID)
	if err != nil {
		return rollbackTaskArchive(err, moved)
	}
	if len(verified.moves) != 0 || verified.name != plan.name {
		return rollbackTaskArchive(errors.New("archive pairing verification failed"), moved)
	}
	return nil
}

func archiveWithBackend(id string, opts ArchiveOptions, bin string) error {
	plan, err := prepareTaskArchive(id)
	if err != nil {
		return err
	}
	if err := prepareArchiveEligibility(id, opts, plan, bin == ""); err != nil {
		return err
	}
	refreshed, err := prepareTaskArchive(id)
	if err != nil {
		return err
	}
	if refreshed.name != plan.name {
		return errors.New("archive date changed during preparation; retry")
	}
	return executeTaskArchive(refreshed, strings.TrimSpace(bin))
}

func verifyArchivedDelivery(branch, worktree string) error {
	if gitx.WorktreeRegistered(worktree) {
		return errors.New("archived isolated Task still has a registered worktree")
	}
	if _, err := os.Lstat(worktree); err == nil {
		return errors.New("archived isolated Task still has a worktree directory")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run()
	if err == nil {
		return errors.New("archived isolated Task still has its task branch")
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		return fmt.Errorf("verify archived task branch: %w", err)
	}
	return nil
}

func prepareArchiveEligibility(id string, opts ArchiveOptions, plan taskArchivePlan, native bool) error {
	src, meta, summary := plan.location.ChangeDir, plan.meta, plan.summary
	forceClosed := summary.Status == workflow.TaskCancelled
	workflowDone := summary.Status == workflow.TaskDone
	legacyTerminal := meta.Status == "DONE" || meta.Status == "CANCELLED"
	if !forceClosed && !workflowDone && !legacyTerminal {
		return fmt.Errorf("task must be DONE or CANCELLED before archive: %s", meta.Status)
	}
	kind := resolvedWorkspaceKind(meta)
	if kind == "unknown" {
		return errors.New("cannot archive Task with unknown workspace binding; repair it first")
	}
	delivery := summary.Delivery
	if !forceClosed {
		delivery = taskworkflow.WorkflowRuntimeFromMeta(meta).Delivery
	}
	if kind == "unassigned" && delivery != workflow.DeliveryMerged && delivery != workflow.DeliveryDiscarded {
		return errors.New("unassigned Task must record merged or discarded delivery before archive")
	}
	if !plan.location.Archived && src != "" && (workflowDone || (meta.Status == "DONE" && !forceClosed)) {
		if err := syncArchiveWorkflow(id, meta, src); err != nil {
			return err
		}
	}
	problems := []string{}
	if !plan.location.Archived && src != "" {
		problems = archiveArtifactProblems(src)
	}
	if len(problems) > 0 && !opts.Force {
		if err := repairArchiveArtifacts(src, problems); err != nil {
			fmt.Fprintf(os.Stderr, "archive repair failed: %v\n", err)
		}
		problems = archiveArtifactProblems(src)
		if len(problems) > 0 && !confirmArchiveWithWarnings(problems) {
			return errors.New("archive cancelled")
		}
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
		if !plan.location.Archived {
			if dirty, _ := gitx.IsDirty(); dirty {
				fmt.Fprintln(os.Stderr, "warning: archiving primary Task with unmanaged Git delivery and uncommitted changes")
			}
		}
	}

	branch := strings.TrimSpace(meta.Branch)
	if branch == "" {
		branch = "feature/" + id
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(task.WorktreeDir, id))
	}

	if plan.location.Archived {
		if (kind == "isolated" || kind == "unassigned") && delivery != workflow.DeliveryDiscarded {
			return verifyArchivedDelivery(branch, wt)
		}
		return nil
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
	if native && src != "" {
		if err := syncSpecSnapshots(src, meta.Specs); err != nil {
			return err
		}
	}
	return nil
}

func syncArchiveWorkflow(id string, meta task.TaskMeta, changeDir string) error {
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(taskworkflow.WorkflowRuntimeFromMeta(meta)); err != nil {
		return err
	}
	state, err := taskworkflow.SyncWorkflowChecklistAtPath(id, store, filepath.Join(changeDir, "tasks.md"))
	if err != nil {
		return fmt.Errorf("archive requires workflow sync: %w", err)
	}
	return taskworkflow.ProjectWorkflowState(id, state)
}

func archiveArtifactProblems(src string) []string {
	problems := []string{}
	tasksPath := filepath.Join(src, "tasks.md")
	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return []string{"tasks.md is missing or unreadable"}
	}
	_, diagnostics := task.ParseNumberedChecklist(string(content))
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
	targetDir := filepath.Join(task.SpecsDir, category)
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
