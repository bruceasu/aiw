//go:build windows

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func knowledgeTestImport(t *testing.T, store *Store, id TaskID, root, body string, priority int) KnowledgeVersion {
	t.Helper()
	original := "Human source: " + body
	version, err := store.ImportHumanKnowledge(id, InputSource{
		Path: "human-note", Status: "loaded", Content: original, SHA256: contentDigest([]byte(original)),
	}, KnowledgeContent{
		Body: body, Scope: []string{"module:memory"}, Usage: "Check current requirements first.",
		InvalidWhen: "The bound requirement changes.", Priority: priority,
		Checks: []InputSource{{Path: "requirement.txt", SHA256: contentDigest([]byte("fixed requirement"))}},
	})
	if err != nil { t.Fatal(err) }
	var source InputSource
	if err := store.ReadExecutionArtifact(id, version.Origin.Acceptance, &source); err != nil { t.Fatal(err) }
	if source.Content != original || version.State != "candidate" || !strings.Contains(version.Content.EvidenceNature, "identity unverified") {
		t.Fatal("human original was altered or imported as an approval")
	}
	return version
}

func knowledgeTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "requirement.txt"), []byte("fixed requirement"), 0600); err != nil { t.Fatal(err) }
	return root
}

func knowledgeTestReview(t *testing.T, store *Store, id TaskID, root string, version KnowledgeVersion, action string) KnowledgeVersion {
	t.Helper()
	result, err := store.ReviewKnowledge(id, root, KnowledgeReviewRequest{
		Version: version.Version, ExpectedRevision: version.Revision, Action: action, Reason: "Explicit fixture decision.",
	}, KnowledgeReviewer{Actor: "fixture operator", IdentityVerified: false})
	if err != nil { t.Fatal(err) }
	return result
}

func TestKnowledgeVersionReviewAndExplicitRecheck(t *testing.T) {
	store, id := auxiliaryDurableFixture(t, 128)
	root := knowledgeTestRoot(t)
	original := knowledgeTestImport(t, store, id, root, "Keep the immutable source.", 0)
	confirmed := knowledgeTestReview(t, store, id, root, original, "confirm")
	if confirmed.State != "confirmed" || confirmed.Reviews[0].IdentityVerified || confirmed.Reviews[0].SuggestedSync == "" {
		t.Fatal("review lost its state, identity disclaimer or non-mutating sync suggestion")
	}
	if _, err := store.ReviewKnowledge(id, root, KnowledgeReviewRequest{
		Version: original.Version, ExpectedRevision: original.Revision, Action: "reject", Reason: "Stale review.",
	}, KnowledgeReviewer{Actor: "fixture operator"}); err == nil { t.Fatal("stale review silently overwrote the current version") }
	editedContent := confirmed.Content
	editedContent.Body = "Keep the immutable source and its exact hash."
	edited, err := store.ReviewKnowledge(id, root, KnowledgeReviewRequest{
		Version: confirmed.Version, ExpectedRevision: confirmed.Revision, Action: "edit", Reason: "New wording.", Edit: &editedContent,
	}, KnowledgeReviewer{Actor: "fixture operator"})
	if err != nil { t.Fatal(err) }
	if edited.State != "candidate" || edited.Version == confirmed.Version { t.Fatal("edited content inherited confirmation") }
	state, err := store.Load(id)
	if err != nil { t.Fatal(err) }
	preserved, err := knowledgeFind(state, confirmed.Version)
	if err != nil || preserved.State != "confirmed" { t.Fatal("editing invalidated the unchanged old entry") }
	if err := os.WriteFile(filepath.Join(root, "requirement.txt"), []byte("changed requirement"), 0600); err != nil { t.Fatal(err) }
	if err := store.RecheckKnowledge(id, root); err != nil { t.Fatal(err) }
	state, err = store.Load(id)
	if err != nil { t.Fatal(err) }
	stale, err := knowledgeFind(state, confirmed.Version)
	if err != nil || stale.State != "needs-review" { t.Fatal("changed bound source retained confirmed trust") }
	if err := os.WriteFile(filepath.Join(root, "requirement.txt"), []byte("fixed requirement"), 0600); err != nil { t.Fatal(err) }
	if err := store.RecheckKnowledge(id, root); err != nil { t.Fatal(err) }
	state, err = store.Load(id)
	if err != nil { t.Fatal(err) }
	stale, err = knowledgeFind(state, confirmed.Version)
	if err != nil || stale.State != "needs-review" { t.Fatal("restored bytes automatically restored confirmation") }
	rechecked := knowledgeTestReview(t, store, id, root, *stale, "recheck")
	if rechecked.State != "confirmed" || rechecked.Version != confirmed.Version { t.Fatal("explicit recheck changed immutable identity") }
	rejected := knowledgeTestReview(t, store, id, root, edited, "reject")
	if rejected.State != "rejected" { t.Fatal("rejection was not retained") }
	text := "A differently named source containing the same rejected entry."
	if _, err := store.ImportHumanKnowledge(id, InputSource{
		Path: "renamed-human-note", Status: "loaded", Content: text, SHA256: contentDigest([]byte(text)),
	}, edited.Content); err == nil { t.Fatal("a new source identity bypassed content rejection") }
	retired := knowledgeTestReview(t, store, id, root, rechecked, "retire")
	if retired.State != "retired" { t.Fatal("retirement without replacement failed") }
}

