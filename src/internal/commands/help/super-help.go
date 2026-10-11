package help

import (
	"aiw/internal/ai"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	plug "aiw/internal/plugin"
	"aiw/internal/version"
)

var executablePathFn = os.Executable
var execCommandFn = exec.Command

type helpJSON struct {
	Command  string      `json:"command"`
	Builtins []helpEntry `json:"builtins"`
	Plugins  []helpEntry `json:"plugins"`
}

type helpEntry struct {
	Name        string `json:"name"`
	Short       string `json:"short,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
}

// Dispatch implements a flexible help command:
//   - no args: list builtins and plugins
//   - help <name>: show help for built-in or plugin
//   - help <free text>: search docs and plugin META/help, optionally ask an LLM
func Dispatch(args []string) error {
	if len(args) > 0 && args[0] == "--json" {
		return listAllJSON()
	}
	if len(args) == 0 {
		return listAll()
	}

	// join args as one query if more than one
	if len(args) == 1 {
		name := args[0]
		// check plugin first
		plugin, err := plug.DiscoverPluginInfo(name)
		if err == nil {
			return showPluginHelp(plugin)
		}
		if !errors.Is(err, plug.ErrPluginNotFound) {
			return err
		}
		// check builtin
		if ok := builtinExists(name); ok {
			return showBuiltinHelp(name)
		}
		// fallback: treat as free-text query
		return searchAndAnswer(strings.Join(args, " "))
	}

	// multi-word query
	return searchAndAnswer(strings.Join(args, " "))
}

func listAllJSON() error {
	builtins, err := listBuiltins()
	if err != nil {
		return err
	}
	plugins, err := listPlugins()
	if err != nil {
		return err
	}
	doc := helpJSON{Command: "help"}
	for _, name := range builtins {
		doc.Builtins = append(doc.Builtins, helpEntry{Name: name, Short: builtinHelpShort(name), Description: builtinHelpShort(name), Source: "builtin"})
	}
	for _, plugin := range plugins {
		description := plugin.Description
		if description == "" {
			description = getPluginShort(plugin.Path)
		}
		doc.Plugins = append(doc.Plugins, helpEntry{Name: plugin.Name, Short: description, Description: description, Source: "plugin"})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

func listAll() error {
	fmt.Print("aiw " + version.Label() + " - FD-first workflow and AI tooling\n\n" +
		"Usage:\n" +
		"  aiw <command> [args...]\n" +
		"  aiw --help\n" +
		"  aiw help <command>\n\n")

	fmt.Print("Core workflow:\n" +
		"  init [--no-setup] [--prompts] [--merge] [--force] [--template <name>]\n" +
		"  fd new <title> [--issue <id>]  Create a numbered FD and request Planner.\n" +
		"  fd list                      Show the FD index.\n" +
		"  fd show <fd-id>               Show an FD and its last handoff.\n" +
		"  fd emit <fd-id> <event> --producer <role> --artifact <path> [--source-event <id>]\n" +
		"                               Record a stage result and route the next role.\n" +
		"                               Test events: test-report-ready, test-accepted, test-rejected.\n" +
		"  fd claim <fd-id> <event-id> --session <id>\n" +
		"                               Bind a pending handoff to one host session.\n" +
		"  fd resume <fd-id>             Resume a pending handoff safely.\n" +
		"  fd request-review <fd-id> --reason <text>\n" +
		"                               Request review for a pending or completed FD.\n" +
		"  fd refresh-worker <fd-id> --reason <text>\n" +
		"                               Replace a stale pending Worker handoff.\n" +
		"  fd refresh-tester <fd-id> --reason <text> --artifact <report>\n" +
		"                               Replace a stale unclaimed Tester handoff.\n" +
		"  fd reopen <fd-id> --reason <text>\n" +
		"                               Resume an archived Closed or Deferred FD.\n" +
		"  fd reopen <fd-id> --reason <text> --correct-reason\n" +
		"                               Correct an unclaimed reopen handoff reason.\n" +
		"  fd close <fd-id> <Complete|Deferred|Closed> [--reason <text>]\n" +
		"                               Archive an FD with the required evidence.\n" +
		"  issue <...>                  Manage Issue intake, split lineage, and promotion.\n" +
		"  req <...>                    Compatibility alias for Issue records.\n\n")

	fmt.Print("Other tools:\n" +
		"  ask <prompt>                 Ask the built-in LLM for AIW guidance.\n" +
		"  completion <shell>           Generate shell completion scripts.\n" +
		"  version                      Print the AIW version.\n\n")

	fmt.Print("Examples:\n" +
		"  aiw init --prompts --template go\n" +
		"  aiw fd new \"Payment retry\"\n" +
		"  aiw fd list\n" +
		"  aiw fd show FD-001\n" +
		"  aiw help fd\n\n")

	pls, err := listPlugins()
	if err != nil {
		return fmt.Errorf("list plugins: %w", err)
	}
	if len(pls) == 0 {
		fmt.Println("Plugins: none discovered beside this aiw binary.")
		fmt.Println("Place executable plugins next to aiw, then run: aiw <plugin> --help")
		return nil
	}

	fmt.Println("Plugins:")
	for _, plugin := range pls {
		desc := plugin.Description
		if desc == "" {
			desc = getPluginShort(plugin.Path)
		}
		if desc == "" {
			fmt.Printf("  %s\n", plugin.Name)
			continue
		}
		fmt.Printf("  %s - %s\n", plugin.Name, desc)
	}

	return nil
}

func listBuiltins() ([]string, error) {
	return staticBuiltinCommands(), nil
}

func getPluginShort(path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	source := string(contents)
	idx := strings.Index(source, "short")
	if idx < 0 {
		return ""
	}
	tail := source[idx:]
	colon := strings.Index(tail, ":")
	if colon < 0 {
		return ""
	}
	value := strings.TrimSpace(tail[colon+1:])
	if value == "" {
		return ""
	}
	if value[0] != '\'' && value[0] != '"' {
		if end := strings.IndexAny(value, ",\n"); end >= 0 {
			return strings.TrimSpace(value[:end])
		}
		return value
	}
	end := strings.IndexByte(value[1:], value[0])
	if end < 0 {
		return strings.TrimSpace(value[1:])
	}
	return strings.TrimSpace(value[1 : end+1])
}

func staticBuiltinCommands() []string {
	return []string{"init", "ask", "completion", "help", "version", "issue"}
}

func listPlugins() ([]plug.PluginInfo, error) {
	pluginsDir, err := resolvePluginsDir()
	if err != nil {
		return nil, err
	}
	plugins, err := plug.ListPluginsIn([]string{pluginsDir})
	if err != nil {
		return nil, err
	}
	out := make([]plug.PluginInfo, 0, len(plugins))
	for _, plugin := range plugins {
		if plugin.Name != "wf" {
			out = append(out, plugin)
		}
	}
	return out, nil
}

func builtinExists(name string) bool {
	for _, b := range staticBuiltinCommands() {
		if b == name {
			return true
		}
	}
	return false
}

func showPluginHelp(plugin plug.PluginInfo) error {
	if strings.TrimSpace(plugin.Help) != "" {
		fmt.Println(strings.TrimRight(plugin.Help, "\n"))
		return nil
	}
	code, err := plug.ExecPluginWithStartup(plugin.Path, plugin.Startup, []string{"-h"}, plug.InvocationEnvironment(plugin.Name, plugin.Path, "help "+plugin.Name))
	if err != nil {
		return fmt.Errorf("running plugin help: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("plugin help exited with code %d", code)
	}
	return nil
}

func showBuiltinHelp(name string) error {
	if usage, ok := builtinUsageText(name); ok {
		fmt.Print(usage)
		return nil
	}

	// attempt to execute the current binary with <name> -h to get help output
	exe, err := executablePathFn()
	if err != nil {
		return fmt.Errorf("cannot locate executable: %w", err)
	}
	cmd := execCommandFn(exe, name, "-h")
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		// if execution fails, fall back to simple message
		if errb.Len() > 0 {
			fmt.Fprintln(os.Stderr, errb.String())
		}
		fmt.Printf("Builtin command '%s' (no inline help available)\n", name)
		fmt.Printf("Run: %s %s -h to view help (executable run failed: %v)\n", exe, name, err)
		return nil
	}
	fmt.Print(outb.String())
	if errb.Len() > 0 {
		fmt.Fprintln(os.Stderr, errb.String())
	}
	return nil
}

func builtinHelpShort(name string) string {
	switch name {
	case "init":
		return "initialize project instructions and prompt templates"
	case "ask":
		return "ask the built-in LLM for AIW guidance"
	case "completion":
		return "generate shell completion scripts"
	case "help":
		return "show command help or search documentation"
	case "version":
		return "print the AIW version"
	case "issue":
		return "manage Issue records through the req plugin"
	default:
		return ""
	}
}

func builtinUsageText(name string) (string, bool) {
	switch name {
	case "init":
		return "usage: aiw init [--no-setup] [--prompts] [--merge] [--force] [--template <name>]\n  --no-setup skips creating base AGENTS.md and Copilot instructions.\n  With --merge, --force refreshes .agents prompt files while instruction files are merged.\n", true
	case "help":
		return "usage: aiw help [--json|command|topic]\n", true
	case "version":
		return "usage: aiw version\n", true
	case "issue":
		return "usage: aiw issue <command> [args...]\nAlias for the req plugin; run aiw help req for subcommand help.\n", true
	default:
		return "", false
	}
}

func searchAndAnswer(query string) error {
	fmt.Fprintf(os.Stderr, "Searching docs for: %s\n", query)
	matches, err := searchDocs(query)
	if err != nil {
		return fmt.Errorf("search docs: %w", err)
	}
	if len(matches) == 0 {
		fmt.Println("no matching docs found")
		return nil
	}

	// Use the provider selected in the shared AIW configuration.
	if cfg, err := ai.LoadConfig(); err == nil && cfg.Name != "" {
		if provider, err := ai.NewProvider(cfg); err == nil {
			prompt := buildHelpPrompt(query, matches)
			if result, err := provider.Generate(context.Background(), ai.Request{Prompt: prompt, Model: cfg.Model}); err == nil && strings.TrimSpace(result.FinalOutput) != "" {
				fmt.Println(result.FinalOutput)
				return nil
			}
		}
	}

	// fallback: print search hits
	for i, m := range matches {
		fmt.Printf("--- result %d ---\n", i+1)
		fmt.Println(m)
	}
	return nil
}

func searchDocs(query string) ([]string, error) {
	out := []string{}
	// A manifest error must remain visible to free-text help searches too.
	pls, err := listPlugins()
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}

	// search docs/usage
	docsGlob := filepath.Join("docs", "usage", "*.md")
	files, _ := filepath.Glob(docsGlob)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		s := strings.ToLower(string(b))
		if strings.Contains(s, strings.ToLower(query)) {
			// include file heading and excerpt
			excerpt := excerptText(string(b), query, 800)
			out = append(out, fmt.Sprintf("%s:\n%s", filepath.Base(f), excerpt))
		}
	}

	// Search declared descriptions and help before scanning legacy source text.
	for _, plugin := range pls {
		metadata := strings.TrimSpace(strings.Join([]string{plugin.Description, plugin.Help}, "\n\n"))
		if strings.Contains(strings.ToLower(metadata), strings.ToLower(query)) {
			out = append(out, fmt.Sprintf("plugin %s:\n%s", plugin.Name, excerptText(metadata, query, 300)))
			continue
		}
		b, err := os.ReadFile(plugin.Path)
		if err == nil && strings.Contains(strings.ToLower(string(b)), strings.ToLower(query)) {
			out = append(out, fmt.Sprintf("plugin %s:\n%s", plugin.Name, excerptText(string(b), query, 300)))
		}
	}
	return out, nil
}

func excerptText(doc, query string, max int) string {
	low := strings.ToLower(doc)
	idx := strings.Index(low, strings.ToLower(query))
	if idx == -1 {
		if len(doc) <= max {
			return doc
		}
		return doc[:max]
	}
	start := idx - 120
	if start < 0 {
		start = 0
	}
	end := idx + 120
	if end > len(doc) {
		end = len(doc)
	}
	ex := doc[start:end]
	if len(ex) > max {
		ex = ex[:max]
	}
	return ex
}

func buildHelpPrompt(query string, docs []string) string {
	return fmt.Sprintf("Answer the user's AIW help question using only the following documentation. If the documentation is insufficient, say so clearly.\n\nQuestion:\n%s\n\nDocumentation:\n%s", query, strings.Join(docs, "\n\n---\n\n"))
}

func resolvePluginsDir() (string, error) {
	exePath, err := executablePathFn()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = resolvedPath
	}
	return filepath.Join(filepath.Dir(exePath), "plugins"), nil
}
