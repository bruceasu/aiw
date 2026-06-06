package cz

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"aiw/internal/envx"
	"aiw/internal/fsx"
)

var dryRun bool

var issueRefRe = regexp.MustCompile(`#\d+`)

type czMessages struct {
	Type               string
	Scope              string
	CustomScope        string
	Subject            string
	Body               string
	Breaking           string
	FooterPrefixes     string
	CustomFooterPrefix string
	Footer             string
	ConfirmCommit      string
}

type czType struct {
	Value string
	Name  string
}

type czConfig struct {
	UseLLM     bool
	Candidates int
	Emoji      bool
	Editor     string
	LLMModel   string
	APIBaseURL string
	APIKey     string
	DebugSource bool
	Messages   czMessages
	Types      []czType
}

type czOptions struct {
	UseLLM     *bool
	Candidates *int
	Emoji      *bool
}

type czDraft struct {
	Type     string `json:"type"`
	Scope    string `json:"scope"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	Breaking string `json:"breaking"`
	Footer   string `json:"footer"`
}

type czLLMResponse struct {
	Candidates []czDraft `json:"candidates"`
}

type openAIChatRequest struct {
	Model          string               `json:"model"`
	Messages       []openAIChatMessage  `json:"messages"`
	ResponseFormat *openAIResponseShape `json:"response_format,omitempty"`
	Temperature    float64              `json:"temperature,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseShape struct {
	Type       string             `json:"type"`
	JSONSchema *openAIJSONSchema  `json:"json_schema,omitempty"`
}

type openAIJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type UI interface {
	DraftFromWizard(cfg czConfig) (czDraft, error)
	DraftFromLLM(cfg czConfig) (czDraft, error)
	ReviewAndCommit(draft czDraft, cfg czConfig, commitFn func(string) error) error
}

type lineUI struct{}

func Dispatch(args []string) error {
	return dispatchWithUI(args, lineUI{}, stagedChanges, commitWithMessage)
}

func dispatchWithUI(args []string, ui UI, stagedFn func() (string, error), commitFn func(string) error) error {
	opts, err := parseCzOptions(args)
	if err != nil {
		return err
	}

	cfg, err := loadCzConfig(opts)
	if err != nil {
		return err
	}
	if cfg.Candidates <= 0 {
		cfg.Candidates = 3
	}

	staged, err := stagedFn()
	if err != nil {
		return err
	}
	if strings.TrimSpace(staged) == "" {
		return errors.New("no staged changes; run git add first")
	}

	var draft czDraft
	if cfg.UseLLM {
		draft, err = ui.DraftFromLLM(cfg)
		if err != nil {
			return err
		}
	} else {
		draft, err = ui.DraftFromWizard(cfg)
		if err != nil {
			return err
		}
	}

	return ui.ReviewAndCommit(draft, cfg, commitFn)
}

func stagedChanges() (string, error) {
	return gitOutput("git", "diff", "--cached", "--name-only")
}

func (lineUI) DraftFromWizard(cfg czConfig) (czDraft, error) {
	return czDraftFromWizard(cfg)
}

func (lineUI) DraftFromLLM(cfg czConfig) (czDraft, error) {
	return czDraftFromLLM(cfg)
}

func (lineUI) ReviewAndCommit(draft czDraft, cfg czConfig, commitFn func(string) error) error {
	for {
		msg := buildCommitMessage(draft, cfg.Emoji)
		fmt.Println("\n--- Commit message preview ---")
		fmt.Println(msg)
		fmt.Println("--------------------------------")
		ans := strings.ToLower(strings.TrimSpace(promptLine(cfg.Messages.ConfirmCommit + " [y=commit/e=edit/n=cancel] ")))
		switch ans {
		case "y", "yes":
			return commitFn(msg)
		case "e", "edit", "m", "modify":
			draft = editDraft(draft, cfg)
		case "n", "no", "q", "quit", "":
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		default:
			fmt.Fprintln(os.Stderr, "invalid choice; please input y/e/n")
		}
	}
}

func parseCzOptions(args []string) (czOptions, error) {
	var opts czOptions
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--llm":
			v := true
			opts.UseLLM = &v
		case "--no-llm":
			v := false
			opts.UseLLM = &v
		case "--emoji":
			v := true
			opts.Emoji = &v
		case "--no-emoji":
			v := false
			opts.Emoji = &v
		case "-N", "--candidates":
			if i+1 >= len(args) {
				return opts, errors.New("missing value for --candidates")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 {
				return opts, fmt.Errorf("invalid candidates value: %q", args[i+1])
			}
			opts.Candidates = &n
			i++
		default:
			return opts, fmt.Errorf("unknown cz option: %s", a)
		}
	}
	return opts, nil
}

