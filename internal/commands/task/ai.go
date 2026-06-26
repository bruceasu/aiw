package task

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	plug "aiw/internal/plugin"
	"aiw/internal/taskx"
)

type AIWorkflowOptions struct {
	Session string
	Last    bool
	Apply   bool
	DryRun  bool
	Prompt  string
}

func runAIWorkflow(args []string) error {
	if len(args) == 0 || isHelpFlag(args[0]) {
		fmt.Print(aiWorkflowUsage())
		return nil
	}

	action := args[0]
	if len(args) == 1 {
		if usage, ok := aiActionUsage(action); ok {
			fmt.Print(usage)
			return nil
		}
		return fmt.Errorf("%s", strings.TrimSpace(aiWorkflowUsage()))
	}
	if isHelpFlag(args[1]) {
		if usage, ok := aiActionUsage(action); ok {
			fmt.Print(usage)
			return nil
		}
		return fmt.Errorf("unknown ai action: %s", action)
	}
	id := args[1]
	if !safeID(id) {
		return fmt.Errorf("invalid id: %s", id)
	}
	rest := args[2:]

	switch action {
	case "new":
		opts, err := parseAIWorkflowOptions(rest)
		if err != nil {
			return err
		}
		return aiNewTask(id, opts)
	case "decision":
		opts, err := parseAIWorkflowOptions(rest)
		if err != nil {
			return err
		}
		return aiCreateDecision(id, opts)
	case "spec":
		opts, err := parseAIWorkflowOptions(rest)
		if err != nil {
			return err
		}
		return aiCreateSpec(id, opts)
	case "archive":
		aopts, aiopts, err := parseAIArchiveOptions(rest)
		if err != nil {
			return err
		}
		return aiArchiveTask(id, aopts, aiopts)
	default:
		return fmt.Errorf("unknown ai action: %s", action)
	}
}

func isHelpFlag(s string) bool {
	return s == "-h" || s == "--help" || s == "help"
}

func aiWorkflowUsage() string {
	return "usage: ai <action> <id> [--session <ref>] [--last] [--prompt <text>] [--apply] [--dry-run]\n" +
		"actions:\n" +
		"  new       create or draft openspec/changes/<id>/tasks.md\n" +
		"  decision  create or draft openspec/changes/<id>/design.md\n" +
		"  spec      create or draft openspec/specs/<id>/spec.md\n" +
		"  archive   draft archive-note and optionally run archive\n" +
		"run `aiw ai <action> -h` for action-specific help\n"
}

func aiActionUsage(action string) (string, bool) {
	switch action {
	case "new":
		return "usage: ai new <task-id> [--session <ref>] [--last] [--prompt <text>] [--apply] [--dry-run]\n" +
			"default behavior writes a draft to openspec/changes/<task-id>/.ai/drafts/tasks.md.ai.md\n" +
			"--apply writes directly to openspec/changes/<task-id>/tasks.md\n", true
	case "decision":
		return "usage: ai decision <task-id> [--session <ref>] [--last] [--prompt <text>] [--apply] [--dry-run]\n" +
			"task must already exist\n" +
			"default behavior writes a draft to openspec/changes/<task-id>/.ai/drafts/design.md.ai.md\n", true
	case "spec":
		return "usage: ai spec <spec-id> [--session <ref>] [--last] [--prompt <text>] [--apply] [--dry-run]\n" +
			"default behavior writes a draft to openspec/specs/<spec-id>/.ai/drafts/spec.md.ai.md\n" +
			"--apply writes directly to openspec/specs/<spec-id>/spec.md\n", true
	case "archive":
		return "usage: ai archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--finalize] [--session <ref>] [--last] [--prompt <text>] [--apply] [--dry-run]\n" +
			"default behavior writes a draft archive-note and does not archive\n" +
			"--apply writes archive-note.md then executes archive with provided archive flags\n", true
	default:
		return "", false
	}
}

func parseAIWorkflowOptions(args []string) (AIWorkflowOptions, error) {
	var opts AIWorkflowOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			if i+1 >= len(args) {
				return AIWorkflowOptions{}, fmt.Errorf("missing value for --session")
			}
			i++
			opts.Session = args[i]
		case "--last":
			opts.Last = true
		case "--apply":
			opts.Apply = true
		case "--dry-run":
			opts.DryRun = true
		case "--prompt":
			if i+1 >= len(args) {
				return AIWorkflowOptions{}, fmt.Errorf("missing value for --prompt")
			}
			i++
			opts.Prompt = args[i]
		default:
			return AIWorkflowOptions{}, fmt.Errorf("unknown ai option: %s", args[i])
		}
	}
	if opts.Session != "" && opts.Last {
		return AIWorkflowOptions{}, fmt.Errorf("--session and --last cannot be used together")
	}
	if opts.Apply && opts.DryRun {
		return AIWorkflowOptions{}, fmt.Errorf("--apply and --dry-run cannot be used together")
	}
	return opts, nil
}

