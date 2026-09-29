package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FeatureDesignArchivePath keeps an authored FD with its archived Task.
func FeatureDesignArchivePath(archiveName string) string {
	return filepath.Join(FeatureDesignDir, "archive", archiveName+".md")
}

// ReadFeatureDesignReadiness reads the explicit design decision without
// inferring readiness from the presence of numbered work items.
func ReadFeatureDesignReadiness(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	inReadiness := false
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			if inReadiness {
				break
			}
			inReadiness = strings.EqualFold(line, "## Design Readiness")
			continue
		}
		if !inReadiness {
			continue
		}
		switch line {
		case "FD_APPLIED", "FD_NOT_REQUIRED", "BLOCKED":
			return line, nil
		}
	}
	return "", fmt.Errorf("Feature Design %s has no valid Design Readiness", path)
}
