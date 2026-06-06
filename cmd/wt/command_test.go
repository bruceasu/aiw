package wt

import "testing"

func TestParseRemoveOptionsRejectsUnknownFlag(t *testing.T) {
	_, err := parseRemoveOptions([]string{"--wat"})
	if err == nil {
		t.Fatal("expected unknown flag to be rejected")
	}
}

func TestParseListOptionsSupportsPorcelain(t *testing.T) {
	opts, err := parseListOptions([]string{"--porcelain"})
	if err != nil {
		t.Fatalf("parse list options: %v", err)
	}
	if !opts.Porcelain {
		t.Fatal("expected porcelain mode to be enabled")
	}
}

func TestParsePruneOptionsSupportsDryRun(t *testing.T) {
	opts, err := parsePruneOptions([]string{"--dry-run"})
	if err != nil {
		t.Fatalf("parse prune options: %v", err)
	}
	if !opts.DryRun {
		t.Fatal("expected dry-run mode to be enabled")
	}
}
