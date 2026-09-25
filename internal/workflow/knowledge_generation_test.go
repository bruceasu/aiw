package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func knowledgeGenerationReference(hash string) ActorReference {
	return ActorReference{Kind: "fixture", Path: "reports/"+hash+".json", SHA256: strings.Repeat(hash, 64)}
}

func knowledgeGenerationJob(t *testing.T, key, kind string, context knowledgeJobContext, text string) AuxiliaryJob {
	t.Helper()
	return AuxiliaryJob{Key: key, Kind: kind, Owner: "task-1", SourceVersion: "fixed-source", Prompt: knowledgeInstruction(kind, context), Result: &AuxiliaryObservation{State: "valid", Text: text}}
}

func TestKnowledgeExtractionRequiresExplicitNoNewEvidence(t *testing.T) {
	acceptance := knowledgeGenerationReference("a")
	job := knowledgeGenerationJob(t, "extraction", "knowledge-extraction", knowledgeJobContext{WorkItem: "one", Acceptance: &acceptance}, "")
	for _, test := range []struct{name, text string; valid bool}{
		{"explained-empty", `{"entries":[],"no_new_reason":"The accepted source repeats an existing rule."}`, true},
		{"unexplained-empty", `{"entries":[],"no_new_reason":""}`, false},
		{"missing-entries", `{"no_new_reason":"No result was returned."}`, false},
		{"null-entries", `{"entries":null,"no_new_reason":"No result was returned."}`, false},
		{"unknown-field", `{"entries":[],"no_new_reason":"No change.","approved":true}`, false},
		{"empty-response", ``, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateKnowledgeOutput(job, test.text)
			if (err == nil) != test.valid { t.Fatalf("valid=%v error=%v", test.valid, err) }
		})
	}
	state := RuntimeState{Protocol: &ExecutionProtocol{Auxiliary: &TaskAuxiliary{}}}
	job.Result.Text = `{"entries":[],"no_new_reason":"The accepted source repeats an existing rule."}`
	if err := (&Store{}).publishKnowledge(&state, job, knowledgeGenerationReference("b"), auxiliaryResources{}); err != nil { t.Fatal(err) }
	k := state.Protocol.Auxiliary.Knowledge
	if !k.Published[job.Key] || len(k.Versions) != 0 { t.Fatal("explained no-new extraction generated a rule or was not recorded") }
}

func TestKnowledgePartialDraftPreservesCoverageAndHistory(t *testing.T) {
	acceptance := knowledgeGenerationReference("a")
	output := knowledgeGenerationReference("b")
	source := AuxiliarySource{Owner: "task-1", Version: "fixed-source", Items: []AuxiliarySourceItem{
		{ID: "success", Accepted: &acceptance}, {ID: "running", Accepted: &acceptance},
		{ID: "missing", Accepted: &acceptance}, {ID: "failed", Accepted: &acceptance},
	}}
	state := RuntimeState{Protocol: &ExecutionProtocol{Auxiliary: &TaskAuxiliary{}}}
	a := state.Protocol.Auxiliary
	k := taskKnowledge(a)
	for _, item := range source.Items {
		if item.ID == "missing" { continue }
		key := string(item.ID)
		k.Extractions[extractionIdentity(item)] = key
		job := AuxiliaryJob{Key: key, State: "running"}
		if item.ID == "failed" { job.State, job.Reason = "unavailable", "extraction exhausted" }
		if item.ID == "success" { job.State, job.Output = "completed", &output }
		a.Jobs = append(a.Jobs, job)
	}
	coverage, entries := knowledgeCoverage(state, source)
	if len(coverage) != 4 { t.Fatal("partial coverage omitted a required Work Item") }
	want := []string{"failed", "missing", "running", "success"}
	for i, row := range coverage {
		if string(row.WorkItem) != want[i] || row.State != want[i] { t.Fatalf("coverage[%d]=%+v", i, row) }
		if row.State != "success" && row.Reason == "" { t.Fatal("partial draft omitted its gap reason") }
	}
	job := knowledgeGenerationJob(t, "partial", "knowledge-summary", knowledgeJobContext{Coverage: coverage, Entries: entries}, "Partial draft: failed, missing and running extractions remain.")
	store := &Store{}
	if err := store.publishKnowledge(&state, job, output, auxiliaryResources{}); err != nil { t.Fatal(err) }
	if len(k.Drafts) != 1 || !equalJSON(k.Drafts[0].Coverage, coverage) { t.Fatal("published partial draft lost exact coverage") }
	original, err := json.Marshal(k.Drafts[0])
	if err != nil { t.Fatal(err) }
	for i := range a.Jobs {
		if a.Jobs[i].Key == "failed" { a.Jobs[i].State, a.Jobs[i].Reason, a.Jobs[i].Output = "completed", "", &output }
	}
	coverage, entries = knowledgeCoverage(state, source)
	updated := knowledgeGenerationJob(t, "updated-partial", "knowledge-summary", knowledgeJobContext{Coverage: coverage, Entries: entries}, "Updated draft: missing and running extractions remain.")
	if err := store.publishKnowledge(&state, updated, knowledgeGenerationReference("c"), auxiliaryResources{}); err != nil { t.Fatal(err) }
	if len(k.Drafts) != 2 || k.Drafts[0].Version == k.Drafts[1].Version || k.Drafts[1].Coverage[0].State != "success" {
		t.Fatal("new extraction coverage did not create a distinct draft")
	}
	preserved, err := json.Marshal(k.Drafts[0])
	if err != nil { t.Fatal(err) }
	if string(original) != string(preserved) { t.Fatal("new draft overwrote old coverage history") }
	if err := store.publishKnowledge(&state, updated, knowledgeGenerationReference("c"), auxiliaryResources{}); err != nil { t.Fatal(err) }
	if len(k.Drafts) != 2 { t.Fatal("publication replay duplicated the draft") }
}

