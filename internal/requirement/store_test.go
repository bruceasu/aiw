package requirement

import (
	"os"
	"path/filepath"
	"testing"
)

func inTempDir(t *testing.T) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(dir); err != nil { t.Fatal(err) }
}

func TestCreateCaptureApproveAndSnapshot(t *testing.T) {
	inTempDir(t)
	if _, err := Create("daily-report", "Daily report"); err != nil { t.Fatal(err) }
	source := filepath.Join(t.TempDir(), "plan.md")
	if err := os.WriteFile(source, []byte("# Requirement Plan\n"), 0o644); err != nil { t.Fatal(err) }
	meta, artifact, err := Capture("daily-report", "requirement-plan", source)
	if err != nil { t.Fatal(err) }
	if meta.Status != "DECIDED" || artifact.Digest == "" { t.Fatalf("unexpected capture: %#v %#v", meta, artifact) }
	meta, err = Approve("daily-report", "APPROVED", "owner", "ready")
	if err != nil { t.Fatal(err) }
	if meta.Status != "APPROVED" || meta.Approval.By != "owner" { t.Fatalf("unexpected approval: %#v", meta) }
	if _, created, err := StartPromotion("daily-report", "daily-report-task"); err != nil || !created { t.Fatalf("unexpected promotion start: created=%v err=%v", created, err) }
	if _, created, err := StartPromotion("daily-report", "daily-report-task"); err != nil || created { t.Fatalf("promotion should be idempotent: created=%v err=%v", created, err) }
	meta, err = CompletePromotion("daily-report", "daily-report-task")
	if err != nil || meta.Promotion.Status != "SPEC_DRAFTED" { t.Fatalf("unexpected promotion completion: %#v %v", meta, err) }
	snapshot, err := ArtifactSnapshot("daily-report")
	if err != nil || len(snapshot) != 1 || snapshot[0].Kind != "requirement-plan" { t.Fatalf("unexpected snapshot: %#v %v", snapshot, err) }
	stored, err := Read("daily-report")
	if err != nil || stored.Artifacts["requirement-plan"].Digest != artifact.Digest { t.Fatalf("artifact metadata was not persisted: %#v %v", stored, err) }
}

func TestArtifactSnapshotRejectsPostCaptureChanges(t *testing.T) {
	inTempDir(t)
	if _, err := Create("changed", "Changed"); err != nil { t.Fatal(err) }
	source := filepath.Join(t.TempDir(), "plan.md")
	if err := os.WriteFile(source, []byte("first"), 0o644); err != nil { t.Fatal(err) }
	if _, _, err := Capture("changed", "requirement-plan", source); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(Dir("changed"), "requirement-plan.md"), []byte("second"), 0o644); err != nil { t.Fatal(err) }
	if _, err := ArtifactSnapshot("changed"); err == nil { t.Fatal("expected changed artifact rejection") }
}

func TestCreateRejectsDuplicateAndMalformedMetadata(t *testing.T) {
	inTempDir(t)
	if _, err := Create("duplicate", "Duplicate"); err != nil { t.Fatal(err) }
	if _, err := Create("duplicate", "Duplicate"); err == nil { t.Fatal("expected duplicate error") }
	if err := os.MkdirAll(Dir("malformed"), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(Dir("malformed"), "requirement.toml"), []byte("id = \"malformed\"\n"), 0o644); err != nil { t.Fatal(err) }
	if _, err := Read("malformed"); err == nil { t.Fatal("expected malformed metadata error") }
}

func TestArchiveAndCancelPreserveRecordsAndListFilters(t *testing.T) {
	inTempDir(t)
	if _, err := Create("ready", "Ready"); err != nil { t.Fatal(err) }
	if _, _, err := Capture("ready", "requirement-plan", writeSource(t, "ready")); err != nil { t.Fatal(err) }
	if _, err := Archive("ready", "owner", "completed"); err != nil { t.Fatal(err) }
	meta, err := Read("ready"); if err != nil || meta.Status != "ARCHIVED" { t.Fatalf("archived read failed: %#v %v", meta, err) }
	if _, err := os.Stat(filepath.Join(Root, "archive", "ready", "requirement-plan.md")); err != nil { t.Fatal(err) }
	if _, err := Create("stopped", "Stopped"); err != nil { t.Fatal(err) }
	if _, err := Cancel("stopped", "owner", "cancelled"); err != nil { t.Fatal(err) }
	active, err := List(ListActive); if err != nil || len(active) != 0 { t.Fatalf("unexpected active list: %#v %v", active, err) }
	archived, err := List(ListArchived); if err != nil || len(archived) != 1 || archived[0].ID != "ready" { t.Fatalf("unexpected archive list: %#v %v", archived, err) }
	cancelled, err := List(ListCancelled); if err != nil || len(cancelled) != 1 || cancelled[0].ID != "stopped" { t.Fatalf("unexpected cancelled list: %#v %v", cancelled, err) }
}

func writeSource(t *testing.T, name string) string {
	t.Helper(); source := filepath.Join(t.TempDir(), name+".md"); if err := os.WriteFile(source, []byte(name), 0o644); err != nil { t.Fatal(err) }; return source
}