func loadCzConfig(opts czOptions) (czConfig, error) {
	cfg := defaultCzConfig()

	progCfgPath := ""
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		progCfgPath = firstExistingFile(filepath.Join(exeDir, "aiw.toml"), filepath.Join(exeDir, ".aiw.toml"))
	}
	if progCfgPath != "" {
		if err := mergeCzConfigFromTomlFile(&cfg, progCfgPath); err != nil {
			return cfg, fmt.Errorf("load program config %s: %w", progCfgPath, err)
		}
	}

	projectCfgPath := ""
	if root, rootErr := detectProjectRoot(); rootErr == nil {
		projectCfgPath = firstExistingFile(filepath.Join(root, "aiw.toml"), filepath.Join(root, ".aiw.toml"))
	}
	if projectCfgPath != "" {
		if err := mergeCzConfigFromTomlFile(&cfg, projectCfgPath); err != nil {
			return cfg, fmt.Errorf("load project config %s: %w", projectCfgPath, err)
		}
	}

	if opts.UseLLM != nil {
		cfg.UseLLM = *opts.UseLLM
	}
	if opts.Candidates != nil {
		cfg.Candidates = *opts.Candidates
	}
	if opts.Emoji != nil {
		cfg.Emoji = *opts.Emoji
	}
	return cfg, nil
}

func detectProjectRoot() (string, error) {
	out, err := gitOutput("git", "rev-parse", "--show-toplevel")
	if err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return wd, nil
}

func firstExistingFile(paths ...string) string {
	for _, p := range paths {
		if fsx.Exists(p) {
			return p
		}
	}
	return ""
}

func defaultCzConfig() czConfig {
	return czConfig{
		UseLLM:     false,
		Candidates: 3,
		Emoji:      false,
		Messages: czMessages{
			Type:               "选择你要提交的类型 / Select commit type:",
			Scope:              "选择一个提交范围（可选）/ Scope (optional):",
			CustomScope:        "请输入自定义的提交范围 / Input custom scope:",
			Subject:            "填写简短精炼的变更描述 / Subject:\n",
			Body:               "填写更加详细的变更描述（可选） / Body, use | for new line:\n",
			Breaking:           "列举非兼容性重大的变更（可选） / Breaking changes, use | for new line:\n",
			FooterPrefixes:     "选择关联 issue 前缀（可选） / Select issue prefix (optional):",
			CustomFooterPrefix: "输入自定义 issue 前缀 / Input custom issue prefix:",
			Footer:             "列举关联 issue（可选），例如: #31, #I3244:\n",
			ConfirmCommit:      "是否提交或修改 commit? / Commit or modify?",
		},
		Types: []czType{
			{Value: "feat", Name: "feat:     新增功能 | A new feature"},
			{Value: "fix", Name: "fix:      修复缺陷 | A bug fix"},
			{Value: "docs", Name: "docs:     文档更新 | Documentation only changes"},
			{Value: "style", Name: "style:    代码格式 | Changes that do not affect the meaning of the code"},
			{Value: "refactor", Name: "refactor: 代码重构 | A code change that neither fixes a bug nor adds a feature"},
			{Value: "perf", Name: "perf:     性能提升 | A code change that improves performance"},
			{Value: "test", Name: "test:     测试相关 | Adding missing tests or correcting existing tests"},
			{Value: "build", Name: "build:    构建相关 | Changes that affect the build system or external dependencies"},
			{Value: "ci", Name: "ci:       持续集成 | Changes to our CI configuration files and scripts"},
			{Value: "revert", Name: "revert:   回退代码 | Revert to a commit"},
			{Value: "chore", Name: "chore:    其他修改 | Other changes that do not modify src or test files"},
		},
	}
}

