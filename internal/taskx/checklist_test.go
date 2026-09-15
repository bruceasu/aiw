package taskx

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseNumberedChecklistReadsAuthoredItems(t *testing.T) {
	content := "# Plan\n\n- [ ] 1.1 first item\n- [x] 1.2 second item\n"
	items, diagnostics := ParseNumberedChecklist(content)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Number != "1.1" || items[0].Title != "first item" || items[0].Completed {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	if items[1].Number != "1.2" || !items[1].Completed {
		t.Fatalf("unexpected second item: %#v", items[1])
	}
}

func TestParseNumberedChecklistReadsExplicitDependencies(t *testing.T) {
	items, diagnostics := ParseNumberedChecklist("- [ ] 5.2 run after compiler <!-- aiw:depends-on=3.2, 4.1 -->\n")
	if len(diagnostics) != 0 || len(items) != 1 {
		t.Fatalf("items=%#v diagnostics=%#v", items, diagnostics)
	}
	if items[0].Title != "run after compiler" || len(items[0].DependsOn) != 2 || items[0].DependsOn[0] != "3.2" || items[0].DependsOn[1] != "4.1" {
		t.Fatalf("parsed item=%#v", items[0])
	}
}

func TestParseNumberedChecklistReportsDuplicateAndUnsupportedItems(t *testing.T) {
	items, diagnostics := ParseNumberedChecklist("- [ ] 1.1 first\n- [ ] 1.1 duplicate\n- [ ] 1 unsupported\n")
	if len(items) != 1 || items[0].Number != "1.1" {
		t.Fatalf("unexpected items: %#v", items)
	}
	if len(diagnostics) != 2 {
		t.Fatalf("got diagnostics %#v, want duplicate and unsupported", diagnostics)
	}
	if diagnostics[0].Code != "duplicate-checklist-number" || diagnostics[1].Code != "unsupported-numbered-checklist" {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestProjectWorkflowChecklistCompletionChangesOnlyMappedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.md")
	before := "# Tasks\n\n- [ ] 1.1 keep open\n- [ ] 1.2 complete this\n- [x] 1.3 already done\n"
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ProjectWorkflowChecklistCompletion(path, "1.2"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Tasks\n\n- [ ] 1.1 keep open\n- [x] 1.2 complete this\n- [x] 1.3 already done\n"
	if string(after) != want {
		t.Fatalf("projected checklist = %q, want %q", after, want)
	}
}

func TestProjectWorkflowChecklistCompletionRejectsAmbiguousOrDeletedItem(t *testing.T) {
	for name, content, number, code := range map[string]struct {
		content string
		number  string
		code    string
	}{
		"duplicate": {"- [ ] 1.1 first\n- [ ] 1.1 duplicate\n", "1.1", "duplicate-checklist-number"},
		"deleted":   {"- [ ] 1.1 present\n", "1.2", "deleted-checklist-identifier"},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tasks.md")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			err := ProjectWorkflowChecklistCompletion(path, number)
			var projectionErr *ChecklistProjectionError
			if !errors.As(err, &projectionErr) || projectionErr.Code != code {
				t.Fatalf("projection error = %#v, want code %q", err, code)
			}
			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if strings.TrimSpace(string(after)) != strings.TrimSpace(content) {
				t.Fatalf("rejected projection changed checklist: %q", after)
			}
		})
	}
}
