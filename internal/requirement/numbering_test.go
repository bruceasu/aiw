package requirement

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequirementNumberingSequentialAndLegacy(t *testing.T) {
	inTempDir(t)
	legacy, err := Create("legacy-report", "Legacy")
	if err != nil || legacy.ID != "legacy-report" { t.Fatalf("legacy: %#v %v", legacy, err) }
	for i, slug := range []string{"add-chat-support", "archive", "req00000-example"} {
		meta, err := CreateNumbered(slug, "Title")
		want := fmt.Sprintf("REQ%05d-%s", i+1, slug)
		if err != nil || meta.ID != want { t.Fatalf("create: %#v %v; want %s", meta, err, want) }
		stored, err := Read(meta.ID)
		if err != nil || stored.ID != want || stored.Title != "Title" { t.Fatalf("read: %#v %v", stored, err) }
	}
	if _, err := Create("legacy-report", "Other"); err == nil { t.Fatal("duplicate legacy ID was accepted") }
}

func TestRequirementNumberingTerminalAndMissingCounter(t *testing.T) {
	inTempDir(t)
	first, err := CreateNumbered("archived", "Archived")
	if err != nil { t.Fatal(err) }
	if _, _, err := Capture(first.ID, "requirement-plan", writeSource(t, "plan")); err != nil { t.Fatal(err) }
	if _, err := Archive(first.ID, "owner", "done"); err != nil { t.Fatal(err) }
	second, err := CreateNumbered("cancelled", "Cancelled")
	if err != nil { t.Fatal(err) }
	if _, err := Cancel(second.ID, "owner", "cancel"); err != nil { t.Fatal(err) }
	// Simulate importing only the durable terminal artifacts into a new checkout.
	if err := os.Remove(filepath.Join(".ai", "requirements", "sequence")); err != nil { t.Fatal(err) }
	next, err := CreateNumbered("next", "Next")
	if err != nil || next.ID != "REQ00003-next" { t.Fatalf("terminal number reused: %#v %v", next, err) }
}

func TestRequirementNumberingExplicitHighWaterAndWidth(t *testing.T) {
	inTempDir(t)
	if _, err := Create("REQ99999-imported", "Imported"); err != nil { t.Fatal(err) }
	next, err := CreateNumbered("next", "Next")
	if err != nil || next.ID != "REQ100000-next" { t.Fatalf("width: %#v %v", next, err) }
	if _, err := Create("REQ99999-other", "Other"); err == nil { t.Fatal("number reused with a different slug") }
	if _, err := Create("req100000-other", "Other"); err == nil { t.Fatal("number reused with different case") }
}

func TestRequirementNumberingCounterFailures(t *testing.T) {
	for _, content := range []string{"18446744073709551615\n", "18446744073709551616\n"} {
		t.Run(fmt.Sprintf("counter-%q", content), func(t *testing.T) {
			inTempDir(t)
			if err := os.MkdirAll(filepath.Join(".ai", "requirements"), 0o755); err != nil { t.Fatal(err) }
			path := filepath.Join(".ai", "requirements", "sequence")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil { t.Fatal(err) }
			if _, err := CreateNumbered("next", "Next"); err == nil { t.Fatal("invalid or exhausted counter was accepted") }
			got, err := os.ReadFile(path)
			if err != nil || string(got) != content { t.Fatalf("counter changed: %q %v", got, err) }
			if _, err := os.Stat(Root); !os.IsNotExist(err) { t.Fatalf("failed allocation created artifacts: %v", err) }
		})
	}
}

func TestRequirementNumberingRecovery(t *testing.T) {
	for _, content := range []string{"", "broken\n", "2", "2\n1\n", "0\n"} {
		for _, maximum := range []int{0, 12} {
			t.Run(fmt.Sprintf("%q-%d", content, maximum), func(t *testing.T) {
				inTempDir(t)
				if _, err := Create("legacy", "Legacy"); err != nil { t.Fatal(err) }
				if maximum != 0 {
					if err := os.MkdirAll(filepath.Join(Root, archiveRoot, "REQ00012-imported"), 0o755); err != nil { t.Fatal(err) }
				}
				path := filepath.Join(".ai", "requirements", "sequence")
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil { t.Fatal(err) }
				meta, err := CreateNumbered("next", "Next")
				if err != nil || meta.ID != fmt.Sprintf("REQ%05d-next", maximum+1) { t.Fatalf("recovery: %#v %v", meta, err) }
				backups, err := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.bak"))
				if err != nil || len(backups) != 1 { t.Fatalf("backups: %v %v", backups, err) }
				original, err := os.ReadFile(backups[0])
				if err != nil || string(original) != content { t.Fatalf("backup changed: %q %v", original, err) }
				next, err := CreateNumbered("after", "After")
				if err != nil || next.ID != fmt.Sprintf("REQ%05d-after", maximum+2) { t.Fatalf("after recovery: %#v %v", next, err) }
			})
		}
	}
}