func parseAIArchiveOptions(args []string) (ArchiveOptions, AIWorkflowOptions, error) {
	var archiveFlags []string
	var aiFlags []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--push", "--cleanup-wt", "--delete-branch", "--finalize":
			archiveFlags = append(archiveFlags, a)
		case "--session", "--prompt":
			if i+1 >= len(args) {
				return ArchiveOptions{}, AIWorkflowOptions{}, fmt.Errorf("missing value for %s", a)
			}
			aiFlags = append(aiFlags, a, args[i+1])
			i++
		case "--last", "--apply", "--dry-run":
			aiFlags = append(aiFlags, a)
		default:
			return ArchiveOptions{}, AIWorkflowOptions{}, fmt.Errorf("unknown archive ai option: %s", a)
		}
	}
	aopts, err := parseArchiveOptions(archiveFlags)
	if err != nil {
		return ArchiveOptions{}, AIWorkflowOptions{}, err
	}
	aiopts, err := parseAIWorkflowOptions(aiFlags)
	if err != nil {
		return ArchiveOptions{}, AIWorkflowOptions{}, err
	}
	return aopts, aiopts, nil
}

func aiNewTask(id string, opts AIWorkflowOptions) error {
	target := filepath.Join(taskx.TaskDir(id), "tasks.md")
	if opts.DryRun {
		fmt.Println("dry-run: no files will be created or modified")
	} else {
		if err := newTask(id); err != nil {
			return err
		}
	}
	current := readFileOrDefault(target, defaultTaskMarkdown())
	prompt := fmt.Sprintf("You are writing %s for task id %s.\\nUse practical and concise sections: Goal, Scope, Constraints, Context, TODO, Verification, Notes.\\nPreserve markdown headings.\\n\\nCurrent file:\\n%s", filepath.ToSlash(target), id, current)
	if strings.TrimSpace(opts.Prompt) != "" {
		prompt += "\\n\\nAdditional user intent:\\n" + opts.Prompt
	}
	draft, err := runAICxsExec(prompt, opts)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Println("dry-run complete: no file output generated")
		return nil
	}
	return writeAIDraftOrApply(target, draft, opts.Apply)
}

func aiCreateDecision(id string, opts AIWorkflowOptions) error {
	taskDir := taskx.TaskDir(id)
	if _, err := os.Stat(taskDir); err != nil {
		return fmt.Errorf("task not found: %s", id)
	}
	target := filepath.Join(taskx.TaskDir(id), "design.md")
	if opts.DryRun {
		fmt.Println("dry-run: no files will be created or modified")
	} else {
		if err := createDecision(id); err != nil {
			return err
		}
	}
	current := readFileOrDefault(target, defaultDecisionMarkdown(id))
	prompt := fmt.Sprintf("You are writing %s for task id %s.\\nOutput markdown with sections: Decision, Why, Alternatives, Risks, Rollback, Future Notes.\\n\\nCurrent file:\\n%s", filepath.ToSlash(target), id, current)
	if strings.TrimSpace(opts.Prompt) != "" {
		prompt += "\\n\\nAdditional user intent:\\n" + opts.Prompt
	}
	draft, err := runAICxsExec(prompt, opts)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Println("dry-run complete: no file output generated")
		return nil
	}
	return writeAIDraftOrApply(target, draft, opts.Apply)
}

func aiCreateSpec(id string, opts AIWorkflowOptions) error {
	target := filepath.Join(taskx.SpecsDir, id, "spec.md")
	if opts.DryRun {
		fmt.Println("dry-run: no files will be created or modified")
	} else {
		if err := createSpec(id); err != nil {
			return err
		}
	}
	current := readFileOrDefault(target, defaultSpecMarkdown(id))
	prompt := fmt.Sprintf("You are writing %s for spec id %s.\\nOutput markdown with sections: Purpose, Invariants, APIs, Notes, Open Questions(if needed).\\n\\nCurrent file:\\n%s", filepath.ToSlash(target), id, current)
	if strings.TrimSpace(opts.Prompt) != "" {
		prompt += "\\n\\nAdditional user intent:\\n" + opts.Prompt
	}
	draft, err := runAICxsExec(prompt, opts)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Println("dry-run complete: no file output generated")
		return nil
	}
	return writeAIDraftOrApply(target, draft, opts.Apply)
}

