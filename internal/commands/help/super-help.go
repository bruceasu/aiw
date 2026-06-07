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

// Dispatch implements a flexible help command:
//   - no args: list builtins and plugins
//   - help <name>: show help for built-in or plugin
//   - help <free text>: search docs and plugin META/help, optionally ask an LLM
func Dispatch(args []string) error {
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

func listAll() error {
	fmt.Print(`aiw — Private workspace CLI

Task management:
  init [--prompts] [--merge] [--force] [--template <name>]
  new <task-id>             Create task folder (task.toml / task.md / notes.md).
  list                      List tasks from openspec/changes.
  show <task-id>            Print task.md.
  status <task-id> <s>      Update task status (auto upper-cased).
  done <task-id>            Shortcut for: status <task-id> DONE.
  archive <task-id> [opts]  Move task to openspec/archive; supports --push / --cleanup-wt.
  context <task-id>         Show files to read before implementing.
  decision <task-id>        Create design.md when design is needed.
  spec <spec-id>            Create long-lived spec under openspec/specs.
  registry                  Rebuild openspec/registry.json.
  prompts [template] [opts] Create or merge AGENTS/CODEX/Copilot prompts.
`)
	fmt.Print(`Examples:
  aiw init --prompts --template go
  aiw new payment-retry
`)

	// Print builtins
	fmt.Println("\nBuilt-in commands:")
	bins, _ := listBuiltins()
	for _, b := range bins {
		fmt.Printf("  %s\n", b)
	}

	// Print plugins with short descriptions (if available)
	fmt.Println("\nPlugins:")
	pls, _ := listPlugins()
	for _, p := range pls {
		desc := getPluginShort(p)
		if desc == "" {
			fmt.Printf("  %s\n", p)
		} else {
			fmt.Printf("  %s - %s\n", p, desc)
		}
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
	// Static list embedded in code — used when source tree is not present
	builtinCommands := []string{
		"init", "new", "list", "show", "status", "done",
		"archive", "context", "decision", "spec", "registry",
		"prompts", "wt", "git", "tcc", "task",
	}

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
	return fsx.Exists(path)
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
	// attempt to execute the current binary with <name> -h to get help output
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate executable: %w", err)
	}
	cmd := exec.Command(exe, name, "-h")
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