func TestKnowledgeInjectionDeterministicAndPreservesRequiredInput(t *testing.T) {
	store, id := auxiliaryDurableFixture(t, 128)
	root := knowledgeTestRoot(t)
	candidate := knowledgeTestImport(t, store, id, root, "Candidate rule.", 100)
	confirmed := knowledgeTestImport(t, store, id, root, "Confirmed rule.", -100)
	confirmed = knowledgeTestReview(t, store, id, root, confirmed, "confirm")
	rejected := knowledgeTestImport(t, store, id, root, "Rejected rule.", 100)
	knowledgeTestReview(t, store, id, root, rejected, "reject")
	retired := knowledgeTestImport(t, store, id, root, "Retired rule.", 100)
	knowledgeTestReview(t, store, id, root, retired, "retire")
	// Deterministic test counter only; this does not establish real model capacity.
	capacity := int64(100000)
	store.AuxiliaryServices.MeasureKnowledge = func(_ *AISelection, text string) (int64, int64, error) {
		return int64(len(text)), capacity, nil
	}
	prompt := "Implement memory from these mandatory requirements.\n"
	first, sources, err := store.InjectKnowledge(id, root, prompt, nil)
	if err != nil { t.Fatal(err) }
	second, repeated, err := store.InjectKnowledge(id, root, prompt, nil)
	if err != nil { t.Fatal(err) }
	if first != second || !equalJSON(sources, repeated) { t.Fatal("same input produced different knowledge selection") }
	if len(sources) != 2 || !strings.Contains(sources[0].Path, confirmed.Version) || !strings.Contains(sources[1].Path, candidate.Version) {
		t.Fatal("trust did not take precedence over priority or terminal entries were injected")
	}
	if !strings.HasPrefix(first, prompt) || !strings.Contains(sources[1].Content, "Secondary reference only") || strings.Contains(first, "Rejected rule.") || strings.Contains(first, "Retired rule.") {
		t.Fatal("required input or trust boundaries were lost")
	}
	capacity = int64(len(prompt))
	limited, skips, err := store.InjectKnowledge(id, root, prompt, nil)
	if err != nil || limited != prompt || len(skips) == 0 || skips[len(skips)-1].Status != "unavailable" {
		t.Fatal("optional entries displaced required input or were silently dropped")
	}
	capacity--
	if _, _, err := store.InjectKnowledge(id, root, prompt, nil); err == nil { t.Fatal("oversized required input was silently accepted") }
	store.AuxiliaryServices.MeasureKnowledge = nil
	unchanged, gaps, err := store.InjectKnowledge(id, root, prompt, nil)
	if err != nil || unchanged != prompt || len(gaps) != 1 || gaps[0].Status != "unavailable" { t.Fatal("unknown model capacity lost primary input") }
}

func TestKnowledgeHistoryLimitsSurviveRestart(t *testing.T) {
	for _, test := range []struct{name string; count int; bytes int64}{
		{"source-count", 16, 1}, {"byte-total", 1, 1024*1024},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, id := auxiliaryDurableFixture(t, 128)
			var last []KnowledgeHistorySource
			for batch := 0; batch < 4; batch++ {
				last = nil
				for i := 0; i < test.count; i++ {
					last = append(last, KnowledgeHistorySource{Owner: "source-task", SourceID: fmt.Sprintf("source-%d-%d", batch, i), Bytes: test.bytes})
				}
				if err := store.reserveKnowledgeHistory(id, last); err != nil { t.Fatal(err) }
			}
			reopened := NewStore(store.Root)
			reopened.AuxiliaryServices = store.AuxiliaryServices
			before, err := reopened.Load(id)
			if err != nil { t.Fatal(err) }
			if err := reopened.reserveKnowledgeHistory(id, last); err != nil { t.Fatal(err) }
			if err := reopened.reserveKnowledgeHistory(id, []KnowledgeHistorySource{{Owner: "source-task", SourceID: "one-more", Bytes: 1}}); err == nil {
				t.Fatal("another batch after reopening bypassed the cumulative limit")
			}
			after, err := reopened.Load(id)
			if err != nil { t.Fatal(err) }
			if before.StateRevision != after.StateRevision || !equalJSON(before.Protocol.Auxiliary.Knowledge.History, after.Protocol.Auxiliary.Knowledge.History) {
				t.Fatal("duplicate or rejected history changed the durable ledger")
			}
		})
	}
}