func TestRequirementNumberingRecoveryStopsOnIOFailure(t *testing.T) {
	for _, failure := range []string{"backup", "scan", "scan-parent", "scan-archive", "scan-cancelled", "counter-directory"} {
		t.Run(failure, func(t *testing.T) {
			inTempDir(t)
			if err := os.MkdirAll(filepath.Join(".ai", "requirements"), 0o755); err != nil { t.Fatal(err) }
			path := filepath.Join(".ai", "requirements", "sequence")
			if failure == "counter-directory" {
				if err := os.Mkdir(path, 0o700); err != nil { t.Fatal(err) }
			} else {
				if err := os.WriteFile(path, []byte("broken"), 0o600); err != nil { t.Fatal(err) }
				blocked := filepath.Join(filepath.Dir(path), "backups")
				if failure == "scan" { blocked = Root }
				if failure == "scan-parent" { blocked = filepath.Dir(Root) }
				if failure == "scan-archive" { blocked = filepath.Join(Root, archiveRoot) }
				if failure == "scan-cancelled" { blocked = filepath.Join(Root, cancelledRoot) }
				if err := os.MkdirAll(filepath.Dir(blocked), 0o755); err != nil { t.Fatal(err) }
				if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil { t.Fatal(err) }
			}
			if _, err := CreateNumbered("next", "Next"); err == nil { t.Fatal("I/O failure was ignored") }
			if strings.HasPrefix(failure, "scan") {
				if _, err := os.Stat(filepath.Join(filepath.Dir(path), "backups")); !os.IsNotExist(err) { t.Fatalf("scan failure reached recovery: %v", err) }
			}
			if failure != "counter-directory" {
				content, err := os.ReadFile(path)
				if err != nil || string(content) != "broken" { t.Fatalf("counter changed: %q %v", content, err) }
			}
		})
	}
}

func TestRequirementNumberingReservedGapAndBusyLock(t *testing.T) {
	inTempDir(t)
	if err := os.MkdirAll(filepath.Join(".ai", "requirements"), 0o755); err != nil { t.Fatal(err) }
	sequence := filepath.Join(".ai", "requirements", "sequence")
	if err := os.WriteFile(sequence, []byte("1\n7\n"), 0o600); err != nil { t.Fatal(err) }
	lock := sequence + ".lock"
	if err := os.WriteFile(lock, []byte("owned"), 0o600); err != nil { t.Fatal(err) }
	if _, err := CreateNumbered("next", "Next"); err == nil { t.Fatal("creation bypassed another writer") }
	if content, err := os.ReadFile(lock); err != nil || string(content) != "owned" { t.Fatalf("another writer's lock was changed: %q %v", content, err) }
	if err := os.Remove(lock); err != nil { t.Fatal(err) }
	next, err := CreateNumbered("next", "Next")
	if err != nil || next.ID != "REQ00008-next" { t.Fatalf("reserved gap reused: %#v %v", next, err) }
}

func TestRequirementNumberingInvalidSlugDoesNotAllocate(t *testing.T) {
	inTempDir(t)
	for _, slug := range []string{"", "../escape", "Upper", "two words", "-start", "end-", "two--parts"} {
		if _, err := CreateNumbered(slug, "Invalid"); err == nil { t.Fatalf("accepted invalid slug %q", slug) }
	}
	if _, err := os.Stat(".ai"); !os.IsNotExist(err) { t.Fatalf("invalid slug allocated state: %v", err) }
}

func TestRequirementNumberingConcurrentCreators(t *testing.T) {
	inTempDir(t)
	type result struct { meta Meta; err error }
	results := make(chan result, 8)
	for i := 0; i < cap(results); i++ {
		go func() { meta, err := CreateNumbered("same-slug", "Concurrent"); results <- result{meta, err} }()
	}
	seen := make(map[string]bool)
	for i := 0; i < cap(results); i++ {
		r := <-results
		if r.err != nil {
			if !strings.Contains(r.err.Error(), "creation lock") { t.Errorf("unexpected failure: %v", r.err) }
			continue
		}
		if seen[r.meta.ID] { t.Errorf("duplicate ID: %s", r.meta.ID) }
		seen[r.meta.ID] = true
	}
	if len(seen) == 0 { t.Fatal("no creator succeeded") }
	next, err := CreateNumbered("after", "After")
	if err != nil || next.ID != fmt.Sprintf("REQ%05d-after", len(seen)+1) { t.Fatalf("lost allocation: %#v %v", next, err) }
}
