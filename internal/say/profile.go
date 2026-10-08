package say

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var safeProfileName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// LoadProfile overlays a named user profile on the supplied base settings.
func LoadProfile(cfg Config, name string) (Config, error) {
	if name == "" {
		return cfg, nil
	}
	if !safeProfileName.MatchString(name) || strings.Contains(name, "..") {
		return Config{}, errors.New("profile name may contain only letters, numbers, underscore, and hyphen")
	}
	root, err := profileRoot()
	if err != nil {
		return Config{}, err
	}
	path := filepath.Join(root, "aiw", "profiles", name+".toml")
	values, exists, err := readConfig(path)
	if err != nil {
		return Config{}, fmt.Errorf("read profile %s: %w", name, err)
	}
	if !exists {
		return Config{}, fmt.Errorf("profile does not exist: %s", name)
	}
	if err := applyConfig(&cfg, values); err != nil {
		return Config{}, fmt.Errorf("profile %s: %w", name, err)
	}
	return cfg, nil
}

func profileRoot() (string, error) {
	if runtime.GOOS == "windows" {
		if value := os.Getenv("APPDATA"); value != "" {
			return value, nil
		}
		return "", errors.New("APPDATA is not set")
	}
	if value := os.Getenv("XDG_CONFIG_HOME"); value != "" {
		return value, nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("resolve user profile directory")
	}
	return filepath.Join(home, ".config"), nil
}

// Validate checks values after base, profile, and CLI overrides are merged.
func Validate(cfg Config) (time.Duration, error) {
	if !oneOf(cfg.Source, "zh", "ja", "en", "auto") {
		return 0, fmt.Errorf("invalid source language %q", cfg.Source)
	}
	if !oneOf(cfg.Target, "zh", "ja", "en") {
		return 0, fmt.Errorf("invalid target language %q", cfg.Target)
	}
	if !oneOf(cfg.Mode, "realtime", "written") {
		return 0, fmt.Errorf("invalid mode %q", cfg.Mode)
	}
	if !oneOf(cfg.Style, "spoken", "teams", "letter", "document", "article") {
		return 0, fmt.Errorf("invalid style %q", cfg.Style)
	}
	if !oneOf(cfg.Polite, "casual", "polite", "formal") {
		return 0, fmt.Errorf("invalid politeness %q", cfg.Polite)
	}
	if !oneOf(cfg.Profanity, "mask", "soften", "preserve") {
		return 0, fmt.Errorf("invalid profanity mode %q", cfg.Profanity)
	}
	if cfg.Provider != "openai" {
		return 0, fmt.Errorf("unsupported provider %q", cfg.Provider)
	}
	if !validTimeout(cfg.Timeout) {
		return 0, errors.New("timeout must be a duration greater than zero and at most 10m")
	}
	duration, _ := time.ParseDuration(cfg.Timeout)
	return duration, nil
}

func validTimeout(value string) bool {
	duration, err := time.ParseDuration(value)
	return err == nil && duration > 0 && duration <= 10*time.Minute
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item { return true }
	}
	return false
}
