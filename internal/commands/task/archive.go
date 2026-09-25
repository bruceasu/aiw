package task

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/gitx"
	"aiw/internal/session"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// A single seam keeps fault injection at the existing CLI boundary.
var archiveRename = os.Rename

type archiveMove struct { source, target string }
type taskArchivePlan struct {
	location taskx.TaskLocation
	meta taskx.TaskMeta
	metaPath string
	summary workflow.TaskSummary
	name string
	moves []archiveMove
}

func sameArchivePath(a, b string) bool {
	a, errA := filepath.Abs(a)
	b, errB := filepath.Abs(b)
	return errA == nil && errB == nil && filepath.Clean(a) == filepath.Clean(b)
}

func requireAbsent(path string) error {
	_, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) { return nil }
	if err != nil { return err }
	return fmt.Errorf("archive target or writer lock already exists: %s", path)
}

func prepareTaskArchive(id string) (taskArchivePlan, error) {
	plan := taskArchivePlan{}
	if !safeID(id) { return plan, errors.New("invalid task id") }
	locations, err := taskx.DiscoverTaskLocations()
	if err != nil { return plan, err }
	found := false
	for _, location := range locations {
		if location.ID == id { plan.location, found = location, true; break }
	}
	if !found { return plan, fmt.Errorf("task not found: %s", id) }
	if err := errors.Join(plan.location.Problems...); err != nil { return plan, err }
	plan.metaPath, err = taskx.MetadataPathInDirectory(plan.location.RuntimeDir)
	if err != nil { return plan, err }
	if plan.metaPath == "" { return plan, errors.New("运行记录缺失: terminal eligibility cannot be established; inspect or minimally reconstruct with list first") }
	plan.meta, err = taskx.ReadTaskMeta(plan.metaPath)
	if err != nil { return plan, err }
	if plan.meta.ID != id { return plan, fmt.Errorf("task identity conflict: %s", plan.metaPath) }
	state, err := workflow.LoadFromDirectory(plan.location.RuntimeDir, workflow.TaskID(id))
	if errors.Is(err, os.ErrNotExist) { state, err = taskx.WorkflowRuntimeFromMeta(plan.meta), nil }
	if err != nil { return plan, err }
	if state.WriteLease != nil { return plan, errors.New("Task has an unreleased write lease") }
	for _, attempt := range state.Attempts {
		if attempt.State == workflow.AttemptCreated || attempt.State == workflow.AttemptRunning || attempt.State == workflow.AttemptPaused {
			return plan, fmt.Errorf("Task has unfinished attempt %s (%s)", attempt.ID, attempt.State)
		}
	}
	for _, item := range state.WorkItems {
		if string(item.State) == "running" || string(item.State) == "leased" { return plan, fmt.Errorf("Task work item %s is occupied", item.ID) }
	}
	if err := requireAbsent(filepath.Join(taskx.RuntimeTasksPath(), "locks", id+".lock")); err != nil { return plan, err }
	plan.summary = workflow.DeriveSummary(state)
	plan.name = plan.location.ArchiveName
	if plan.name == "" { plan.name = taskx.Today()+"-"+id }
	if plan.location.ChangeDir == "" {
		fmt.Fprintln(os.Stderr, "规格已删除:", id)
	} else if !plan.location.Archived {
		plan.moves = append(plan.moves, archiveMove{plan.location.ChangeDir, taskx.ArchiveTaskDir(plan.name)})
	}
	runtimeTarget := filepath.Join(taskx.RuntimeRoot(), ".ai", "archive", plan.name)
	if !sameArchivePath(plan.location.RuntimeDir, runtimeTarget) {
		plan.moves = append(plan.moves, archiveMove{plan.location.RuntimeDir, runtimeTarget})
	}
	if plan.meta.Session != "" {
		// Fail closed if any supported record cannot establish exclusive binding.
		for _, location := range locations {
			if err := errors.Join(location.Problems...); err != nil { return plan, err }
			path, err := taskx.MetadataPathInDirectory(location.RuntimeDir)
			if err != nil { return plan, err }
			if path == "" { continue }
			meta, err := taskx.ReadTaskMeta(path)
			if err != nil { return plan, err }
			if meta.ID != location.ID { return plan, fmt.Errorf("task identity conflict: %s", path) }
			if meta.ID != id && meta.Session == plan.meta.Session { return plan, fmt.Errorf("Session %s is also bound to Task %s", meta.Session, meta.ID) }
		}
		store := session.NewStore(os.Getenv("AIW_SESSION_ROOT"))
		if err := requireAbsent(filepath.Join(store.Root, "locks", plan.meta.Session+".lock")); err != nil { return plan, err }
		location, err := store.Resolve(plan.meta.Session)
		if errors.Is(err, session.ErrSessionNotFound) {
			fmt.Fprintln(os.Stderr, "会话记录缺失:", plan.meta.Session)
		} else if err != nil { return plan, err
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
				if startErr != nil || endErr != nil || end.Before(start) { return plan, fmt.Errorf("Session %s has unfinished or invalid execution timestamps", plan.meta.Session) }
			}
			target := filepath.Join(store.Root, "sessions", "archive", plan.name, plan.meta.Session)
			if !sameArchivePath(location.Dir, target) { plan.moves = append(plan.moves, archiveMove{location.Dir, target}) }
		}
	}
	for _, move := range plan.moves {
		if err := requireAbsent(move.target); err != nil { return plan, err }
	}
	return plan, nil
}

