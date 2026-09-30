package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFeatureDesignReadinessRequiresExplicitDecision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feature.md")
	for _, test := range []struct{ content, want string; valid bool }{
		{"## Design Readiness\n\nBLOCKED\n\n## Work Items\n- [ ] 1.1 Work\n", "BLOCKED", true},
		{"## Design Readiness\n\nFD_APPLIED\n\n## Work Items\n- [ ] 1.1 Work\n", "FD_APPLIED", true},
		{"## Work Items\n- [ ] 1.1 Work\n", "", false},
	} {
		if err := os.WriteFile(path, []byte(test.content), 0o644); err != nil { t.Fatal(err) }
		got, err := ReadFeatureDesignReadiness(path)
		if (err == nil) != test.valid || got != test.want { t.Fatalf("readiness=%q err=%v, want %q valid=%t", got, err, test.want, test.valid) }
	}
}
