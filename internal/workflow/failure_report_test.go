package workflow

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCompilerFailurePersistsImmutableAndReadableLatestReport(t *testing.T) {
	store := NewStore(t.TempDir())
	state := compatibleState()
	state.WorkItems = []WorkItem{{ID: "wi-0001", State: WorkItemReady}}
	if _, err := store.Create(state); err != nil {
		t.Fatal(err)
	}
	diagnostics := ActorReference{Kind: "compile-diagnostics", Path: "compile-diagnostics/wi-0001.json"}
	updated, err := store.RecordCompilerOutcome("task-1", "wi-0001", diagnostics, false)
	if err != nil {
		t.Fatal(err)
	}
	if updated.LastEventSequence != 1 {
		t.Fatalf("unexpected event sequence: %d", updated.LastEventSequence)
	}

	content, err := store.ReadLatestFailureReport("task-1")
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)
	for _, expected := range []string{"Category: `compiler`", "Detail:", "Retryable: `true`", "Owner: `supervisor`", "Next action:", "wi-0001", "compile-diagnostics/wi-0001.json"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("latest report missing %q:\n%s", expected, report)
		}
	}
	entries, err := os.ReadDir(store.path("task-1", "reports", "attempts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".json") {
		t.Fatalf("immutable reports = %#v", entries)
	}
	before, err := os.ReadFile(store.path("task-1", runtimeStateFile))
	if err != nil { t.Fatal(err) }
	if _, err := store.ReadLatestFailureReport("task-1"); err != nil { t.Fatal(err) }
	after, err := os.ReadFile(store.path("task-1", runtimeStateFile))
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(before, after) { t.Fatal("report reader changed persisted runtime") }
	loaded, err := store.Load("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LastEventSequence != updated.LastEventSequence {
		t.Fatalf("read-only report changed runtime state: %d -> %d", updated.LastEventSequence, loaded.LastEventSequence)
	}
}
