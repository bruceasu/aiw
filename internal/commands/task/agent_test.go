package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteLineageIsReadable(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.MkdirAll(filepath.Join("openspec", "changes", "T-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := agentLineage{TaskID: "T-1", SessionID: "S-1", ParentThread: "p", ChildThread: "c", ParentState: "handed-off", ChildState: "completed", Status: "completed"}
	if err := writeLineage("T-1", want); err != nil {
		t.Fatal(err)
	}
	b, err := readLineage("T-1")
	if err != nil {
		t.Fatal(err)
	}
	var got agentLineage
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.TaskID != want.TaskID || got.ChildThread != want.ChildThread || got.ParentState != want.ParentState || got.ChildState != want.ChildState || got.Status != want.Status {
		t.Fatalf("unexpected lineage: %+v", got)
	}
}

func TestParseAgentOptions(t *testing.T) {
	opts, err := parseAgentOptions([]string{"--handoff", "handoff.md", "--takeover", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Handoff != "handoff.md" || !opts.Takeover || !opts.Yes {
		t.Fatalf("unexpected options: %+v", opts)
	}
}

func TestNormalizeIDIsDeterministic(t *testing.T) {
	if got := normalizeID("Feature/Task 42"); got != "feature-task-42" {
		t.Fatalf("unexpected normalized ID: %q", got)
	}
}