func mergeCzConfigFromTomlFile(cfg *czConfig, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	section := ""
	curType := map[string]string{}
	hasType := false
	typeSectionSeen := false

	applyType := func() {
		if !hasType {
			return
		}
		v := strings.TrimSpace(curType["value"])
		n := strings.TrimSpace(curType["name"])
		if v != "" && n != "" {
			cfg.Types = append(cfg.Types, czType{Value: v, Name: n})
		}
		curType = map[string]string{}
		hasType = false
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			applyType()
			tag := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
			section = tag
			if section == "cz.types" {
				if !typeSectionSeen {
					cfg.Types = nil
					typeSectionSeen = true
				}
				hasType = true
			}
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			applyType()
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valRaw := strings.TrimSpace(parts[1])
		val := parseTomlStringValue(valRaw)

		switch section {
		case "cz":
			switch key {
			case "llm", "use_llm":
				if b, ok := parseTomlBool(valRaw); ok {
					cfg.UseLLM = b
				}
			case "model", "llm_model", "openai_model":
				cfg.LLMModel = val
			case "base_url", "openai_base_url":
				cfg.APIBaseURL = val
			case "api_key", "openai_api_key":
				cfg.APIKey = val
			case "debug_source", "debug":
				if b, ok := parseTomlBool(valRaw); ok {
					cfg.DebugSource = b
				}
			case "candidates":
				if n, ok := parseTomlInt(valRaw); ok && n > 0 {
					cfg.Candidates = n
				}
			case "emoji":
				if b, ok := parseTomlBool(valRaw); ok {
					cfg.Emoji = b
				}
			case "editor", "EDITOR":
				cfg.Editor = val
			}
		case "cz.messages":
			switch key {
			case "type":
				cfg.Messages.Type = val
			case "scope":
				cfg.Messages.Scope = val
			case "customScope":
				cfg.Messages.CustomScope = val
			case "subject":
				cfg.Messages.Subject = val
			case "body":
				cfg.Messages.Body = val
			case "breaking":
				cfg.Messages.Breaking = val
			case "footerPrefixesSelect":
				cfg.Messages.FooterPrefixes = val
			case "customFooterPrefix":
				cfg.Messages.CustomFooterPrefix = val
			case "footer":
				cfg.Messages.Footer = val
			case "confirmCommit":
				cfg.Messages.ConfirmCommit = val
			}
		case "cz.types":
			if !hasType {
				hasType = true
			}
			curType[key] = val
		}
	}
	applyType()
	if len(cfg.Types) == 0 {
		cfg.Types = defaultCzConfig().Types
	}
	return scanner.Err()
}

func parseTomlStringValue(raw string) string {
	v := strings.TrimSpace(raw)
	if idx := strings.Index(v, "#"); idx >= 0 {
		prefix := strings.TrimSpace(v[:idx])
		if strings.Count(prefix, `"`)%2 == 0 {
			v = prefix
		}
	}
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) && len(v) >= 2 {
		v = strings.Trim(v, `"`)
	}
	return strings.ReplaceAll(v, `\n`, "\n")
}

