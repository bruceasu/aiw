package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"aiw/internal/envx"
	"aiw/internal/fsx"
)

type LLMConfig struct {
	Provider       string
	Model          string
	APIBaseURL     string
	APIKey         string
	OpenAIKey      string
	GeminiKey      string
	CodexCommand   string
	CopilotCommand string
	DryRun         bool
	DebugSource    bool
	// Workspace and AdditionalDirs limit filesystem-aware CLI providers for a
	// single call. They are deliberately opt-in so existing callers preserve
	// their current provider behavior.
	Workspace      string
	AdditionalDirs []string
	ReadOnly       bool
}

func RunLLM(prompt string, cfg LLMConfig, outputSchema map[string]any) (string, error) {
	return RunLLMWithSystemPrompt(prompt, cfg, outputSchema, "You generate Conventional Commit candidates. Return JSON only.")
}

func RunLLMWithSystemPrompt(prompt string, cfg LLMConfig, outputSchema map[string]any, systemPrompt string) (string, error) {
	provider := normalizeProviderName(cfg.Provider)
	if provider != "" {
		return runProvider(provider, prompt, cfg, outputSchema, systemPrompt)
	}

	providers, err := detectAvailableProviders(cfg)
	if err != nil {
		return "", err
	}
	if len(providers) == 0 {
		return "", errors.New("no available LLM provider detected; configure provider=openai|gemini|ollama|codex-cli|copilot-cli or set corresponding credentials/CLI")
	}

	var errs []string
	for _, p := range providers {
		out, runErr := runProvider(p, prompt, cfg, outputSchema, systemPrompt)
		if runErr == nil {
			return out, nil
		}
		errSummary := runErr.Error()
		if len(errSummary) > 200 {
			errSummary = errSummary[:200] + "..."
		}
		errs = append(errs, fmt.Sprintf("%s: %s", p, errSummary))
	}
	return "", fmt.Errorf("all auto-detected providers failed (%s)", strings.Join(errs, " | "))
}

func normalizeProviderName(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "", "auto":
		return ""
	case "openai", "gemini", "ollama", "codex", "copilot", "llama.cpp", "llamacpp", "llama-cpp":
		if p == "llama-cpp" {
			return "llama.cpp"
		}
		return p
	case "codex-cli", "codex_cli", "codexcli":
		return "codex"
	case "copilot-cli", "copilot_cli", "copilotcli":
		return "copilot"
	default:
		return p
	}
}

func runProvider(provider, prompt string, cfg LLMConfig, outputSchema map[string]any, systemPrompt string) (string, error) {
	if cfg.DryRun {
		return `{"candidates":[{"type":"chore","scope":"","subject":"dry run preview","body":"","breaking":"","footer":""}]}`, nil
	}
	cwdEnv, aiwEnv, exeEnv, err := loadLLMEnvFromDotEnv()
	if err != nil {
		return "", err
	}
	modelKey, defaultModel := providerModelEnv(provider)
	model, _ := resolveLLMValue(cfg.Model, modelKey, cwdEnv, aiwEnv, exeEnv, defaultModel)
	baseURL, _ := resolveLLMValue(cfg.APIBaseURL, providerBaseEnv(provider), cwdEnv, aiwEnv, exeEnv, providerDefaultBase(provider))
	configuredKey := cfg.APIKey
	if provider == "openai" && cfg.OpenAIKey != "" {
		configuredKey = cfg.OpenAIKey
	}
	if provider == "gemini" && cfg.GeminiKey != "" {
		configuredKey = cfg.GeminiKey
	}
	apiKey, _ := resolveLLMValue(configuredKey, providerKeyEnv(provider), cwdEnv, aiwEnv, exeEnv, "")
	command := ""
	if provider == "codex" {
		command = cfg.CodexCommand
	}
	if provider == "copilot" {
		command = cfg.CopilotCommand
	}
	p, err := NewProvider(Config{Name: provider, Model: model, APIKey: apiKey, BaseURL: baseURL, Command: command})
	if err != nil {
		return "", err
	}
	result, err := p.Generate(context.Background(), Request{
		Prompt:       prompt,
		Model:        model,
		SystemPrompt: systemPrompt,
		OutputSchema: outputSchema,
		Workspace:    cfg.Workspace,
		AdditionalDirs: cfg.AdditionalDirs,
		ReadOnly:     cfg.ReadOnly,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.FinalOutput), nil
}

func providerModelEnv(provider string) (string, string) {
	switch provider {
	case "openai":
		return "OPENAI_MODEL", "gpt-4o-mini"
	case "gemini":
		return "GEMINI_MODEL", "gemini-1.5-flash"
	case "ollama":
		return "OLLAMA_MODEL", "llama3"
	case "llama.cpp":
		return "LLAMACPP_MODEL", "local-model"
	default:
		return "", ""
	}
}

func providerBaseEnv(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_BASE_URL"
	case "gemini":
		return "GEMINI_BASE_URL"
	case "ollama":
		return "OLLAMA_BASE_URL"
	case "llama.cpp":
		return "LLAMACPP_BASE_URL"
	default:
		return ""
	}
}

