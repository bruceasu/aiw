package taskx

import "testing"

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
