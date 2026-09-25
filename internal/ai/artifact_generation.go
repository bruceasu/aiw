package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const MaxArtifactProfiles = 8

// ArtifactProfile is resolved independently. Config (including credentials) is
// deliberately excluded from JSON; only the public budget belongs in audit.
type ArtifactProfile struct {
	ID string `json:"id"`
	Profile string `json:"profile"`
	Provider string `json:"provider"`
	Model string `json:"model"`
	Endpoint string `json:"endpoint,omitempty"`
	Command string `json:"command,omitempty"`
	Diagnostic string `json:"diagnostic,omitempty"`
	Config Config `json:"-"`
}

// LoadArtifactProfiles never uses auto detection or the general fallback chain.
func LoadArtifactProfiles() ([]ArtifactProfile, error) {
	values, err := loadConfigValues()
	if err != nil { return nil, err }
	return artifactProfiles(values, os.Getenv)
}

func artifactProfiles(values map[string]string, env func(string) string) ([]ArtifactProfile, error) {
	names := []string{""}
	if raw, present := values["ai.artifact_generation.profiles"]; present {
		var err error
		names, err = artifactProfileNames(raw)
		if err != nil { return nil, err }
	}
	fileProvider := artifactProvider(firstValue(values, "ai.provider", "ai.llm_provider"))
	global := fileProvider
	if v := strings.TrimSpace(env("AIW_LLM_PROVIDER")); v != "" { global = artifactProvider(v) }
	result := []ArtifactProfile{}
	seen := map[string]bool{}
	for _, name := range names {
		prefix := "ai.profiles." + name + "."
		provider := artifactProvider(values[prefix+"provider"])
		if name == "" { provider = global }
		if name == "" && (provider == "" || provider == "auto") { continue }
		cfg := Config{Name: provider}
		// Generic transport settings belong only to the explicit global provider.
		if provider == fileProvider && provider != "" && provider != "auto" {
			cfg.Model = firstValue(values, "ai.model", "ai.llm_model")
			cfg.BaseURL = firstValue(values, "ai.base_url", "ai.llm_base_url")
			cfg.APIKey = firstValue(values, "ai.api_key", "ai.llm_api_key")
			cfg.Command = values["ai.command"]
		}
		stem := provider
		if provider == "llama.cpp" { stem = "llamacpp" }
		for _, item := range []struct { suffix string; target *string }{
			{"model", &cfg.Model}, {"base_url", &cfg.BaseURL}, {"api_key", &cfg.APIKey}, {"command", &cfg.Command},
		} {
			if v := strings.TrimSpace(values["ai."+stem+"_"+item.suffix]); v != "" { *item.target = v }
			if name != "" {
				if v := strings.TrimSpace(values[prefix+item.suffix]); v != "" { *item.target = v }
			}
		}
		modelEnv, baseEnv, keyEnv := providerEnvNames(provider)
		for _, item := range []struct { key string; target *string }{
			{modelEnv, &cfg.Model}, {baseEnv, &cfg.BaseURL}, {keyEnv, &cfg.APIKey},
		} {
			if item.key != "" { if v := strings.TrimSpace(env(item.key)); v != "" { *item.target = v } }
		}
		if provider == global && provider != "" && provider != "auto" {
			for _, item := range []struct { key string; target *string }{
				{"AIW_LLM_MODEL", &cfg.Model}, {"AIW_LLM_BASE_URL", &cfg.BaseURL}, {"AIW_LLM_API_KEY", &cfg.APIKey},
			} {
				if v := strings.TrimSpace(env(item.key)); v != "" { *item.target = v }
			}
		}
		// An explicitly selected Profile model wins over model environment defaults.
		if name != "" { cfg.Model = strings.TrimSpace(values[prefix+"model"]) }
		applyProviderDefaults(&cfg)
		if (provider == "codex" || provider == "copilot") && cfg.Command == "" { cfg.Command = provider }
		if provider == "ollama" && !strings.HasSuffix(strings.TrimRight(cfg.BaseURL, "/"), "/v1") { cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/") + "/v1" }
		cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
		item := ArtifactProfile{Profile: name, Provider: provider, Model: cfg.Model, Endpoint: cfg.BaseURL, Command: cfg.Command, Config: cfg}
		if _, err := NewProvider(cfg); err != nil || provider == "" || provider == "auto" {
			item.Diagnostic = "profile requires a supported explicit provider"
		}
		if name != "" && strings.TrimSpace(values[prefix+"model"]) == "" { item.Diagnostic = "profile requires an explicit model" }
		if cfg.Model == "" { item.Diagnostic = "artifact generation requires a resolved model" }
		if cfg.BaseURL != "" {
			u, err := url.Parse(cfg.BaseURL)
			if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
				return nil, fmt.Errorf("artifact generation endpoint must be an HTTP URL without credentials, query or fragment")
			}
		}
		// Credentials never enter the stable identity. Changing a key does not
		// replenish a consumed request budget; explicit regeneration is required.
		identity, _ := json.Marshal([]string{item.Provider, item.Model, item.Endpoint, item.Command, item.Diagnostic})
		sum := sha256.Sum256(identity)
		item.ID = hex.EncodeToString(sum[:])
		if seen[item.ID] { continue }
		seen[item.ID] = true
		result = append(result, item)
		if len(result) > MaxArtifactProfiles { return nil, fmt.Errorf("artifact generation supports at most %d distinct profiles", MaxArtifactProfiles) }
	}
	return result, nil
}