func moveArchiveDirectory(move archiveMove) error {
	if err := requireAbsent(move.target); err != nil { return err }
	if err := os.MkdirAll(filepath.Dir(move.target), 0o755); err != nil { return err }
	return archiveRename(move.source, move.target)
}

func rollbackTaskArchive(cause error, moved []archiveMove) error {
	problems := []error{cause}
	for i := len(moved)-1; i >= 0; i-- {
		move := moved[i]
		if err := moveArchiveDirectory(archiveMove{move.target, move.source}); err != nil {
			problems = append(problems, fmt.Errorf("restore %s -> %s failed: %w; inspect both paths, remove the conflict manually, then retry archive", move.target, move.source, err))
		} else {
			problems = append(problems, fmt.Errorf("restored %s -> %s", move.target, move.source))
		}
	}
	return errors.Join(problems...)
}

// delegatedArchiveLocation verifies the actual result even after a failing CLI.
func delegatedArchiveLocation(id string) (string, error) {
	var found string
	for _, root := range []string{taskx.ArchiveDir, taskx.LegacyArchiveDir} {
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) { continue }
		if err != nil { return "", err }
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() || len(name) < 12 || name[10] != '-' || name[11:] != id { continue }
			if _, err := time.Parse("2006-01-02", name[:10]); err != nil { continue }
			path := filepath.Join(root, name)
			if found != "" { return "", fmt.Errorf("ambiguous delegated archives: %s and %s; inspect before retrying", found, path) }
			found = path
		}
	}
	if found == "" { return "", fmt.Errorf("OpenSpec did not produce a supported archive for %s; inspect %s and %s", id, taskx.ArchiveDir, taskx.LegacyArchiveDir) }
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
			if locateErr != nil { return rollbackTaskArchive(errors.Join(runErr, locateErr), moved) }
			if err := requireAbsent(move.source); err != nil {
				return rollbackTaskArchive(fmt.Errorf("OpenSpec left source %s and archive %s: %w; inspect both paths before retrying", move.source, actual, errors.Join(runErr, err)), moved)
			}
			moved = append(moved, archiveMove{move.source, actual})
			if runErr != nil || filepath.Base(actual) != plan.name {
				return rollbackTaskArchive(fmt.Errorf("OpenSpec archive expected %s, got %s: %v", plan.name, actual, runErr), moved)
			}
		} else {
			if err := moveArchiveDirectory(move); err != nil { return rollbackTaskArchive(fmt.Errorf("archive %s -> %s: %w", move.source, move.target, err), moved) }
			moved = append(moved, move)
		}
	}
	// Re-resolve identities and actual locations before reporting success.
	verified, err := prepareTaskArchive(plan.meta.ID)
	if err != nil { return rollbackTaskArchive(err, moved) }
	if len(verified.moves) != 0 || verified.name != plan.name { return rollbackTaskArchive(errors.New("archive pairing verification failed"), moved) }
	return nil
}

func archiveWithBackend(id string, opts ArchiveOptions, bin string) error {
	plan, err := prepareTaskArchive(id)
	if err != nil { return err }
	if err := prepareArchiveEligibility(id, opts, plan, bin == ""); err != nil { return err }
	// Eligibility synchronization may refresh the runtime. Preflight again before moving.
	refreshed, err := prepareTaskArchive(id)
	if err != nil { return err }
	if refreshed.name != plan.name { return errors.New("archive date changed during preparation; retry") }
	return executeTaskArchive(refreshed, strings.TrimSpace(bin))
}

func verifyArchivedDelivery(branch, worktree string) error {
	if gitx.WorktreeRegistered(worktree) { return errors.New("archived isolated Task still has a registered worktree") }
	if _, err := os.Lstat(worktree); err == nil { return errors.New("archived isolated Task still has a worktree directory") } else if !errors.Is(err, os.ErrNotExist) { return err }
	err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run()
	if err == nil { return errors.New("archived isolated Task still has its task branch") }
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 { return fmt.Errorf("verify archived task branch: %w", err) }
	return nil
}
