// Package taskpath defines the on-disk locations for active Task records.
package taskpath

import (
	"errors"
	"path/filepath"
	"strings"
)

// ActiveTaskDir returns the only supported active Task directory. root is the
// AIW runtime root (normally .ai), not the repository root.
func ActiveTaskDir(root, id string) string {
	return filepath.Join(root, "tasks", id)
}

// LegacyTaskDir returns the former active Task directory for diagnostics and
// explicit migration only.
func LegacyTaskDir(root, id string) string {
	return filepath.Join(root, id)
}

// ValidateID prevents a Task ID from escaping either storage root.
func ValidateID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\\`) {
		return errors.New("invalid Task ID")
	}
	return nil
}