func parseTomlBool(raw string) (bool, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func parseTomlInt(raw string) (int, bool) {
	v := strings.TrimSpace(raw)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

func czDraftFromWizard(cfg czConfig) (czDraft, error) {
	d := czDraft{}

	t, err := promptType(cfg)
	if err != nil {
		return d, err
	}
	d.Type = t

	scope := strings.TrimSpace(promptLine(cfg.Messages.Scope + " "))
	if scope == "." || scope == "custom" {
		scope = strings.TrimSpace(promptLine(cfg.Messages.CustomScope + " "))
	}
	d.Scope = scope

	for {
		d.Subject = strings.TrimSpace(promptLine(cfg.Messages.Subject))
		if d.Subject != "" {
			break
		}
		fmt.Fprintln(os.Stderr, "subject is required")
	}

	d.Body = normalizeMultiline(strings.TrimSpace(promptLine(cfg.Messages.Body)))
	if d.Body, err = promptMultilineWithEditor(cfg.Messages.Body, cfg, ""); err != nil {
		return d, err
	}
	if d.Breaking, err = promptMultilineWithEditor(cfg.Messages.Breaking, cfg, ""); err != nil {
		return d, err
	}
	prefix := strings.TrimSpace(promptLine(cfg.Messages.FooterPrefixes + " [#, refs, closes, custom, skip] "))
	if strings.EqualFold(prefix, "custom") {
		prefix = strings.TrimSpace(promptLine(cfg.Messages.CustomFooterPrefix + " "))
	}
	footer, err := promptMultilineWithEditor(cfg.Messages.Footer, cfg, "")
	if err != nil {
		return d, err
	}
	footer = strings.TrimSpace(footer)
	if footer != "" && prefix != "" && !strings.HasPrefix(strings.ToLower(footer), strings.ToLower(prefix)) {
		footer = prefix + " " + footer
	}
	d.Footer = normalizeMultiline(footer)
	return d, nil
}

func promptType(cfg czConfig) (string, error) {
	fmt.Println(cfg.Messages.Type)
	for i, t := range cfg.Types {
		fmt.Printf("  %2d) %s\n", i+1, t.Name)
	}
	for {
		in := strings.TrimSpace(promptLine("Type # or value: "))
		if in == "" {
			return "", errors.New("type is required")
		}
		if idx, err := strconv.Atoi(in); err == nil {
			if idx >= 1 && idx <= len(cfg.Types) {
				return cfg.Types[idx-1].Value, nil
			}
		}
		for _, t := range cfg.Types {
			if t.Value == in {
				return in, nil
			}
		}
		fmt.Fprintln(os.Stderr, "invalid type, try again")
	}
}

func promptLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

func normalizeMultiline(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, "\n")
}

func czDraftFromLLM(cfg czConfig) (czDraft, error) {
	diff, err := gitOutput("git", "diff", "--cached", "--")
	if err != nil {
		return czDraft{}, err
	}
	files, err := gitOutput("git", "diff", "--cached", "--name-only")
	if err != nil {
		return czDraft{}, err
	}
	hist, _ := gitOutput("git", "log", "--oneline", "-n", "5")

	if len(diff) > 12000 {
		diff = diff[:12000] + "\n... (truncated)"
	}
	typeList := make([]string, 0, len(cfg.Types))
	for _, t := range cfg.Types {
		typeList = append(typeList, t.Value)
	}

	prompt := fmt.Sprintf(`You are generating Conventional Commit messages.
Return strict JSON only, no prose.
Schema:
{"candidates":[{"type":"feat","scope":"optional","subject":"short","body":"optional","breaking":"optional","footer":"optional"}]}
Rules:
- candidate count: %d
- type must be one of: %s
- subject concise, imperative.
- output language follows changed code/comments language.

Changed files:
%s

Recent commits:
%s

Staged diff:
%s
`, cfg.Candidates, strings.Join(typeList, ", "), files, hist, diff)

	out, err := runCodex(prompt, cfg)
	if err != nil {
		return czDraft{}, err
	}

	allowedIssueRefs := detectIssueRefs(diff + "\n" + files + "\n" + hist)

	cands, err := parseLLMCandidates(out)
	if err != nil {
		return czDraft{}, fmt.Errorf("parse llm output: %w", err)
	}
	if len(cands) == 0 {
		return czDraft{}, errors.New("llm returned no candidates")
	}
	if len(cands) == 1 {
		d := sanitizeDraft(cands[0], cfg)
		d.Footer = filterLLMIssueFooter(d.Footer, allowedIssueRefs)
		return d, nil
	}

	fmt.Println("\nLLM candidates:")
	for i, c := range cands {
		preview := buildHeader(c, false)
		fmt.Printf("  %d) %s\n", i+1, preview)
	}
	for {
		in := strings.TrimSpace(promptLine("Select candidate # (or r=regen, q=cancel): "))
		switch strings.ToLower(in) {
		case "q", "quit", "n", "no":
			return czDraft{}, errors.New("aborted")
		case "r", "regen":
			return czDraftFromLLM(cfg)
		default:
			idx, convErr := strconv.Atoi(in)
			if convErr == nil && idx >= 1 && idx <= len(cands) {
				d := sanitizeDraft(cands[idx-1], cfg)
				d.Footer = filterLLMIssueFooter(d.Footer, allowedIssueRefs)
				return d, nil
			}
			fmt.Fprintln(os.Stderr, "invalid selection")
		}
	}
}

func detectIssueRefs(text string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, m := range issueRefRe.FindAllString(text, -1) {
		set[strings.ToLower(strings.TrimSpace(m))] = struct{}{}
	}
	return set
}

func filterLLMIssueFooter(footer string, allowed map[string]struct{}) string {
	footer = strings.TrimSpace(footer)
	if footer == "" {
		return ""
	}

	refs := issueRefRe.FindAllString(footer, -1)
	if len(refs) == 0 {
		// No issue number in footer, keep as-is.
		return footer
	}
	for _, r := range refs {
		if _, ok := allowed[strings.ToLower(strings.TrimSpace(r))]; !ok {
			return ""
		}
	}
	return footer
}

func runCodex(prompt string, cfg czConfig) (string, error) {
	cwdEnv, exeEnv, err := loadOpenAIEnvFromDotEnv()
	if err != nil {
		return "", err
	}

	model, modelSource := resolveOpenAIValue(cfg.LLMModel, "OPENAI_MODEL", cwdEnv, exeEnv, "gpt-4o-mini")
	baseURL, baseURLSource := resolveOpenAIValue(cfg.APIBaseURL, "OPENAI_BASE_URL", cwdEnv, exeEnv, "https://api.openai.com/v1")
	baseURL = strings.TrimRight(baseURL, "/")

	fmt.Fprintf(os.Stderr, "+ openai chat.completions model=%s endpoint=%s/chat/completions\n", model, baseURL)
	if dryRun {
		return `{"candidates":[{"type":"chore","scope":"","subject":"dry run preview","body":"","breaking":"","footer":""}]}`, nil
	}

	apiKey, apiKeySource := resolveOpenAIValue(cfg.APIKey, "OPENAI_API_KEY", cwdEnv, exeEnv, "")
	if shouldDebugSource(cfg) {
		printOpenAIDebugSource("OPENAI_MODEL", model, modelSource, false)
		printOpenAIDebugSource("OPENAI_BASE_URL", baseURL, baseURLSource, false)
		printOpenAIDebugSource("OPENAI_API_KEY", apiKey, apiKeySource, true)
	}
	if apiKey == "" {
		return "", errors.New("OPENAI_API_KEY is required for --llm mode")
	}

	out, status, err := callOpenAIChat(prompt, model, baseURL, apiKey, true)
	if err == nil {
		return strings.TrimSpace(out), nil
	}
	if status == http.StatusBadRequest {
		fallbackOut, _, fallbackErr := callOpenAIChat(prompt, model, baseURL, apiKey, false)
		if fallbackErr == nil {
			return strings.TrimSpace(fallbackOut), nil
		}
		return "", fallbackErr
	}
	return "", err
}

func resolveOpenAIValue(configValue, envKey string, cwdEnv, exeEnv map[string]string, defaultValue string) (string, string) {
	if v := strings.TrimSpace(configValue); v != "" {
		return v, "config"
	}
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, "env"
	}
	if v := strings.TrimSpace(cwdEnv[envKey]); v != "" {
		return v, "cwd .env"
	}
	if v := strings.TrimSpace(exeEnv[envKey]); v != "" {
		return v, "exe .env"
	}
	if defaultValue != "" {
		return defaultValue, "default"
	}
	return "", "missing"
}

