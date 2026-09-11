package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
	"aiw/internal/plugin"
	"aiw/internal/taskx"
)

const officialSetupPlugin = "setup-project"

var runOfficialSetupFn = runOfficialSetup

type InitOptions struct {
	Prompts     PromptOptions
	WithPrompts bool
	SkipSetup   bool
}

func initWorkspace(opts InitOptions) error {
	dirs := []string{
		taskx.OpenspecDir,
		taskx.ChangesDir,
		taskx.SpecsDir,
		taskx.ArchiveDir,
		taskx.RuntimeTasksPath(),
		taskx.WorktreeDir,
		filepath.Dir(copilotFile),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	baseAgentsCreated, err := writeIfMissing(agentsFile, agentsTemplate())
	if err != nil {
		return err
	}
	if _, err := writeIfMissing(copilotFile, copilotTemplate()); err != nil {
		return err
	}
	if err := taskx.EnsureWorktreeIgnored(); err != nil {
		return err
	}
	if opts.WithPrompts {
		if err := syncPrompts(opts.Prompts); err != nil {
			return err
		}
	}
	if !opts.SkipSetup {
		runOfficialSetupFn(baseAgentsCreated)
	}
	return nil
}

func runOfficialSetup(baseAgentsCreated bool) {
	path, err := plugin.DiscoverPlugin(officialSetupPlugin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "official setup plugin is unavailable; continuing with base initialization")
		return
	}
	args := []string{}
	if baseAgentsCreated {
		args = append(args, "--base-agents-created")
	}
	code, err := plugin.ExecPlugin(path, args, map[string]string{"AIW_SETUP_MODE": "init"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "official setup plugin could not start; continuing with base initialization")
		return
	}
	if code != 0 {
		fmt.Fprintln(os.Stderr, "official setup plugin failed; continuing with base initialization")
	}
}

func agentsTemplate() string {
	return `# AGENTS.md
This repository uses OpenSpec-lite TOML workflow.
Before coding:
- read .ai/tasks/<task>/task.toml if exists
- read openspec/changes/<task>/tasks.md if exists
- read design.md if exists
- read related specs under openspec/specs/
- archived changes live under openspec/changes/archive/

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
- openspec/changes/archive/
Keep changes scoped.
Avoid broad refactors.
`
}

func writeIfMissing(path, content string) (bool, error) {
	if fsx.Exists(path) {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func parseInitOptions(args []string) (InitOptions, error) {
	opts := InitOptions{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--prompts":
			opts.WithPrompts = true
		case "--no-setup":
			opts.SkipSetup = true
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
			return InitOptions{}, errors.New("help requested")
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

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
