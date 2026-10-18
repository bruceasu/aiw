package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LoadExecutableDefaults imports only the executable directory's .env. Existing
// process variables, including empty values, take priority over file defaults.
func LoadExecutableDefaults() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable for environment defaults: %w", err)
	}
	filename := filepath.Join(filepath.Dir(executable), ".env")
	loader := New()
	loader.defaultsOnly = true
	if err := loader.ParseFile(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load installed environment %s: %w", filename, err)
	}
	for key, value := range loader.Env {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set environment variable %q from %s: %w", key, filename, err)
		}
	}
	return nil
}
