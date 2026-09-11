package task

import (
	"strings"
	"testing"
)

func TestCreationPreflightClassifiesTargetAndUnrelatedPaths(t *testing.T) {
	result := classifyCreationDirtyPaths(
		[]string{"openspec/changes/new-task/tasks.md", "README.md"},
		[]string{"openspec/changes/new-task"},
		taskCreationSharedWritePaths(),
	)
	if len(result.TargetOverlaps) != 1 || len(result.SharedWriteOverlaps) != 0 || len(result.UnrelatedDirtyPaths) != 1 {
		t.Fatalf("unexpected classification: %#v", result)
	}
	if err := result.authorize(true); err == nil || !strings.Contains(err.Error(), "target paths") {
		t.Fatalf("target overlap must block creation: %v", err)
	}
}

func TestCreationPreflightRequiresExplicitConfirmationForUnrelatedPaths(t *testing.T) {
	result := creationPreflight{UnrelatedDirtyPaths: []string{"README.md"}}
	if err := result.authorize(false); err == nil || !strings.Contains(err.Error(), "--allow-unrelated-dirty") {
		t.Fatalf("missing confirmation must block creation: %v", err)
	}
	if err := result.authorize(true); err != nil {
		t.Fatalf("explicit confirmation should allow unrelated paths: %v", err)
	}
}