func artifactProvider(name string) string {
	switch name = normalize(name); name {
	case "codex-cli", "codex_cli", "codexcli", "exec": return "codex"
	case "copilot-cli", "copilot_cli", "copilotcli": return "copilot"
	case "llamacpp", "llama-cpp": return "llama.cpp"
	default: return name
	}
}

// Parse the small TOML string-array subset used by Profile references. Invalid
// configuration fails closed instead of silently invoking the global provider.
func artifactArrayClosed(raw string) bool {
	var quote rune
	comment, escaped := false, false
	for _, ch := range raw {
		if comment { if ch == '\n' { comment = false }; continue }
		if quote != 0 {
			if escaped { escaped = false; continue }
			if quote == '"' && ch == '\\' { escaped = true; continue }
			if ch == quote { quote = 0 }
			continue
		}
		switch ch {
		case '#': comment = true
		case '"', '\'': quote = ch
		case ']': return true
		}
	}
	return false
}

func artifactProfileNames(raw string) ([]string, error) {
	fail := func() ([]string, error) { return nil, fmt.Errorf("artifact_generation.profiles must be an array of quoted profile names") }
	trim := func(s string) string {
		for {
			s = strings.TrimSpace(s)
			if !strings.HasPrefix(s, "#") { return s }
			if i := strings.IndexByte(s, '\n'); i >= 0 { s = s[i+1:] } else { return "" }
		}
	}
	raw = trim(raw)
	if !strings.HasPrefix(raw, "[") { return fail() }
	raw = trim(raw[1:])
	names := []string{}
	for !strings.HasPrefix(raw, "]") {
		if len(raw) == 0 || (raw[0] != '"' && raw[0] != '\'') { return fail() }
		quote, end := raw[0], 1
		for end < len(raw) {
			if raw[end] == quote { break }
			if quote == '"' && raw[end] == '\\' { end++ }
			end++
		}
		if end >= len(raw) { return fail() }
		name := raw[1:end]
		if quote == '"' { var err error; name, err = strconv.Unquote(raw[:end+1]); if err != nil { return fail() } }
		if strings.TrimSpace(name) == "" || strings.ContainsAny(name, "\r\n") { return fail() }
		names = append(names, name)
		raw = trim(raw[end+1:])
		if strings.HasPrefix(raw, "]") { break }
		if !strings.HasPrefix(raw, ",") { return fail() }
		raw = trim(raw[1:])
	}
	if trim(raw[1:]) != "" { return fail() }
	return names, nil
}
