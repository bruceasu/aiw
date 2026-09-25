package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aiw/internal/requirement"
	"aiw/internal/session"
)

func TestRequirementRegressionCaptureCheckpointChanges(t *testing.T) {
	for _, change := range []string{"action", "revision", "no-display"} {
		t.Run(change, func(t *testing.T) {
			previous, err := os.Getwd()
			if err != nil { t.Fatal(err) }
			dir := t.TempDir()
			if err := os.Chdir(dir); err != nil { t.Fatal(err) }
			t.Cleanup(func() { _ = os.Chdir(previous) })
			store := session.NewStore(filepath.Join(dir, ".ai"))
			if _, err := store.Create("regression", "Regression", dir, "codex", "", "instructions"); err != nil { t.Fatal(err) }
			if _, err := requirement.Create("notify", "Notify"); err != nil { t.Fatal(err) }
			if err := os.WriteFile("draft.md", []byte("Task completion only"), 0o644); err != nil { t.Fatal(err) }
			action := requirementPendingAction{Kind: "capture", RequirementID: "notify", Artifact: "problem-brief", Source: "draft.md", Facts: []string{"Task completion only"}}
			save := func() {
				t.Helper()
				b, err := json.Marshal(action)
				if err != nil { t.Fatal(err) }
				if err := store.WriteArtifact("regression", "pending-requirement-action.json", b); err != nil { t.Fatal(err) }
			}
			save()
			if change != "no-display" {
				if _, err := displayRequirementCheckpoint(store, requirementChatPlan{SessionID: "regression", RequirementID: "notify"}); err != nil { t.Fatal(err) }
			}
			switch change {
			case "action":
				action.Facts = nil
				save()
			case "revision":
				if _, _, err := requirement.Capture("notify", "business-case", "draft.md"); err != nil { t.Fatal(err) }
			}
			before, err := requirement.Read("notify")
			if err != nil { t.Fatal(err) }
			if _, err := confirmRequirementAction(store, "regression"); err == nil { t.Fatal("changed or absent checkpoint was accepted") }
			after, err := requirement.Read("notify")
			if err != nil || after.Revision != before.Revision || after.Status != before.Status { t.Fatal("rejected confirmation changed Requirement") }
			if _, ok := after.Artifacts["problem-brief"]; ok { t.Fatal("rejected action captured an artifact") }
			if _, err := store.ReadArtifact("regression", "confirmed-requirement-facts.json"); !os.IsNotExist(err) { t.Fatal("rejected action wrote confirmation") }
		})
	}
}
