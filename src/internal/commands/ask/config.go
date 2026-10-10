package ask

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// resolveSystemPrompt applies the ask-specific precedence without consulting
// project configuration: command-line text, command-line file, then private
// ask configuration. An absent private configuration is equivalent to no prompt.
func resolveSystemPrompt(o options, policy *readPolicy, interactive bool) (string, error) {
	if o.systemPrompt != "" {
		return o.systemPrompt, nil
	}
	if o.systemPromptFile != "" {
		return readSystemPromptFile(o.systemPromptFile, policy, true, interactive)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory for ask configuration: %w", err)
	}
	configPath := filepath.Join(home, ".aiw", "ask", "config.toml")
	config, err := loadConfig(configPath)
	if err != nil {
		return "", err
	}
	if config.systemPrompt != "" {
		return config.systemPrompt, nil
	}
	if config.systemPromptFile != "" {
		return readSystemPromptFile(config.systemPromptFile, policy, false, interactive)
	}
	return "", nil
}

type askConfig struct {
	systemPrompt     string
	systemPromptFile string
}

func loadConfig(path string) (askConfig, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return askConfig{}, nil
	}
	if err != nil {
		return askConfig{}, fmt.Errorf("open ask configuration %s: %w", path, err)
	}
	defer file.Close()

	var config askConfig
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
		if section != "ask" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value, ok := parseConfigValue(strings.TrimSpace(parts[1]))
		if !ok {
			continue
		}
		switch strings.TrimSpace(parts[0]) {
		case "system_prompt":
			config.systemPrompt = value
		case "system_prompt_file":
			config.systemPromptFile = value
		}
	}
	if err := scanner.Err(); err != nil {
		return askConfig{}, fmt.Errorf("read ask configuration %s: %w", path, err)
	}
	return config, nil
}

func readSystemPromptFile(path string, policy *readPolicy, explicit, interactive bool) (string, error) {
	path, err := policy.authorizeSystemPromptFile(path, explicit, interactive)
	if err != nil {
		return "", err
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read system prompt file %s: %w", path, err)
	}
	return string(contents), nil
}

func parseConfigValue(raw string) (string, bool) {
	if strings.HasPrefix(raw, "\"") || strings.HasPrefix(raw, "'") {
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", false
		}
		return value, true
	}
	if index := strings.IndexAny(raw, "#;"); index >= 0 {
		raw = strings.TrimSpace(raw[:index])
	}
	return strings.TrimSpace(raw), raw != ""
}
