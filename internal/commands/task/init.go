package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
	"aiw/internal/taskx"
)

type InitOptions struct {
	Prompts     PromptOptions
	WithPrompts bool
}

func initWorkspace(opts InitOptions) error {
	dirs := []string{
		taskx.OpenspecDir,
		taskx.ChangesDir,
		taskx.SpecsDir,
		taskx.ArchiveDir,
		taskx.WorktreeDir,
		filepath.Dir(copilotFile),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	if err := writeIfMissing(agentsFile, agentsTemplate()); err != nil {
		return err
	}
	if err := writeIfMissing(copilotFile, copilotTemplate()); err != nil {
		return err
	}
	if err := taskx.EnsureWorktreeIgnored(); err != nil {
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
	if fsx.Exists(path) {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0o644)
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