func loadOpenAIEnvFromDotEnv() (map[string]string, map[string]string, error) {
	exeLoader := &envx.Loader{Env: map[string]string{}}
	cwdLoader := &envx.Loader{Env: map[string]string{}}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		exeEnv := filepath.Join(exeDir, ".env")
		if fsx.Exists(exeEnv) {
			if err := exeLoader.ParseFile(exeEnv); err != nil {
				return nil, nil, fmt.Errorf("parse %s: %w", exeEnv, err)
			}
		}
	}

	if wd, err := os.Getwd(); err == nil {
		wdEnv := filepath.Join(wd, ".env")
		if fsx.Exists(wdEnv) {
			if err := cwdLoader.ParseFile(wdEnv); err != nil {
				return nil, nil, fmt.Errorf("parse %s: %w", wdEnv, err)
			}
		}
	}

	return cwdLoader.Env, exeLoader.Env, nil
}

func shouldDebugSource(cfg czConfig) bool {
	if cfg.DebugSource {
		return true
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AIW_CZ_DEBUG")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func printOpenAIDebugSource(key, value, source string, secret bool) {
	display := value
	if secret {
		display = maskSecret(value)
	}
	fmt.Fprintf(os.Stderr, "[cz llm debug] %s=%s (source=%s)\n", key, display, source)
}

func maskSecret(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "<empty>"
	}
	if len(v) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(v)-4) + v[len(v)-4:]
}

