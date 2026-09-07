package ai

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadConfig reads the global AI configuration. The canonical section is
// [ai]; provider settings in [cz] remain supported as a legacy fallback.
func LoadConfig() (Config, error) {
	values := map[string]string{}
	for _, path := range configPaths() {
		if err := mergeConfigFile(values, path); err != nil {
			return Config{}, err
		}
	}
	return configFromValues(values), nil
}

// ConfigFor resolves a named provider using global file configuration and
// environment overrides while preserving an explicit model from the caller.
func ConfigFor(name, model string) (Config, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return Config{}, err
	}
	cfg.Name = name
	if model != "" {
		cfg.Model = model
	}
	applyProviderDefaults(&cfg)
	return cfg, nil
}

func configPaths() []string {
	paths := []string{}
	if root := findProjectRoot(); root != "" {
		paths = append(paths, filepath.Join(root, "aiw.toml"), filepath.Join(root, ".aiw.toml"))
	}
	if root := os.Getenv("AIW_ROOT"); root != "" {
		paths = append(paths, filepath.Join(root, "aiw.toml"), filepath.Join(root, ".aiw.toml"))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		paths = append(paths, filepath.Join(dir, "aiw.toml"), filepath.Join(dir, ".aiw.toml"))
	}
	return paths
}

func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if fileExists(filepath.Join(dir, "aiw.toml")) || fileExists(filepath.Join(dir, ".aiw.toml")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func mergeConfigFile(values map[string]string, path string) error {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open AI config %s: %w", path, err)
	}
	defer file.Close()
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		if section != "ai" && section != "cz" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value, ok := parseConfigValue(strings.TrimSpace(parts[1]))
		if !ok || !isAIConfigKey(key) {
			continue
		}
		values[section+"."+key] = value
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read AI config %s: %w", path, err)
	}
	return nil
}

func configFromValues(values map[string]string) Config {
	name := firstValue(values, "ai.provider", "ai.llm_provider", "cz.provider", "cz.llm_provider")
	name = normalize(name)
	model := firstValue(values, "ai.model", "ai.llm_model", "cz.model", "cz.llm_model")
	baseURL := firstValue(values, "ai.base_url", "ai.llm_base_url", "cz.base_url", "cz.llm_base_url")
	apiKey := firstValue(values, "ai.api_key", "ai.llm_api_key", "cz.api_key", "cz.llm_api_key")
	switch name {
	case "openai":
		model = firstValue(values, "ai.openai_model", "cz.openai_model", "ai.model", "cz.model")
		baseURL = firstValue(values, "ai.openai_base_url", "cz.openai_base_url", "ai.base_url", "cz.base_url")
		apiKey = firstValue(values, "ai.openai_api_key", "cz.openai_api_key", "ai.api_key", "cz.api_key")
	case "gemini":
		model = firstValue(values, "ai.gemini_model", "cz.gemini_model", "ai.model", "cz.model")
		baseURL = firstValue(values, "ai.gemini_base_url", "cz.gemini_base_url", "ai.base_url", "cz.base_url")
		apiKey = firstValue(values, "ai.gemini_api_key", "cz.gemini_api_key", "ai.api_key", "cz.api_key")
	case "ollama":
		model = firstValue(values, "ai.ollama_model", "cz.ollama_model", "ai.model", "cz.model")
		baseURL = firstValue(values, "ai.ollama_base_url", "cz.ollama_base_url", "ai.base_url", "cz.base_url")
		apiKey = firstValue(values, "ai.ollama_api_key", "cz.ollama_api_key", "ai.api_key", "cz.api_key")
	case "llama.cpp", "llamacpp", "llama-cpp":
		model = firstValue(values, "ai.llamacpp_model", "cz.llamacpp_model", "ai.model", "cz.model")
		baseURL = firstValue(values, "ai.llamacpp_base_url", "cz.llamacpp_base_url", "ai.base_url", "cz.base_url")
		apiKey = firstValue(values, "ai.llamacpp_api_key", "cz.llamacpp_api_key", "ai.api_key", "cz.api_key")
	}
	command := firstValue(values, "ai.command", "cz.command")
	if name == "codex" || name == "codex-cli" {
		command = firstValue(values, "ai.codex_command", "cz.codex_command", "ai.command", "cz.command")
	}
	if name == "copilot" || name == "copilot-cli" {
		command = firstValue(values, "ai.copilot_command", "cz.copilot_command", "ai.command", "cz.command")
	}

	modelEnv, baseEnv, keyEnv := providerEnvNames(name)
	if modelEnv != "" {
		if v := strings.TrimSpace(os.Getenv(modelEnv)); v != "" {
			model = v
		}
	}
	if baseEnv != "" {
		if v := strings.TrimSpace(os.Getenv(baseEnv)); v != "" {
			baseURL = v
		}
	}
	if keyEnv != "" {
		if v := strings.TrimSpace(os.Getenv(keyEnv)); v != "" {
			apiKey = v
		}
	}
	if v := strings.TrimSpace(os.Getenv("AIW_LLM_PROVIDER")); v != "" {
		name = normalize(v)
	}
	if v := strings.TrimSpace(os.Getenv("AIW_LLM_MODEL")); v != "" {
		model = v
	}
	if v := strings.TrimSpace(os.Getenv("AIW_LLM_BASE_URL")); v != "" {
		baseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("AIW_LLM_API_KEY")); v != "" {
		apiKey = v
	}

	cfg := Config{Name: name, Model: model, APIKey: apiKey, BaseURL: baseURL, Command: command}
	if name != "" && name != "auto" {
		applyProviderDefaults(&cfg)
	}
	return cfg
}

