package say

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

// Config contains the effective user-facing say settings.
type Config struct {
	Source    string
	Target    string
	Mode      string
	Style     string
	Polite    string
	Simple    bool
	Profanity string
	Provider  string
	Model     string
	Timeout   string
}

func DefaultConfig() Config {
	return Config{Source: "auto", Target: "ja", Mode: "realtime", Style: "spoken", Polite: "polite", Profanity: "mask", Provider: "openai", Timeout: "30s"}
}

// LoadBase reads the executable-directory aiw.toml, or the explicit override.
// A missing default file is allowed; an explicit missing file is an error.
func LoadBase(exePath, override string) (Config, error) {
	cfg := DefaultConfig()
	path := override
	explicit := override != ""
	if !explicit {
		if exePath == "" {
			return cfg, errors.New("resolve executable path for aiw.toml")
		}
		path = filepath.Join(filepath.Dir(exePath), "aiw.toml")
	}
	values, exists, err := readConfig(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if !exists {
		if explicit {
			return Config{}, fmt.Errorf("config file does not exist: %s", path)
		}
		return cfg, nil
	}
	if err := applyConfig(&cfg, values); err != nil {
		return Config{}, fmt.Errorf("config %s: %w", path, err)
	}
	return cfg, nil
}

func readConfig(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !utf8.Valid(data) {
		return nil, true, errors.New("file is not valid UTF-8")
	}
	var document map[string]any
	if _, err := toml.Decode(string(data), &document); err != nil {
		return nil, true, err
	}
	values := make(map[string]any)
	addConfigTable(values, "say", document["say"])
	if say, ok := document["say"].(map[string]any); ok {
		addConfigTable(values, "say.llm", say["llm"])
	}
	addConfigTable(values, "profile", document["profile"])
	return values, true, nil
}

func addConfigTable(values map[string]any, section string, raw any) {
	table, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for key, value := range table {
		values[section+"."+key] = value
	}
}

func applyConfig(cfg *Config, values map[string]any) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, ok := configValueString(key, values[key])
		if !ok {
			continue
		}
		candidate := *cfg
		if !applyConfigValue(&candidate, key, value) {
			continue
		}
		*cfg = candidate
	}
	return nil
}

func configValueString(key string, raw any) (string, bool) {
	switch key {
	case "say.source", "profile.source", "say.target", "profile.target",
		"say.mode", "profile.mode", "say.style", "profile.style",
		"say.polite", "profile.polite", "say.profanity", "profile.profanity",
		"say.llm.provider", "profile.provider", "say.llm.model", "profile.model",
		"say.llm.timeout", "profile.timeout":
		value, ok := raw.(string)
		return value, ok
	case "say.simple", "say.simplify", "profile.simple", "profile.simplify", "say.preserve_technical_terms":
		value, ok := raw.(bool)
		if !ok {
			return "", false
		}
		return strconv.FormatBool(value), true
	case "say.llm.timeout_seconds", "profile.timeout_seconds":
		value, ok := raw.(int64)
		if !ok {
			return "", false
		}
		return strconv.FormatInt(value, 10), true
	default:
		return "", false
	}
}

// applyConfigValue ignores invalid or unknown settings, preserving the value
// already supplied by a lower-precedence configuration layer.
func applyConfigValue(cfg *Config, key, value string) bool {
	switch key {
	case "say.source", "profile.source":
		if !oneOf(value, "zh", "ja", "en", "auto") {
			return false
		}
		cfg.Source = value
	case "say.target", "profile.target":
		if !oneOf(value, "zh", "ja", "en") {
			return false
		}
		cfg.Target = value
	case "say.mode", "profile.mode":
		if !oneOf(value, "realtime", "written") {
			return false
		}
		cfg.Mode = value
	case "say.style", "profile.style":
		if !oneOf(value, "spoken", "teams", "letter", "document", "article") {
			return false
		}
		cfg.Style = value
	case "say.polite", "profile.polite":
		if !oneOf(value, "casual", "polite", "formal") {
			return false
		}
		cfg.Polite = value
	case "say.simple", "say.simplify", "profile.simple", "profile.simplify":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return false
		}
		cfg.Simple = parsed
	case "say.profanity", "profile.profanity":
		if !oneOf(value, "mask", "soften", "preserve") {
			return false
		}
		cfg.Profanity = value
	case "say.llm.provider", "profile.provider":
		if value != "openai" {
			return false
		}
		cfg.Provider = value
	case "say.llm.model", "profile.model":
		if strings.TrimSpace(value) == "" {
			return false
		}
		cfg.Model = value
	case "say.llm.timeout", "profile.timeout":
		if !validTimeout(value) {
			return false
		}
		cfg.Timeout = value
	case "say.llm.timeout_seconds", "profile.timeout_seconds":
		seconds, err := strconv.ParseInt(value, 10, 64)
		if err != nil || seconds <= 0 || seconds > 600 {
			return false
		}
		cfg.Timeout = fmt.Sprintf("%ds", seconds)
	case "say.preserve_technical_terms":
		return value == "true" || value == "false"
	default:
		return false
	}
	return true
}