func callOpenAIChat(prompt, model, baseURL, apiKey string, useSchema bool) (string, int, error) {
	reqBody := openAIChatRequest{
		Model: model,
		Messages: []openAIChatMessage{
			{
				Role: "system",
				Content: "You generate Conventional Commit candidates. Return JSON only.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
	}
	if useSchema {
		reqBody.ResponseFormat = &openAIResponseShape{
			Type: "json_schema",
			JSONSchema: &openAIJSONSchema{
				Name:   "cz_candidates",
				Strict: true,
				Schema: czCandidatesSchema(),
			},
		}
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	endpoint := baseURL + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 1200 {
			msg = msg[:1200] + "..."
		}
		return "", resp.StatusCode, fmt.Errorf("openai api failed (%d): %s", resp.StatusCode, msg)
	}

	content, err := extractOpenAIContent(body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return content, resp.StatusCode, nil
}

func extractOpenAIContent(raw []byte) (string, error) {
	var resp openAIChatResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("decode openai response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("openai response has no choices")
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("openai response content is empty")
	}
	return content, nil
}

func czCandidatesSchema() map[string]any {
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":     map[string]any{"type": "string"},
			"scope":    map[string]any{"type": "string"},
			"subject":  map[string]any{"type": "string"},
			"body":     map[string]any{"type": "string"},
			"breaking": map[string]any{"type": "string"},
			"footer":   map[string]any{"type": "string"},
		},
		"required":             []string{"type", "subject", "scope", "body", "breaking", "footer"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"candidates": map[string]any{
				"type":  "array",
				"items": item,
			},
		},
		"required":             []string{"candidates"},
		"additionalProperties": false,
	}
}

func parseLLMCandidates(out string) ([]czDraft, error) {
	if cands, ok := parseCandidatesJSON(strings.TrimSpace(out)); ok {
		return cands, nil
	}

	chunks := strings.Split(out, "```")
	for i := 1; i < len(chunks); i += 2 {
		block := strings.TrimSpace(chunks[i])
		block = strings.TrimPrefix(block, "json")
		if cands, ok := parseCandidatesJSON(strings.TrimSpace(block)); ok {
			return cands, nil
		}
	}

	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start >= 0 && end > start {
		candidate := out[start : end+1]
		if cands, ok := parseCandidatesJSON(candidate); ok {
			return cands, nil
		}
	}

	lines := strings.Split(out, "\n")
	var cands []czDraft
	for _, ln := range lines {
		ln = cleanCandidateLine(ln)
		if ln == "" {
			continue
		}
		if d, ok := parseConventionalHeader(ln); ok {
			cands = append(cands, d)
		}
	}
	if len(cands) == 0 {
		return nil, errors.New("invalid output")
	}
	return cands, nil
}

func parseCandidatesJSON(raw string) ([]czDraft, bool) {
	if raw == "" {
		return nil, false
	}
	var resp czLLMResponse
	if err := json.Unmarshal([]byte(raw), &resp); err == nil && len(resp.Candidates) > 0 {
		return resp.Candidates, true
	}

	var list []czDraft
	if err := json.Unmarshal([]byte(raw), &list); err == nil && len(list) > 0 {
		return list, true
	}
	return nil, false
}

func cleanCandidateLine(line string) string {
	v := strings.TrimSpace(line)
	v = strings.TrimPrefix(v, "-")
	v = strings.TrimPrefix(v, "*")
	v = strings.TrimPrefix(v, "•")
	v = strings.TrimSpace(v)

	if idx := strings.Index(v, ")"); idx > 0 {
		prefix := strings.TrimSpace(v[:idx])
		if _, err := strconv.Atoi(prefix); err == nil {
			v = strings.TrimSpace(v[idx+1:])
		}
	}
	if idx := strings.Index(v, "."); idx > 0 {
		prefix := strings.TrimSpace(v[:idx])
		if _, err := strconv.Atoi(prefix); err == nil {
			v = strings.TrimSpace(v[idx+1:])
		}
	}
	return v
}

func parseConventionalHeader(line string) (czDraft, bool) {
	i := strings.Index(line, ":")
	if i <= 0 || i+1 >= len(line) {
		return czDraft{}, false
	}

	left := strings.TrimSpace(line[:i])
	subject := strings.TrimSpace(line[i+1:])
	if left == "" || subject == "" {
		return czDraft{}, false
	}

	left = strings.TrimSuffix(left, "!")
	typePart := left
	scope := ""
	if l := strings.Index(left, "("); l >= 0 {
		r := strings.LastIndex(left, ")")
		if r <= l || r != len(left)-1 {
			return czDraft{}, false
		}
		typePart = strings.TrimSpace(left[:l])
		scope = strings.TrimSpace(left[l+1 : r])
		if scope == "" {
			return czDraft{}, false
		}
	}

	typePart = strings.ToLower(strings.TrimSpace(typePart))
	if typePart == "" || strings.ContainsAny(typePart, " `\t") {
		return czDraft{}, false
	}
	if _, ok := conventionalTypeSet()[typePart]; !ok {
		return czDraft{}, false
	}

	return czDraft{Type: typePart, Scope: scope, Subject: subject}, true
}

func conventionalTypeSet() map[string]struct{} {
	set := map[string]struct{}{}
	for _, t := range defaultCzConfig().Types {
		set[t.Value] = struct{}{}
	}
	return set
}

func sanitizeDraft(d czDraft, cfg czConfig) czDraft {
	d.Type = strings.TrimSpace(d.Type)
	d.Scope = strings.TrimSpace(d.Scope)
	d.Subject = strings.TrimSpace(d.Subject)
	d.Body = strings.TrimSpace(d.Body)
	d.Breaking = strings.TrimSpace(d.Breaking)
	d.Footer = strings.TrimSpace(d.Footer)
	if d.Type == "" {
		d.Type = cfg.Types[0].Value
	}
	if d.Subject == "" {
		d.Subject = "update"
	}
	valid := false
	for _, t := range cfg.Types {
		if t.Value == d.Type {
			valid = true
			break
		}
	}
	if !valid {
		d.Type = "chore"
	}
	return d
}

func editDraft(d czDraft, cfg czConfig) czDraft {
	fmt.Println("Modify fields (leave blank to keep current value).")
	fmt.Printf("Current type: %s\n", d.Type)
	newType := strings.TrimSpace(promptLine("Type: "))
	if newType != "" {
		d.Type = newType
	}
	fmt.Printf("Current scope: %s\n", d.Scope)
	if s := strings.TrimSpace(promptLine("Scope: ")); s != "" {
		d.Scope = s
	}
	fmt.Printf("Current subject: %s\n", d.Subject)
	if s := strings.TrimSpace(promptLine("Subject: ")); s != "" {
		d.Subject = s
	}
	fmt.Printf("Current body: %s\n", strings.ReplaceAll(d.Body, "\n", " | "))
	if s := strings.TrimSpace(promptLine("Body (use | for newline, /edit=editor): ")); s != "" {
		if isEditorShortcut(s) {
			if edited, err := editInExternalEditor(cfg, d.Body); err == nil {
				d.Body = normalizeMultiline(strings.TrimSpace(edited))
			} else {
				fmt.Fprintf(os.Stderr, "external editor failed: %v\n", err)
			}
		} else {
			d.Body = normalizeMultiline(s)
		}
	}
	fmt.Printf("Current breaking: %s\n", strings.ReplaceAll(d.Breaking, "\n", " | "))
	if s := strings.TrimSpace(promptLine("Breaking (use | for newline, /edit=editor): ")); s != "" {
		if isEditorShortcut(s) {
			if edited, err := editInExternalEditor(cfg, d.Breaking); err == nil {
				d.Breaking = normalizeMultiline(strings.TrimSpace(edited))
			} else {
				fmt.Fprintf(os.Stderr, "external editor failed: %v\n", err)
			}
		} else {
			d.Breaking = normalizeMultiline(s)
		}
	}
	fmt.Printf("Current footer: %s\n", strings.ReplaceAll(d.Footer, "\n", " | "))
	if s := strings.TrimSpace(promptLine("Footer (use | for newline, /edit=editor): ")); s != "" {
		if isEditorShortcut(s) {
			if edited, err := editInExternalEditor(cfg, d.Footer); err == nil {
				d.Footer = normalizeMultiline(strings.TrimSpace(edited))
			} else {
				fmt.Fprintf(os.Stderr, "external editor failed: %v\n", err)
			}
		} else {
			d.Footer = normalizeMultiline(s)
		}
	}
	return sanitizeDraft(d, cfg)
}

func promptMultilineWithEditor(prompt string, cfg czConfig, current string) (string, error) {
	in := strings.TrimSpace(promptLine(prompt + "(输入 /edit 或 Ctrl+E 文本快捷触发编辑器) "))
	if !isEditorShortcut(in) {
		return normalizeMultiline(in), nil
	}
	edited, err := editInExternalEditor(cfg, current)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(edited), nil
}

func isEditorShortcut(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "/edit", "^e", "ctrl+e", "control+e":
		return true
	}
	// Ctrl+E control character (ENQ) if terminal forwards it.
	return strings.ContainsRune(s, rune(5))
}