func TestKnowledgeGeneratedVersionsPreservePriorReview(t *testing.T) {
	store := &Store{}
	state := RuntimeState{Protocol: &ExecutionProtocol{Auxiliary: &TaskAuxiliary{}}}
	acceptance := knowledgeGenerationReference("a")
	content := KnowledgeContent{Body: "Preserve fixed inputs.", EvidenceNature: "observed fixture evidence", Scope: []string{"module:memory"}, Usage: "Check current requirements.", InvalidWhen: "The bound source changes.", Checks: []InputSource{}}
	publish := func(key string, value KnowledgeContent, accepted ActorReference) {
		t.Helper()
		text, err := json.Marshal(knowledgeExtraction{Entries: []KnowledgeContent{value}, NoNewReason: ""})
		if err != nil { t.Fatal(err) }
		job := knowledgeGenerationJob(t, key, "knowledge-extraction", knowledgeJobContext{WorkItem: "one", Acceptance: &accepted}, string(text))
		if err := store.publishKnowledge(&state, job, knowledgeGenerationReference("b"), auxiliaryResources{}); err != nil { t.Fatal(err) }
	}
	publish("first", content, acceptance)
	k := state.Protocol.Auxiliary.Knowledge
	if len(k.Versions) != 1 || k.Versions[0].State != "candidate" { t.Fatal("generation did not produce an unconfirmed candidate") }
	// Seed a previously reviewed version; the human review API is tested separately.
	k.Versions[0].State, k.Versions[0].Revision = "confirmed", 2
	k.Versions[0].Reviews = []KnowledgeReview{{Action: "confirm", Actor: "fixture reviewer", Reason: "Reviewed fixture."}}
	confirmed, err := json.Marshal(k.Versions[0])
	if err != nil { t.Fatal(err) }
	publish("same-input-new-job", content, acceptance)
	if len(k.Versions) != 1 { t.Fatal("unchanged accepted content duplicated a reviewed version") }
	changed := content
	changed.Body = "Preserve fixed inputs and exact digests."
	publish("new-body", changed, acceptance)
	publish("new-acceptance", content, knowledgeGenerationReference("c"))
	if len(k.Versions) != 3 { t.Fatal("changed body or bound acceptance did not create a version") }
	for _, v := range k.Versions[1:] {
		if v.State != "candidate" || len(v.Reviews) != 0 || v.Version == k.Versions[0].Version { t.Fatal("new generation inherited confirmation or old identity") }
	}
	preserved, err := json.Marshal(k.Versions[0])
	if err != nil { t.Fatal(err) }
	if string(confirmed) != string(preserved) { t.Fatal("generation rewrote the existing human review") }
}
