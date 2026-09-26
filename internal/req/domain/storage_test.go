package requirement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequirementStorageCanonicalLifecycle(t *testing.T) {
	inTempDir(t)
	meta, err := CreateNumbered("layout", "Layout")
	if err != nil { t.Fatal(err) }
	if _, err := os.Stat(filepath.Join("docs", "requirements", meta.ID, "requirement.toml")); err != nil { t.Fatal(err) }
	if _, err := os.Stat(filepath.Join(".ai", "requirements", "sequence")); err != nil { t.Fatal(err) }
	if _, _, err := Capture(meta.ID, "requirement-plan", writeSource(t, "Plan")); err != nil { t.Fatal(err) }
	if _, err := Approve(meta.ID, "APPROVED", "owner", "accepted"); err != nil { t.Fatal(err) }
	if _, err := Archive(meta.ID, "owner", "done"); err != nil { t.Fatal(err) }
	stored, err := Read(meta.ID)
	if err != nil || stored.Status != "ARCHIVED" || stored.Artifacts["requirement-plan"].Path != "docs/requirements/archive/"+meta.ID+"/requirement-plan.md" { t.Fatalf("archive: %#v %v", stored, err) }
	archived, err := List(ListArchived)
	if err != nil || len(archived) != 1 { t.Fatalf("archived list: %v %v", archived, err) }
	next, err := CreateNumbered("cancel", "Cancel")
	if err != nil { t.Fatal(err) }
	if _, err := Cancel(next.ID, "owner", "stop"); err != nil { t.Fatal(err) }
	if _, err := os.Stat(filepath.Join("docs", "requirements", "cancelled", next.ID, "requirement.toml")); err != nil { t.Fatal(err) }
	for _, old := range []string{"requirements", ".ai/requirement-sequence"} {
		if _, err := os.Stat(old); !os.IsNotExist(err) { t.Fatalf("old path created: %s %v", old, err) }
	}
	temporary, err := os.ReadDir(filepath.Join(".ai", "requirements", "temporary"))
	if err != nil || len(temporary) != 0 { t.Fatalf("temporary files leaked: %v %v", temporary, err) }
}

func TestRequirementStorageNoLegacyFallback(t *testing.T) {
	inTempDir(t)
	legacy := filepath.Join("requirements", "REQ00999-old")
	if err := os.MkdirAll(legacy, 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(legacy, "requirement.toml"), []byte("id = \"REQ00999-old\"\n"), 0o600); err != nil { t.Fatal(err) }
	if _, err := Read("REQ00999-old"); err == nil { t.Fatal("read fell back to old root") }
	listed, err := List(ListAll)
	if err != nil || len(listed) != 0 { t.Fatalf("old records listed: %v %v", listed, err) }
	meta, err := CreateNumbered("first", "First")
	if err != nil || meta.ID != "REQ00001-first" { t.Fatalf("old root scanned: %#v %v", meta, err) }
}

func TestRequirementStorageMovedSourcesRequireReview(t *testing.T) {
	inTempDir(t)
	if _, err := Create("moved", "Moved"); err != nil { t.Fatal(err) }
	if _, _, err := Capture("moved", "problem-brief", writeSource(t, "Keep identity")); err != nil { t.Fatal(err) }
	path := filepath.Join(Dir("moved"), "requirement.toml")
	data, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	// Model an unchanged metadata file moved from the former root.
	old := strings.ReplaceAll(string(data), "docs/requirements/", "requirements/")
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil { t.Fatal(err) }
	meta, err := Read("moved")
	if err != nil || meta.Artifacts["problem-brief"].Path != "docs/requirements/moved/problem-brief.md" { t.Fatalf("effective path: %#v %v", meta, err) }
	snapshot, err := LoadConversationContext("moved", "Continue", nil)
	if err != nil { t.Fatal(err) }
	if snapshot.Sources[0].Path != meta.Artifacts["problem-brief"].Path { t.Fatal("context uses historical path") }
	for i := range snapshot.Sources { snapshot.Sources[i].Path = strings.TrimPrefix(snapshot.Sources[i].Path, "docs/") }
	record := ConversationEvidence{Version: 1, RequirementID: "moved", State: "completed", Turns: []int{1, 2}, Turn: ConversationTurn{Context: snapshot}}
	raw, err := json.Marshal(record)
	if err != nil { t.Fatal(err) }
	if candidate, notice := RestoreConversationCandidate(string(raw), "moved", ConfirmedFacts{}); candidate != nil || !strings.Contains(notice, "stale") { t.Fatalf("old candidate accepted: %v %s", candidate, notice) }
	after, err := os.ReadFile(path)
	if err != nil || string(after) != old { t.Fatal("read rewrote historical metadata") }
}
