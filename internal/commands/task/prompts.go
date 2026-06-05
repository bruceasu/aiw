package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aiw/internal/fsx"
)

type PromptOptions struct {
	Template string
	Merge    bool
	Force    bool
	List     bool
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

	if err := os.MkdirAll(filepath.Dir(copilotFile), 0o755); err != nil {
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

func detectPromptTemplate() (string, error) {
	if fsx.Exists("go.mod") {
		return "go", nil
	}
	if fsx.Exists("pom.xml") || fsx.Exists("build.gradle") || fsx.Exists("build.gradle.kts") {
		return "java", nil
	}
	if fsx.Exists("pyproject.toml") || fsx.Exists("requirements.txt") || fsx.Exists("setup.py") {
		return "python", nil
	}
	return "", errors.New("cannot auto-detect prompts template; pass one explicitly, e.g.: aiw prompts go --merge")
}

func readOptionalTemplate(paths ...string) (string, error) {
	for _, path := range paths {
		if !fsx.Exists(path) {
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
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return "", err
		}
		return "wrote", nil
	}
	if !fsx.Exists(path) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
	if err := os.WriteFile(path, []byte(merged), 0o644); err != nil {
		return "", err
	}
	return "merged", nil
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