func aiArchiveTask(id string, aopts ArchiveOptions, opts AIWorkflowOptions) error {
	src := taskx.TaskDir(id)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("task not found: %s", id)
	}
	taskPath := filepath.Join(src, "tasks.md")
	designPath := filepath.Join(src, "design.md")
	taskBody, _ := os.ReadFile(taskPath)
	designBody, _ := os.ReadFile(designPath)
	prompt := fmt.Sprintf("Generate archive-note.md for task %s with sections: Summary, What Changed, Validation, Risks Remaining, Follow-ups.\\n\\nTask:\\n%s\\n\\nDesign:\\n%s", id, string(taskBody), string(designBody))
	if strings.TrimSpace(opts.Prompt) != "" {
		prompt += "\\n\\nAdditional user intent:\\n" + opts.Prompt
	}
	draft, err := runAICxsExec(prompt, opts)
	if err != nil {
		return err
	}
	noteTarget := filepath.Join(src, "archive-note.md")
	if opts.DryRun {
		fmt.Println("dry-run: no files will be created or modified")
		fmt.Println("dry-run complete: no file output generated")
		return nil
	}
	if err := writeAIDraftOrApply(noteTarget, draft, opts.Apply); err != nil {
		return err
	}
	if !opts.Apply {
		fmt.Println("ai archive preview generated. Re-run with --apply to archive task.")
		return nil
	}
	return archiveTask(id, aopts)
}

func runAICxsExec(prompt string, opts AIWorkflowOptions) (string, error) {
	path, err := plug.DiscoverPlugin("cxs")
	if err != nil {
		return "", fmt.Errorf("discover cxs plugin: %w", err)
	}

	args := []string{"exec"}
	var tmpPath string
	if opts.DryRun {
		args = append(args, "--output-last-message", dryRunOutputPath())
	} else {
		tmp, err := os.CreateTemp("", "aiw-cxs-output-*.md")
		if err != nil {
			return "", err
		}
		tmpPath = tmp.Name()
		_ = tmp.Close()
		defer os.Remove(tmpPath)
		args = append(args, "--output-last-message", tmpPath)
	}
	if opts.Session != "" {
		args = append(args, "--session", opts.Session)
	}
	if opts.Last {
		args = append(args, "--last")
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, prompt)

	code, err := plug.ExecPlugin(path, args, map[string]string{})
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("cxs plugin exited with code %d", code)
	}
	if opts.DryRun {
		return "", nil
	}
	b, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(b)) == "" {
		return "", fmt.Errorf("ai output is empty")
	}
	return string(b), nil
}

func readFileOrDefault(path, fallback string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	if strings.TrimSpace(string(b)) == "" {
		return fallback
	}
	return string(b)
}

func dryRunOutputPath() string {
	if runtime.GOOS == "windows" {
		return "NUL"
	}
	return "/dev/null"
}

func defaultTaskMarkdown() string {
	return `# Goal
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
}

func defaultDecisionMarkdown(id string) string {
	return fmt.Sprintf(`# %s Design
## Decision
...
## Why
...
## Risks
...
## Future Notes
...
`, id)
}

func defaultSpecMarkdown(id string) string {
	return fmt.Sprintf(`# %s Spec
## Purpose
...
## Invariants
-
## APIs
-
## Notes
...
`, strings.Title(id))
}

func writeAIDraftOrApply(targetPath, body string, apply bool) error {
	if strings.TrimSpace(body) == "" {
		fmt.Println("ai draft is empty; no file updated")
		return nil
	}
	if apply {
		if err := os.WriteFile(targetPath, []byte(ensureTrailingNewline(body)), 0o644); err != nil {
			return err
		}
		fmt.Println("ai applied:", filepath.ToSlash(targetPath))
		return nil
	}
	draftDir := filepath.Join(filepath.Dir(targetPath), ".ai", "drafts")
	if err := os.MkdirAll(draftDir, 0o755); err != nil {
		return err
	}
	draftPath := filepath.Join(draftDir, filepath.Base(targetPath)+".ai.md")
	if err := os.WriteFile(draftPath, []byte(ensureTrailingNewline(body)), 0o644); err != nil {
		return err
	}
	fmt.Println("ai draft:", filepath.ToSlash(draftPath))
	fmt.Println("run with --apply to write target file")
	return nil
}