func editInExternalEditor(cfg czConfig, initial string) (string, error) {
	editor := resolveEditor(cfg)
	if strings.TrimSpace(editor) == "" {
		return "", errors.New("no editor available")
	}
	tmpDir, err := os.MkdirTemp("", "aiw-cz-edit-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := filepath.Join(tmpDir, "message.txt")
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return "", err
	}

	seed := ensureTrailingNewline(initial)
	if initial == "" {
		seed = "# Write your text below. Lines starting with # are ignored.\n"
	}
	if _, err := tmp.WriteString(seed); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if st, err := os.Stat(tmpPath); err != nil {
		return "", err
	} else if st.IsDir() {
		return "", fmt.Errorf("editor temp path became a directory before launch: %s", tmpPath)
	}

	if dryRun {
		fmt.Fprintf(os.Stderr, "+ %s %s\n", editor, tmpPath)
		fmt.Fprintln(os.Stderr, "  (dry-run: skipped editor launch)")
		return initial, nil
	}

	if err := runEditorCommand(editor, tmpPath); err != nil {
		return "", err
	}
	if st, err := os.Stat(tmpPath); err != nil {
		return "", err
	} else if st.IsDir() {
		return "", fmt.Errorf("editor temp path became a directory after editor exit: %s", tmpPath)
	}
	b, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	filtered := make([]string, 0, len(lines))
	for _, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "#") {
			continue
		}
		filtered = append(filtered, ln)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n")), nil
}

