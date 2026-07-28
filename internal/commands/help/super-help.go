package help

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/fsx"
	plug "aiw/internal/plugin"
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
		if ok, _ := pluginExists(name); ok {
			return showPluginHelp(name)
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
	for _, name := range plugins {
		doc.Plugins = append(doc.Plugins, helpEntry{Name: name, Short: getPluginShort(name), Description: getPluginShort(name), Source: "plugin"})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

func listAll() error {
	fmt.Print("aiw - workflow-first task and AI tooling\n\n" +
		"Usage:\n" +
		"  aiw <command> [args...]\n" +
		"  aiw --help\n" +
		"  aiw help <command>\n\n")

	fmt.Print("Core workflow:\n" +
		"  init [--prompts] [--merge] [--force] [--template <name>]\n" +
		"  new <task-id>             Create a task/change.\n" +
		"  list                      List tasks under openspec/changes.\n" +
		"  show <task-id>            Print tasks.md.\n" +
		"  status <task-id> <s>      Update task status.\n" +
		"  done <task-id>            Shortcut for: status <task-id> DONE.\n" +
		"  archive <task-id> [opts]  Archive a completed task/change.\n" +
		"  context <task-id>         Show files to read before implementing.\n" +
		"  decision <task-id>        Create design.md for a task.\n" +
		"  spec <spec-id>            Create a long-lived spec under openspec/specs.\n" +
		"  registry                  Rebuild openspec/registry.json.\n\n")

	fmt.Print("Worktree:\n" +
		"  wt add <task-id> [base]   Create or attach a task worktree.\n" +
		"  wt rm <task-id>           Remove a task worktree.\n" +
		"  wt list                   List known worktrees.\n" +
		"  wt prune [--dry-run]      Remove stale worktree metadata.\n" +
		"  wt lock <task-id> [r]     Protect a worktree from removal.\n" +
		"  wt unlock <task-id>       Remove a worktree lock.\n" +
		"  wt repair                 Repair worktree links after relocation.\n" +
		"  wt ignore                 Add .wt/ to .gitignore.\n\n")

	fmt.Print("Auxiliary:\n" +
		"  prompts <...>             Generate or merge prompt files.\n" +
		"  task agent <...>          Fresh-agent handoff and lineage tools.\n" +
		"  cxs <...>                 Inspect and resume Codex CLI sessions.\n" +
		"  flow <...>                Automate repeatable AI processing flows.\n\n")

	fmt.Print("Plugins:\n" +
		"  git <subcommand>         Git helpers and discoverable Git subcommands.\n" +
		"  github <subcommand>      Read and publish GitHub Issues and PRs.\n" +
		"  cz <subcommand>          Conventional Commit wizard.\n" +
		"  tcc <args...>            Tiny C Compiler wrapper.\n\n")

	fmt.Print("Examples:\n" +
		"  aiw init --prompts --template go\n" +
		"  aiw new payment-retry\n" +
		"  aiw cxs exec --last \"continue latest session\"\n\n")

	pls, err := listPlugins()
	if err != nil || len(pls) == 0 {
		fmt.Println("Plugins: none discovered beside this aiw binary.")
		fmt.Println("Place executable plugins next to aiw, then run: aiw <plugin> --help")
		return nil
	}

	fmt.Println("Plugins:")
	for _, p := range pls {
		desc := getPluginShort(p)
		if desc == "" {
			fmt.Printf("  %s\n", p)
			continue
		}
		fmt.Printf("  %s - %s\n", p, desc)
	}

	return nil
}

// getPluginShort attempts to read the plugin source and extract META['short'].
// If not found, returns empty string.
func getPluginShort(name string) string {
	path := pluginScriptPath(name)
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return extractShortFromSource(string(b))
}

// extractShortFromSource looks for patterns like 'short': '...' or "short": "...".
func extractShortFromSource(src string) string {
	// look for "short"
	idx := strings.Index(src, "short")
	if idx == -1 {
		return ""
	}
	// search from idx to next newline for colon
	tail := src[idx:]
	// find first colon
	cidx := strings.Index(tail, ":")
	if cidx == -1 {
		return ""
	}
	// rest after colon
	rest := tail[cidx+1:]
	// find first quote (single or double)
	rest = strings.TrimSpace(rest)
	if len(rest) == 0 {
		return ""
	}
	var quote byte
	if rest[0] == '\'' || rest[0] == '"' {
		quote = rest[0]
	} else {
		// not quoted; return until comma or newline
		end := strings.IndexAny(rest, ",\n")
		if end == -1 {
			return strings.TrimSpace(rest)
		}
		return strings.TrimSpace(rest[:end])
	}
	// find closing quote
	rest = rest[1:]
	end := strings.IndexByte(rest, quote)
	if end == -1 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

func listBuiltins() ([]string, error) {
	// Static list embedded in code 閳?used when source tree is not present
	builtinCommands := staticBuiltinCommands()

	out := []string{}
	seen := map[string]bool{}
	for _, b := range builtinCommands {
		out = append(out, b)
		seen[b] = true
	}

	// Try to merge with actual internal/commands directory if available
	cmdsDir := filepath.Join("internal", "commands")
	if entries, err := os.ReadDir(cmdsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if name == "help" || name == "cz" {
				continue
			}
			if !seen[name] {
				out = append(out, name)
				seen[name] = true
			}
		}
	}
	return out, nil
}

func staticBuiltinCommands() []string {
	return []string{
		"init", "new", "list", "show", "status", "done",
		"archive", "context", "decision", "spec", "registry",
		"prompts", "wt", "task",
	}
}

func listPlugins() ([]string, error) {
	pluginsDir, err := resolvePluginsDir()
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, err
	}
	out := []string{}
	seen := map[string]bool{}
	for _, f := range files {
		if f.IsDir() {
			subEntries, err := os.ReadDir(filepath.Join(pluginsDir, f.Name()))
			if err != nil {
				continue
			}
			for _, sub := range subEntries {
				name, ok := pluginNameFromFile(sub.Name())
				if ok && !seen[name] {
					out = append(out, name)
					seen[name] = true
				}
			}
			continue
		}
		if name, ok := pluginNameFromFile(f.Name()); ok && !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
	}
	return out, nil
}

func pluginExists(name string) (bool, string) {
	if p := pluginScriptPath(name); p != "" {
		return true, p
	}
	return false, ""
}

func builtinExists(name string) bool {
	path := filepath.Join("internal", "commands", name)
	if fsx.Exists(path) {
		return true
	}
	for _, b := range staticBuiltinCommands() {
		if b == name {
			return true
		}
	}
	return false
}

func showPluginHelp(name string) error {
	path, err := plug.DiscoverPlugin(name)
	if err != nil {
		return errors.New("plugin not found")
	}
	code, err := plug.ExecPlugin(path, []string{"-h"}, map[string]string{
		"AIW_PLUGIN_NAME": name,
		"AIW_PLUGIN_PATH": path,
		"AIW_CMDLINE":     "help " + name,
	})
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
		return "initialize prompts and worktree scaffolding"
	case "new":
		return "create a new change"
	case "list":
		return "list changes"
	case "show":
		return "show change tasks"
	case "status":
		return "update change status"
	case "done":
		return "mark change done"
	case "archive":
		return "archive a completed change"
	case "context":
		return "show implementation context"
	case "decision":
		return "create or show a design decision"
	case "spec":
		return "create a long-lived spec"
	case "registry":
		return "rebuild the registry"
	case "prompts":
		return "generate or merge prompt files"
	case "task":
		return "manage fresh-agent handoff and lineage"
	default:
		return ""
	}
}

