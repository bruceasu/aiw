package initcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
)

const (
	agentsFile        = "AGENTS.md"
	codexFile         = "CODEX.md"
	copilotFile       = ".github/copilot-instructions.md"
	agentTemplatesDir = "agent-templates"
	usageText         = "usage: aiw init [--no-setup] [--prompts] [--merge] [--force] [--template <name>]\n  --no-setup skips creating base AGENTS.md and Copilot instructions.\n  With --merge, --force refreshes .agents/prompts while instruction files are merged.\n"
)

type InitOptions struct {
	Prompts     PromptOptions
	WithPrompts bool
	SkipSetup   bool
}

func Dispatch(args []string) error {
	opts, showHelp, err := parseInitOptions(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Print(usageText)
		return nil
	}
	return initWorkspace(opts)
}

func initWorkspace(opts InitOptions) error {
	if !opts.SkipSetup {
		if err := setupBaseInstructions(); err != nil {
			return err
		}
	}
	if opts.WithPrompts {
		if err := syncPrompts(opts.Prompts); err != nil {
			return err
		}
	}
	if opts.SkipSetup && !opts.WithPrompts {
		fmt.Println("init result: no files changed")
	}
	return nil
}

func setupBaseInstructions() error {
	templatesRoot, err := resolvePromptTemplatesDir()
	if err != nil {
		return err
	}
	agents, err := readOptionalTemplate(filepath.Join(templatesRoot, agentsFile))
	if err != nil {
		return err
	}
	copilot, err := readOptionalTemplate(filepath.Join(templatesRoot, "dot.github", filepath.Base(copilotFile)))
	if err != nil {
		return err
	}
	if strings.TrimSpace(agents) == "" {
		return fmt.Errorf("no base AGENTS template found under %s", templatesRoot)
	}
	if err := os.MkdirAll(filepath.Dir(copilotFile), 0o755); err != nil {
		return err
	}
	if err := writeIfMissing(agentsFile, agents); err != nil {
		return err
	}
	if err := writeIfMissing(copilotFile, copilot); err != nil {
		return err
	}
	return nil
}

func writeIfMissing(path, content string) error {
	if fsx.Exists(path) || strings.TrimSpace(content) == "" {
		return nil
	}
	return os.WriteFile(path, []byte(ensureTrailingNewline(content)), 0o644)
}

func parseInitOptions(args []string) (InitOptions, bool, error) {
	opts := InitOptions{}
	for index := 0; index < len(args); index++ {
		switch arg := args[index]; arg {
		case "--prompts":
			opts.WithPrompts = true
		case "--no-setup":
			opts.SkipSetup = true
		case "--merge":
			opts.Prompts.Merge = true
		case "--force":
			opts.Prompts.Force = true
		case "--template":
			if index+1 >= len(args) || args[index+1] == "" || strings.HasPrefix(args[index+1], "--") {
				return InitOptions{}, false, errors.New("missing value for --template")
			}
			index++
			opts.Prompts.Template = args[index]
		case "--help", "-h":
			return InitOptions{}, true, nil
		default:
			return InitOptions{}, false, fmt.Errorf("unknown init option: %s", arg)
		}
	}
	if (opts.Prompts.Merge || opts.Prompts.Force || opts.Prompts.Template != "") && !opts.WithPrompts {
		return InitOptions{}, false, errors.New("--merge, --force, and --template require --prompts")
	}
	return opts, false, nil
}

func ensureTrailingNewline(content string) string {
	if strings.HasSuffix(content, "\n") {
		return content
	}
	return content + "\n"
}