func runEditorCommand(editor, file string) error {
	args, err := buildEditorArgs(editor, file)
	if err != nil {
		return err
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildEditorArgs(editor, file string) ([]string, error) {
	parts, err := splitCommandLine(editor)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, errors.New("invalid editor command")
	}
	hasFile := false
	for i, p := range parts {
		if strings.Contains(p, "{file}") {
			parts[i] = strings.ReplaceAll(p, "{file}", file)
			hasFile = true
		}
	}
	if !hasFile {
		parts = append(parts, file)
	}
	return parts, nil
}

func splitCommandLine(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	var args []string
	var cur strings.Builder
	var quote rune

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		args = append(args, cur.String())
		cur.Reset()
	}

	for _, r := range s {
		switch {
		case quote == 0 && (r == '"' || r == '\''):
			quote = r
		case quote != 0 && r == quote:
			quote = 0
		case quote == 0 && (r == ' ' || r == '\t'):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, errors.New("editor command has unmatched quote")
	}
	flush()
	return args, nil
}

func resolveEditor(cfg czConfig) string {
	if strings.TrimSpace(cfg.Editor) != "" {
		return strings.TrimSpace(cfg.Editor)
	}
	for _, key := range []string{"GIT_EDITOR", "VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	if out, err := gitOutput("git", "config", "--get", "core.editor"); err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

func buildCommitMessage(d czDraft, emoji bool) string {
	header := buildHeader(d, emoji)
	parts := []string{header}
	if d.Body != "" {
		parts = append(parts, "", d.Body)
	}
	if d.Breaking != "" {
		parts = append(parts, "", "BREAKING CHANGE: "+d.Breaking)
	}
	if d.Footer != "" {
		parts = append(parts, "", d.Footer)
	}
	return strings.Join(parts, "\n")
}

func buildHeader(d czDraft, emoji bool) string {
	header := d.Type
	if d.Scope != "" {
		header += "(" + d.Scope + ")"
	}
	header += ": " + d.Subject
	if emoji {
		if e := emojiForType(d.Type); e != "" {
			header = e + " " + header
		}
	}
	return header
}

func emojiForType(t string) string {
	switch t {
	case "feat":
		return "✨"
	case "fix":
		return "🐛"
	case "docs":
		return "📝"
	case "style":
		return "🎨"
	case "refactor":
		return "♻️"
	case "perf":
		return "⚡"
	case "test":
		return "✅"
	case "build":
		return "📦"
	case "ci":
		return "👷"
	case "revert":
		return "⏪"
	case "chore":
		return "🔧"
	default:
		return ""
	}
}

func commitWithMessage(msg string) error {
	tmp, err := os.CreateTemp("", "aiw-cz-*.msg")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(ensureTrailingNewline(msg)); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return run("git", "commit", "-F", tmp.Name())
}

func run(name string, args ...string) error {
	fmt.Fprintf(os.Stderr, "+ %s %s\n", name, strings.Join(args, " "))
	if dryRun {
		fmt.Fprintln(os.Stderr, "  (dry-run: skipped)")
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func gitOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