func applyProviderDefaults(cfg *Config) {
	switch normalize(cfg.Name) {
	case "openai":
		if cfg.Model == "" { cfg.Model = "gpt-4o-mini" }
		if cfg.BaseURL == "" { cfg.BaseURL = "https://api.openai.com/v1" }
	case "gemini":
		if cfg.Model == "" { cfg.Model = "gemini-1.5-flash" }
		if cfg.BaseURL == "" { cfg.BaseURL = "https://generativelanguage.googleapis.com" }
	case "ollama":
		if cfg.Model == "" { cfg.Model = "llama3" }
		if cfg.BaseURL == "" { cfg.BaseURL = "http://localhost:11434" }
	case "llama.cpp", "llamacpp", "llama-cpp":
		if cfg.Model == "" { cfg.Model = "local-model" }
		if cfg.BaseURL == "" { cfg.BaseURL = "http://localhost:8080/v1" }
	}
}

func providerEnvNames(name string) (string, string, string) {
	switch normalize(name) {
	case "openai": return "OPENAI_MODEL", "OPENAI_BASE_URL", "OPENAI_API_KEY"
	case "gemini": return "GEMINI_MODEL", "GEMINI_BASE_URL", "GEMINI_API_KEY"
	case "ollama": return "OLLAMA_MODEL", "OLLAMA_BASE_URL", "OLLAMA_API_KEY"
	case "llama.cpp", "llamacpp", "llama-cpp": return "LLAMACPP_MODEL", "LLAMACPP_BASE_URL", "LLAMACPP_API_KEY"
	default: return "", "", ""
	}
}

func firstValue(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func isAIConfigKey(key string) bool {
	return map[string]bool{
		"provider": true, "llm_provider": true, "model": true, "llm_model": true,
		"base_url": true, "llm_base_url": true, "api_key": true, "llm_api_key": true,
		"command": true, "codex_command": true, "copilot_command": true,
		"openai_model": true, "openai_base_url": true, "openai_api_key": true,
		"gemini_model": true, "gemini_base_url": true, "gemini_api_key": true,
		"ollama_model": true, "ollama_base_url": true, "ollama_api_key": true,
		"llamacpp_model": true, "llamacpp_base_url": true, "llamacpp_api_key": true,
	}[key]
}

func parseConfigValue(raw string) (string, bool) {
	if strings.HasPrefix(raw, "\"") || strings.HasPrefix(raw, "'") {
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", false
		}
		return value, true
	}
	if idx := strings.IndexAny(raw, "#;"); idx >= 0 {
		raw = strings.TrimSpace(raw[:idx])
	}
	return strings.TrimSpace(raw), raw != ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
