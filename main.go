package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	openspecDir   = "openspec"
	changesDir    = "openspec/changes"
	specsDir      = "openspec/specs"
	archiveDir    = "openspec/archive"
	registryFile  = "openspec/registry.json"
	worktreeDir   = ".wt"
	gitignoreFile = ".gitignore"
	promptsDir    = "docs/agent-templates"
	agentsFile    = "AGENTS.md"
	codexFile     = "CODEX.md"
	copilotFile   = ".github/copilot-instructions.md"
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

type RemoveWorktreeOptions struct {
	DeleteBranch bool
	Force        bool
}

type WorktreeListOptions struct {
	Porcelain bool
}

type WorktreePruneOptions struct {
	DryRun bool
}

type PromptOptions struct {
	Template string
	Merge    bool
	Force    bool
	List     bool
}

type InitOptions struct {
	Prompts     PromptOptions
	WithPrompts bool
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	var err error
	switch os.Args[1] {
	case "init":
		opts, parseErr := parseInitOptions(os.Args[2:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = initWorkspace(opts)
	case "new":
		requireArgs(3, "new <task-id>")
		err = newTask(os.Args[2])
	case "list":
		err = listTasks()
	case "show":
		requireArgs(3, "show <task-id>")
		err = showTask(os.Args[2])
	case "status":
		requireArgs(4, "status <task-id> <status>")
		err = updateStatus(os.Args[2], os.Args[3])
	case "done":
		requireArgs(3, "done <task-id>")
		err = updateStatus(os.Args[2], "DONE")
	case "archive":
		requireArgs(3, "archive <task-id> [--push] [--cleanup-wt] [--delete-branch]")
		opts, parseErr := parseArchiveOptions(os.Args[3:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = archiveTask(os.Args[2], opts)
	case "wt":
		requireArgs(3, "wt <task-id> [base-branch]")
		base := "origin/main"
		if len(os.Args) >= 4 {
			base = os.Args[3]
		}
		err = createWorktree(os.Args[2], base)
	case "wt-rm":
		requireArgs(3, "wt-rm <task-id> [--delete-branch] [--force]")
		opts, parseErr := parseRemoveWorktreeOptions(os.Args[3:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = removeWorktree(os.Args[2], opts)
	case "wt-list":
		opts, parseErr := parseWorktreeListOptions(os.Args[2:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = listWorktrees(opts)
	case "wt-prune":
		opts, parseErr := parseWorktreePruneOptions(os.Args[2:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = pruneWorktrees(opts)
	case "wt-lock":
		requireArgs(3, "wt-lock <task-id> [reason]")
		reason := strings.TrimSpace(strings.Join(os.Args[3:], " "))
		err = lockWorktree(os.Args[2], reason)
	case "wt-unlock":
		requireArgs(3, "wt-unlock <task-id>")
		err = unlockWorktree(os.Args[2])
	case "wt-repair":
		err = repairWorktrees()
	case "context":
		requireArgs(3, "context <task-id>")
		err = printContext(os.Args[2])
	case "decision":
		requireArgs(3, "decision <task-id>")
		err = createDecision(os.Args[2])
	case "spec":
		requireArgs(3, "spec <spec-id>")
		err = createSpec(os.Args[2])
	case "registry":
		err = writeRegistry()
	case "prompts":
		opts, parseErr := parsePromptOptions(os.Args[2:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = syncPrompts(opts)
	case "ignore-wt":
		err = ensureWorktreeIgnored()
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
func usage() {
	fmt.Println(`OpenSpec-lite TOML Workspace CLI
Usage:
	aiw init [--prompts] [--merge] [--force] [--template <name>]
	aiw new <task-id>
	aiw list
	aiw show <task-id>
	aiw status <task-id> <status>
	aiw done <task-id>
	aiw archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--finalize]
	aiw wt <task-id> [base-branch]
	aiw wt-rm <task-id> [--delete-branch] [--force]
	aiw wt-list [--porcelain]
	aiw wt-prune [--dry-run]
	aiw wt-lock <task-id> [reason]
	aiw wt-unlock <task-id>
	aiw wt-repair
	aiw context <task-id>
	aiw decision <task-id>
	aiw spec <spec-id>
	aiw registry
	aiw prompts list
	aiw prompts [template] [--merge] [--force]
	aiw ignore-wt

Command Summary:
	init                      Initialize workspace; can also sync prompts in one step.
	new <task-id>             Create a new task folder with task.toml/task.md/notes.md.
	list                      List tasks from openspec/changes.
	show <task-id>            Print openspec/changes/<task-id>/task.md.
	status <task-id> <status> Update task status (auto upper-case) and updated date.
	done <task-id>            Shortcut for: status <task-id> DONE.
	archive ...               Move task to openspec/archive; supports push/cleanup/delete.
	wt <task-id> [base]       Create worktree + branch (default base: origin/main).
	wt-rm ...                 Remove task worktree (safe by default; --force optional).
	wt-list [--porcelain]     Show all registered worktrees.
	wt-prune [--dry-run]      Clean stale worktree metadata (or preview with --dry-run).
	wt-lock <task-id> [why]   Lock a task worktree to prevent accidental removal/prune.
	wt-unlock <task-id>       Unlock a previously locked task worktree.
	wt-repair                 Repair worktree links after repo/path moves.
	context <task-id>         Show files/instructions to read before implementation.
	decision <task-id>        Create design.md for a task when design is needed.
	spec <spec-id>            Create long-lived spec under openspec/specs.
	registry                  Rebuild openspec/registry.json.
	prompts list              List available prompts templates under docs/agent-templates.
	prompts [template] ...    Create or merge AGENTS/CODEX/Copilot prompts from docs templates.
	ignore-wt                 Create or append .wt/ ignore rule into .gitignore.

Examples:
	aiw init
	aiw init --prompts --merge
	aiw init --prompts --template go
	aiw new payment-retry
	aiw wt payment-retry origin/main
	aiw wt-lock payment-retry "hotfix in progress"
	aiw status payment-retry IN_PROGRESS
	aiw archive payment-retry --finalize

	aiw wt-list --porcelain
	aiw wt-prune --dry-run
	aiw wt-rm payment-retry --delete-branch --force
	aiw prompts list
	aiw prompts go --merge
	aiw ignore-wt

`)
}
func requireArgs(n int, syntax string) {
	if len(os.Args) < n {
		fmt.Println("usage:", syntax)
		os.Exit(1)
	}
}
func initWorkspace(opts InitOptions) error {
	dirs := []string{
		openspecDir,
		changesDir,
		specsDir,
		archiveDir,
		worktreeDir,
		filepath.Dir(copilotFile),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	if err := writeIfMissing(agentsFile, agentsTemplate()); err != nil {
		return err
	}
	if err := writeIfMissing(copilotFile, copilotTemplate()); err != nil {
		return err
	}
	if err := ensureWorktreeIgnored(); err != nil {
		return err
	}
	if err := writeRegistry(); err != nil {
		return err
	}
	if opts.WithPrompts {
		return syncPrompts(opts.Prompts)
	}
	return nil
}

func syncPrompts(opts PromptOptions) error {
	if opts.List {
		return listPromptTemplates()
	}

	templatesRoot, err := resolvePromptTemplatesDir()
	if err != nil {
		return err
	}

	template := opts.Template
	if template == "" {
		detected, err := detectPromptTemplate()
		if err != nil {
			return err
		}
		template = detected
	}

	baseAgents, err := readOptionalTemplate(filepath.Join(templatesRoot, agentsFile))
	if err != nil {
		return err
	}
	langAgents, err := readOptionalTemplate(
		filepath.Join(templatesRoot, template, agentsFile),
		filepath.Join(templatesRoot, template, "ANGETS.md"),
	)
	if err != nil {
		return err
	}
	langCopilot, err := readOptionalTemplate(filepath.Join(templatesRoot, template, filepath.Base(copilotFile)))
	if err != nil {
		return err
	}
	langCodex, err := readOptionalTemplate(filepath.Join(templatesRoot, template, codexFile))
	if err != nil {
		return err
	}

	if strings.TrimSpace(baseAgents) == "" && strings.TrimSpace(langAgents) == "" {
		return fmt.Errorf("no AGENTS templates found for %s under %s", template, templatesRoot)
	}

	if err := os.MkdirAll(filepath.Dir(copilotFile), 0755); err != nil {
		return err
	}

	var actions []string
	agentsContent := joinPromptSections(baseAgents, langAgents)
	action, err := applyPromptTarget(agentsFile, agentsContent, opts, fmt.Sprintf("aiw-prompts:%s:agents", template))
	if err != nil {
		return err
	}
	if action != "" {
		actions = append(actions, fmt.Sprintf("%s %s", action, filepath.ToSlash(agentsFile)))
	}
	if strings.TrimSpace(langCopilot) != "" {
		action, err := applyPromptTarget(copilotFile, langCopilot, opts, fmt.Sprintf("aiw-prompts:%s:copilot", template))
		if err != nil {
			return err
		}
		if action != "" {
			actions = append(actions, fmt.Sprintf("%s %s", action, filepath.ToSlash(copilotFile)))
		}
	}
	if strings.TrimSpace(langCodex) != "" {
		action, err := applyPromptTarget(codexFile, langCodex, opts, fmt.Sprintf("aiw-prompts:%s:codex", template))
		if err != nil {
			return err
		}
		if action != "" {
			actions = append(actions, fmt.Sprintf("%s %s", action, filepath.ToSlash(codexFile)))
		}
	}

	fmt.Println("prompts template:", template)
	if len(actions) == 0 {
		fmt.Println("prompts result: no files changed")
		return nil
	}
	fmt.Println("prompts result:")
	for _, action := range actions {
		fmt.Println("-", action)
	}
	return nil
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

func removeWorktree(id string, opts RemoveWorktreeOptions) error {
	taskDir := filepath.Join(changesDir, id)
	if !exists(taskDir) {
		return fmt.Errorf("task not found: %s", id)
	}

	metaPath := filepath.Join(taskDir, "task.toml")
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

	removeArgs := []string{"worktree", "remove", wt}
	if opts.Force {
		removeArgs = append(removeArgs, "--force")
	}
	if err := run("git", removeArgs...); err != nil {
		return err
	}

	meta.Worktree = ""

	if opts.DeleteBranch {
		if err := run("git", "branch", "-d", branch); err != nil {
			return err
		}
		meta.Branch = ""
	}

	meta.Updated = today()
	if err := writeTaskMeta(metaPath, meta); err != nil {
		return err
	}
	return writeRegistry()
}

func listWorktrees(opts WorktreeListOptions) error {
	args := []string{"worktree", "list"}
	if opts.Porcelain {
		args = append(args, "--porcelain")
	}
	return run("git", args...)
}

func pruneWorktrees(opts WorktreePruneOptions) error {
	args := []string{"worktree", "prune"}
	if opts.DryRun {
		args = append(args, "-n", "-v")
	}
	return run("git", args...)
}

func lockWorktree(id, reason string) error {
	wt, err := resolveTaskWorktree(id)
	if err != nil {
		return err
	}
	args := []string{"worktree", "lock", wt}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	return run("git", args...)
}

func unlockWorktree(id string) error {
	wt, err := resolveTaskWorktree(id)
	if err != nil {
		return err
	}
	return run("git", "worktree", "unlock", wt)
}

func repairWorktrees() error {
	return run("git", "worktree", "repair")
}

func createWorktree(id, base string) error {
	taskDir := filepath.Join(changesDir, id)
	if !exists(taskDir) {
		return fmt.Errorf("task not found: %s", id)
	}
	branch := "feature/" + id
	wt := filepath.Join(worktreeDir, id)
	if err := run("git", "fetch", "origin"); err != nil {
		return err
	}
	if err := run("git", "worktree", "add", wt, "-b", branch, base); err != nil {
		return err
	}
	metaPath := filepath.Join(taskDir, "task.toml")
	meta, err := readTaskMeta(metaPath)
	if err != nil {
		return err
	}
	meta.Branch = branch
	meta.Worktree = filepath.ToSlash(wt)
	meta.Updated = today()
	if err := writeTaskMeta(metaPath, meta); err != nil {
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
func writeRegistry() error {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return err
	}
	var changes []RegistryEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := readTaskMeta(filepath.Join(changesDir, e.Name(), "task.toml"))
		if err != nil {
			continue
		}
		changes = append(changes, RegistryEntry{
			ID:        meta.ID,
			Status:    meta.Status,
			Branch:    meta.Branch,
			Worktree:  meta.Worktree,
			Path:      filepath.ToSlash(filepath.Join(changesDir, e.Name())),
			UpdatedAt: meta.Updated,
		})
	}
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].ID < changes[j].ID
	})
	payload := map[string]any{
		"version": "1",
		"updated": time.Now().Format(time.RFC3339),
		"changes": changes,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(registryFile, b, 0644)
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
func agentsTemplate() string {
	return `# AGENTS.md
This repository uses OpenSpec-lite TOML workflow.
Before coding:
- read openspec/changes/<task>/task.toml
- read openspec/changes/<task>/task.md
- read design.md if exists
- read related specs under openspec/specs/
Rules:
- one task at a time
- avoid unrelated refactors
- preserve backward compatibility
- update TODO and Verification
- use %% notes for uncertainties
`
}
func copilotTemplate() string {
	return `# Copilot Instructions
Use OpenSpec-lite TOML workflow.
Always check:
- openspec/changes/
- openspec/specs/
Keep changes scoped.
Avoid broad refactors.
`
}
func writeIfMissing(path, content string) error {
	if exists(path) {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func ensureWorktreeIgnored() error {
	entry := worktreeDir + "/"
	if !exists(gitignoreFile) {
		if err := os.WriteFile(gitignoreFile, []byte(entry+"\n"), 0644); err != nil {
			return err
		}
		fmt.Println("created:", gitignoreFile)
		return nil
	}
	b, err := os.ReadFile(gitignoreFile)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == worktreeDir || trimmed == entry {
			fmt.Println("exists:", gitignoreFile, entry)
			return nil
		}
	}
	content := string(b)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"
	if err := os.WriteFile(gitignoreFile, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Println("updated:", gitignoreFile, entry)
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

func resolveTaskWorktree(id string) (string, error) {
	taskDir := filepath.Join(changesDir, id)
	if !exists(taskDir) {
		return "", fmt.Errorf("task not found: %s", id)
	}
	meta, err := readTaskMeta(filepath.Join(taskDir, "task.toml"))
	if err != nil {
		return "", err
	}
	wt := strings.TrimSpace(meta.Worktree)
	if wt == "" {
		wt = filepath.ToSlash(filepath.Join(worktreeDir, id))
	}
	return wt, nil
}

func detectPromptTemplate() (string, error) {
	if exists("go.mod") {
		return "go", nil
	}
	if exists("pom.xml") || exists("build.gradle") || exists("build.gradle.kts") {
		return "java", nil
	}
	if exists("pyproject.toml") || exists("requirements.txt") || exists("setup.py") {
		return "python", nil
	}
	return "", errors.New("cannot auto-detect prompts template; pass one explicitly, e.g.: aiw prompts go --merge")
}

func readOptionalTemplate(paths ...string) (string, error) {
	for _, path := range paths {
		if !exists(path) {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", nil
}

func joinPromptSections(parts ...string) string {
	var sections []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		sections = append(sections, trimmed)
	}
	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n---\n\n") + "\n"
}

func applyPromptTarget(path, content string, opts PromptOptions, marker string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	content = ensureTrailingNewline(content)
	if opts.Force {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", err
		}
		return "wrote", nil
	}
	if !exists(path) {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", err
		}
		return "created", nil
	}
	if !opts.Merge {
		return "skipped existing", nil
	}
	existingBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	merged := mergePromptSection(string(existingBytes), content, marker)
	if err := os.WriteFile(path, []byte(merged), 0644); err != nil {
		return "", err
	}
	return "merged", nil
}

func listPromptTemplates() error {
	templatesRoot, err := resolvePromptTemplatesDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(templatesRoot)
	if err != nil {
		return err
	}
	var templates []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		templates = append(templates, entry.Name())
	}
	sort.Strings(templates)
	if len(templates) == 0 {
		fmt.Println("no prompt templates found")
		return nil
	}
	fmt.Println("available prompts templates:")
	for _, template := range templates {
		fmt.Println("-", template)
	}
	return nil
}

func resolvePromptTemplatesDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}

	resolvedPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = resolvedPath
	}

	appRoot := filepath.Dir(exePath)
	return filepath.Join(appRoot, promptsDir), nil
}

func mergePromptSection(existing, content, marker string) string {
	begin := "<!-- " + marker + " begin -->"
	end := "<!-- " + marker + " end -->"
	block := begin + "\n" + strings.TrimRight(content, "\n") + "\n" + end + "\n"
	start := strings.Index(existing, begin)
	stop := strings.Index(existing, end)
	if start >= 0 && stop >= start {
		stop += len(end)
		replaced := existing[:start] + block + existing[stop:]
		return ensureTrailingNewline(strings.TrimSpace(replaced))
	}
	trimmed := strings.TrimSpace(existing)
	if trimmed == "" {
		return block
	}
	return ensureTrailingNewline(trimmed) + "\n" + block
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
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

func parseRemoveWorktreeOptions(args []string) (RemoveWorktreeOptions, error) {
	allowed := map[string]bool{
		"--delete-branch": true,
		"--force":         true,
	}
	for _, a := range args {
		if !allowed[a] {
			return RemoveWorktreeOptions{}, fmt.Errorf("unknown wt-rm option: %s", a)
		}
	}
	return RemoveWorktreeOptions{
		DeleteBranch: hasFlag(args, "--delete-branch"),
		Force:        hasFlag(args, "--force"),
	}, nil
}

func parseWorktreeListOptions(args []string) (WorktreeListOptions, error) {
	allowed := map[string]bool{
		"--porcelain": true,
	}
	for _, a := range args {
		if !allowed[a] {
			return WorktreeListOptions{}, fmt.Errorf("unknown wt-list option: %s", a)
		}
	}
	return WorktreeListOptions{Porcelain: hasFlag(args, "--porcelain")}, nil
}

func parseWorktreePruneOptions(args []string) (WorktreePruneOptions, error) {
	allowed := map[string]bool{
		"--dry-run": true,
	}
	for _, a := range args {
		if !allowed[a] {
			return WorktreePruneOptions{}, fmt.Errorf("unknown wt-prune option: %s", a)
		}
	}
	return WorktreePruneOptions{DryRun: hasFlag(args, "--dry-run")}, nil
}

func parsePromptOptions(args []string) (PromptOptions, error) {
	opts := PromptOptions{}
	for _, arg := range args {
		switch arg {
		case "list":
			if opts.Template != "" || opts.Merge || opts.Force || opts.List {
				return PromptOptions{}, errors.New("prompts list cannot be combined with template or flags")
			}
			opts.List = true
		case "--merge":
			if opts.List {
				return PromptOptions{}, errors.New("prompts list cannot be combined with --merge")
			}
			opts.Merge = true
		case "--force":
			if opts.List {
				return PromptOptions{}, errors.New("prompts list cannot be combined with --force")
			}
			opts.Force = true
		default:
			if strings.HasPrefix(arg, "--") {
				return PromptOptions{}, fmt.Errorf("unknown prompts option: %s", arg)
			}
			if opts.List {
				return PromptOptions{}, errors.New("prompts list cannot be combined with a template")
			}
			if opts.Template != "" {
				return PromptOptions{}, fmt.Errorf("multiple prompts templates provided: %s, %s", opts.Template, arg)
			}
			opts.Template = arg
		}
	}
	if opts.Merge && opts.Force {
		return PromptOptions{}, errors.New("--merge and --force cannot be used together")
	}
	return opts, nil
}

func parseInitOptions(args []string) (InitOptions, error) {
	opts := InitOptions{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--prompts":
			opts.WithPrompts = true
		case "--merge":
			opts.Prompts.Merge = true
		case "--force":
			opts.Prompts.Force = true
		case "--template":
			if index+1 >= len(args) {
				return InitOptions{}, errors.New("missing value for --template")
			}
			index++
			opts.Prompts.Template = args[index]
		case "--help", "-h":
			usage()
			os.Exit(0)
		default:
			return InitOptions{}, fmt.Errorf("unknown init option: %s", arg)
		}
	}
	if opts.Prompts.Merge && opts.Prompts.Force {
		return InitOptions{}, errors.New("--merge and --force cannot be used together")
	}
	if (opts.Prompts.Merge || opts.Prompts.Force || opts.Prompts.Template != "") && !opts.WithPrompts {
		return InitOptions{}, errors.New("--merge, --force, and --template require --prompts")
	}
	return opts, nil
}

func today() string {
	return time.Now().Format("2006-01-02")
}
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
