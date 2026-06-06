package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TaskMeta struct {
	ID       string
	Type     string
	Status   string
	Created  string
	Updated  string
	Branch   string
	Worktree string
	Specs    []string
	Tags     []string
}

type RegistryEntry struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Branch    string `json:"branch"`
	Worktree  string `json:"worktree"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
}

type ArchiveOptions struct {
	Push         bool
	CleanupWT    bool
	DeleteBranch bool
}

func newTask(id string) error {
	if !safeID(id) {
		return errors.New("invalid task id")
	}
	dir := filepath.Join(changesDir, id)
	if exists(dir) {
		return fmt.Errorf("task already exists: %s", dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	meta := TaskMeta{
		ID:       id,
		Type:     "task",
		Status:   "TODO",
		Created:  today(),
		Updated:  today(),
		Branch:   "feature/" + id,
		Worktree: filepath.ToSlash(filepath.Join(worktreeDir, id)),
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
	if err := writeTaskMeta(filepath.Join(dir, "task.toml"), meta); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "task.md"), []byte(taskMD), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(notesMD), 0644); err != nil {
		return err
	}
	return writeRegistry()
}

func createDecision(id string) error {
	dir := filepath.Join(changesDir, id)
	if !exists(dir) {
		return fmt.Errorf("task not found: %s", id)
	}
	design := filepath.Join(dir, "design.md")
	if exists(design) {
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
	return os.WriteFile(design, []byte(content), 0644)
}

func createSpec(id string) error {
	dir := filepath.Join(specsDir, id)
	if exists(dir) {
		return fmt.Errorf("spec already exists: %s", id)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	meta := `id = "` + id + `"
type = "spec"
status = "active"
created = "` + today() + `"
updated = "` + today() + `"
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
	if err := os.WriteFile(filepath.Join(dir, "spec.toml"), []byte(meta), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "spec.md"), []byte(spec), 0644)
}

func listTasks() error {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := readTaskMeta(filepath.Join(changesDir, e.Name(), "task.toml"))
		if err != nil {
			continue
		}
		fmt.Printf("%-24s %-12s %s\n",
			meta.ID,
			meta.Status,
			filepath.ToSlash(filepath.Join(changesDir, e.Name())),
		)
	}
	return nil
}

func showTask(id string) error {
	path := filepath.Join(changesDir, id, "task.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}

func updateStatus(id, status string) error {
	metaPath := filepath.Join(changesDir, id, "task.toml")
	meta, err := readTaskMeta(metaPath)
	if err != nil {
		return err
	}
	meta.Status = strings.ToUpper(status)
	meta.Updated = today()
	if err := writeTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return writeRegistry()
}

func archiveTask(id string, opts ArchiveOptions) error {
	src := filepath.Join(changesDir, id)
	if !exists(src) {
		return fmt.Errorf("task not found: %s", id)
	}

	metaPath := filepath.Join(src, "task.toml")
	meta, err := readTaskMeta(metaPath)
	if err != nil {
		return err
	}

	branch := strings.TrimSpace(meta.Branch)
	if branch == "" {
		branch = "feature/" + id
	}

	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(worktreeDir, id))
	}

	if opts.Push {
		if err := run("git", "push", "-u", "origin", branch); err != nil {
			return err
		}
	}

	if opts.CleanupWT {
		if err := run("git", "worktree", "remove", wt); err != nil {
			return err
		}
	}

	if opts.DeleteBranch {
		if err := run("git", "branch", "-d", branch); err != nil {
			return err
		}
	}

	dst := filepath.Join(archiveDir, today()+"-"+id)
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return writeRegistry()
}

func printContext(id string) error {
	changeDir := filepath.Join(changesDir, id)
	if !exists(changeDir) {
		return fmt.Errorf("task not found: %s", id)
	}
	fmt.Println("Read these files first:\n")
	files := []string{
		filepath.Join(changeDir, "task.toml"),
		filepath.Join(changeDir, "task.md"),
		filepath.Join(changeDir, "design.md"),
		filepath.Join(changeDir, "notes.md"),
	}
	for _, f := range files {
		if exists(f) {
			fmt.Println("-", filepath.ToSlash(f))
		}
	}
	meta, err := readTaskMeta(filepath.Join(changeDir, "task.toml"))
	if err == nil {
		for _, spec := range meta.Specs {
			fmt.Println("-", filepath.ToSlash(filepath.Join(specsDir, spec, "spec.md")))
		}
	}
	fmt.Println(`
Instruction:
- implement only the scoped task
- avoid unrelated refactors
- preserve backward compatibility
- update TODO and Verification before finishing
- use %% notes instead of guessing
`)
	return nil
}

func readTaskMeta(path string) (TaskMeta, error) {
	file, err := os.Open(path)
	if err != nil {
		return TaskMeta{}, err
	}
	defer file.Close()
	meta := TaskMeta{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"`)
		switch key {
		case "id":
			meta.ID = value
		case "type":
			meta.Type = value
		case "status":
			meta.Status = value
		case "created":
			meta.Created = value
		case "updated":
			meta.Updated = value
		case "branch":
			meta.Branch = value
		case "worktree":
			meta.Worktree = value
		}
	}
	return meta, scanner.Err()
}

func writeTaskMeta(path string, meta TaskMeta) error {
	content := fmt.Sprintf(`id = "%s"
type = "%s"
status = "%s"
created = "%s"
updated = "%s"
branch = "%s"
worktree = "%s"
`,
		meta.ID,
		meta.Type,
		meta.Status,
		meta.Created,
		meta.Updated,
		meta.Branch,
		meta.Worktree,
	)
	return os.WriteFile(path, []byte(content), 0644)
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