func builtinUsageText(name string) (string, bool) {
	switch name {
	case "init":
		return "usage: aiw init [--prompts] [--merge] [--force] [--template <name>]\n", true
	case "new":
		return "usage: aiw new <task-id>\n", true
	case "list":
		return "usage: aiw list\n", true
	case "show":
		return "usage: aiw show <task-id>\n", true
	case "status":
		return "usage: aiw status <task-id> <status>\n", true
	case "done":
		return "usage: aiw done <task-id>\n", true
	case "archive":
		return "usage: aiw archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--finalize]\n", true
	case "context":
		return "usage: aiw context <task-id>\n", true
	case "decision":
		return "usage: aiw decision <task-id>\n", true
	case "spec":
		return "usage: aiw spec <spec-id>\n", true
	case "registry":
		return "usage: aiw registry\n", true
	case "prompts":
		return "usage: aiw prompts [list|<template>] [--merge] [--force]\n", true
	case "task":
		return "usage: aiw task agent <next|status> <task-id>\n", true
	default:
		return "", false
	}
}
func searchAndAnswer(query string) error {
	fmt.Fprintf(os.Stderr, "Searching docs for: %s\n", query)
	matches := searchDocs(query)
	if len(matches) == 0 {
		fmt.Println("no matching docs found")
		return nil
	}

	// try LLM if configured
	if url := os.Getenv("AIW_LLM_URL"); url != "" {
		if ans, err := askLLM(url, query, matches); err == nil && ans != "" {
			fmt.Println(ans)
			return nil
		}
	}

	// fallback: print search hits
	for i, m := range matches {
		fmt.Printf("--- result %d ---\n", i+1)
		fmt.Println(m)
	}
	return nil
}

func searchDocs(query string) []string {
	out := []string{}
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

	// search plugin META (quick scan)
	pls, _ := listPlugins()
	for _, p := range pls {
		path := pluginScriptPath(p)
		if path == "" {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		s := strings.ToLower(string(b))
		if strings.Contains(s, strings.ToLower(query)) {
			snippet := excerptText(string(b), query, 300)
			out = append(out, fmt.Sprintf("plugin %s:\n%s", p, snippet))
		}
	}
	return out
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

// askLLM posts a JSON payload to configured URL and expects a text response.
// Payload: {"query": "...", "docs": ["...",...]}
func askLLM(url, query string, docs []string) (string, error) {
	payload := map[string]any{"query": query, "docs": docs}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// allow API key via AIW_LLM_KEY
	if k := os.Getenv("AIW_LLM_KEY"); k != "" {
		req.Header.Set("Authorization", "Bearer "+k)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llm error: %d %s", resp.StatusCode, string(b))
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func pluginNameFromFile(filename string) (string, bool) {
	if strings.HasPrefix(filename, "aiw-") && strings.HasSuffix(filename, ".py") {
		return strings.TrimSuffix(strings.TrimPrefix(filename, "aiw-"), ".py"), true
	}
	return "", false
}

func pluginScriptPath(name string) string {
	pluginsDir, err := resolvePluginsDir()
	if err != nil {
		return ""
	}
	candidates := []string{
		filepath.Join(pluginsDir, fmt.Sprintf("aiw-%s.py", name)),
		filepath.Join(pluginsDir, fmt.Sprintf("aiw-%s", name), fmt.Sprintf("aiw-%s.py", name)),
	}
	for _, candidate := range candidates {
		if fsx.Exists(candidate) {
			return candidate
		}
	}
	return ""
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