func providerKeyEnv(provider string) string {
	switch provider {
	case "gemini":
		return "GEMINI_API_KEY"
	case "ollama", "llama.cpp":
		return ""
	default:
		return "OPENAI_API_KEY"
	}
}

func providerDefaultBase(provider string) string {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com"
	case "ollama":
		return "http://localhost:11434"
	case "llama.cpp":
		return "http://localhost:8080/v1"
	default:
		return ""
	}
}

func detectAvailableProviders(cfg LLMConfig) ([]string, error) {
	cwdEnv, aiwEnv, exeEnv, err := loadLLMEnvFromDotEnv()
	if err != nil {
		return nil, err
	}

	providers := []string{}
	seen := map[string]struct{}{}
	add := func(p string) {
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		providers = append(providers, p)
	}

	if codexCLIAvailable(cfg) {
		add("codex")
	}
	if copilotCLIAvailable(cfg) {
		add("copilot")
	}
	if openAIConfigured(cfg, cwdEnv, aiwEnv, exeEnv) {
		add("openai")
	}
	if geminiConfigured(cfg, cwdEnv, aiwEnv, exeEnv) {
		add("gemini")
	}
	// Keep ollama as a final fallback for local environments.
	add("ollama")

	return providers, nil
}

func openAIConfigured(cfg LLMConfig, cwdEnv, aiwEnv, exeEnv map[string]string) bool {
	v, _ := resolveLLMValue(cfg.OpenAIKey, "OPENAI_API_KEY", cwdEnv, aiwEnv, exeEnv, "")
	if strings.TrimSpace(v) != "" {
		return true
	}
	v, _ = resolveLLMValue(cfg.APIKey, "OPENAI_API_KEY", cwdEnv, aiwEnv, exeEnv, "")
	return strings.TrimSpace(v) != ""
}

func geminiConfigured(cfg LLMConfig, cwdEnv, aiwEnv, exeEnv map[string]string) bool {
	v, _ := resolveLLMValue(cfg.GeminiKey, "GEMINI_API_KEY", cwdEnv, aiwEnv, exeEnv, "")
	if strings.TrimSpace(v) != "" {
		return true
	}
	v, _ = resolveLLMValue(cfg.APIKey, "GEMINI_API_KEY", cwdEnv, aiwEnv, exeEnv, "")
	return strings.TrimSpace(v) != ""
}

func codexCLIAvailable(cfg LLMConfig) bool {
	name := strings.TrimSpace(cfg.CodexCommand)
	if name == "" {
		name = "codex"
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func copilotCLIAvailable(cfg LLMConfig) bool {
	name := strings.TrimSpace(cfg.CopilotCommand)
	if name == "" {
		name = "copilot"
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func supportedProviderNames() string {
	providers := []string{"openai", "gemini", "ollama", "llama.cpp", "codex-cli", "copilot-cli"}
	sort.Strings(providers)
	return strings.Join(providers, ", ")
}

func resolveLLMValue(configValue, envKey string, cwdEnv, aiwEnv, exeEnv map[string]string, defaultValue string) (string, string) {
	if v := strings.TrimSpace(configValue); v != "" {
		return v, "config"
	}
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, "env"
	}
	if v := strings.TrimSpace(cwdEnv[envKey]); v != "" {
		return v, "cwd .env"
	}
	if v := strings.TrimSpace(aiwEnv[envKey]); v != "" {
		return v, "aiw_root .env"
	}
	if v := strings.TrimSpace(exeEnv[envKey]); v != "" {
		return v, "exe .env"
	}
	if defaultValue != "" {
		return defaultValue, "default"
	}
	return "", "missing"
}

func loadLLMEnvFromDotEnv() (map[string]string, map[string]string, map[string]string, error) {
	exeLoader := &envx.Loader{Env: map[string]string{}}
	aiwLoader := &envx.Loader{Env: map[string]string{}}
	cwdLoader := &envx.Loader{Env: map[string]string{}}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		exeEnv := filepath.Join(exeDir, ".env")
		if fsx.Exists(exeEnv) {
			if err := exeLoader.ParseFile(exeEnv); err != nil {
				return nil, nil, nil, fmt.Errorf("parse %s: %w", exeEnv, err)
			}
		}
	}

	if aiwRoot := os.Getenv("AIW_ROOT"); aiwRoot != "" {
		aiwEnv := filepath.Join(aiwRoot, ".env")
		if fsx.Exists(aiwEnv) {
			if err := aiwLoader.ParseFile(aiwEnv); err != nil {
				return nil, nil, nil, fmt.Errorf("parse %s: %w", aiwEnv, err)
			}
		}
	}

	if wd, err := os.Getwd(); err == nil {
		wdEnv := filepath.Join(wd, ".env")
		if fsx.Exists(wdEnv) {
			if err := cwdLoader.ParseFile(wdEnv); err != nil {
				return nil, nil, nil, fmt.Errorf("parse %s: %w", wdEnv, err)
			}
		}
	}

	return cwdLoader.Env, aiwLoader.Env, exeLoader.Env, nil
}

func shouldDebugSource(cfg LLMConfig) bool {
	if cfg.DebugSource {
		return true
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AIW_CZ_DEBUG")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func printLLMDebugSource(key, value, source string, secret bool) {
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
