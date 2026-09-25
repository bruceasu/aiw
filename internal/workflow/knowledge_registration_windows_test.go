//go:build windows

package workflow

import (
	"fmt"
	"testing"
)

func knowledgeRegistrationFixture(t *testing.T) (*Store, TaskID) {
	t.Helper()
	store, id := auxiliaryDurableFixture(t, 128)
	store.AuxiliaryServices.Authorize = func(state RuntimeState, job AuxiliaryJob) error {
		if state.Task.ID != id || job.Owner != id || (job.Kind != "knowledge-extraction" && job.Kind != "knowledge-summary") {
			return fmt.Errorf("outside fixture knowledge authorization")
		}
		return nil
	}
	if err := store.RegisterTaskKnowledge(id); err != nil { t.Fatal(err) }
	state, err := store.Load(id)
	if err != nil { t.Fatal(err) }
	for _, job := range state.Protocol.Auxiliary.Jobs {
		if job.Kind == "knowledge-extraction" || job.Kind == "knowledge-summary" { t.Fatal("unaccepted progress scheduled knowledge generation") }
	}
	// Seed a committed acceptance fact; the full acceptance pipeline is outside
	// this fixture. Its artifact is real immutable temporary Store content.
	_, err = store.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.test.accepted-source"}, func(current *RuntimeState) error {
		ref, err := store.persistProtocolArtifactLocked(id, "test-acceptance", map[string]string{"requirement": "fixed accepted fixture"})
		if err != nil { return err }
		current.WorkItems[0].State = WorkItemCompleted
		current.WorkItems[0].AcceptedReference = &ref
		return nil
	})
	if err != nil { t.Fatal(err) }
	if err := store.RegisterTaskKnowledge(id); err != nil { t.Fatal(err) }
	return store, id
}

func knowledgeLatestJob(t *testing.T, store *Store, id TaskID, kind string) AuxiliaryReference {
	t.Helper()
	state, err := store.Load(id)
	if err != nil { t.Fatal(err) }
	jobs := state.Protocol.Auxiliary.Jobs
	for i := len(jobs)-1; i >= 0; i-- {
		if jobs[i].Kind == kind { return AuxiliaryReference{SourceOwner: id, Key: jobs[i].Key} }
	}
	t.Fatalf("missing registered %s", kind)
	return AuxiliaryReference{}
}

func knowledgePublishFixtureOutput(t *testing.T, store *Store, ref AuxiliaryReference, text string) {
	t.Helper()
	job, err := store.ResolveAuxiliary(ref)
	if err != nil { t.Fatal(err) }
	if err := ValidateKnowledgeOutput(job, text); err != nil { t.Fatal(err) }
	call := auxiliaryTestClaim(t, store, ref)
	auxiliaryTestObserve(t, store, call, "valid", text)
	if err := store.PublishAuxiliaryOutput(ref); err != nil { t.Fatal(err) }
}

func TestKnowledgeAcceptedRegistrationDeduplicatesAfterRestart(t *testing.T) {
	store, id := knowledgeRegistrationFixture(t)
	before, err := store.Load(id)
	if err != nil { t.Fatal(err) }
	k := before.Protocol.Auxiliary.Knowledge
	if len(k.Extractions) != 1 || k.Cursor != len(before.Protocol.Auxiliary.Sources) { t.Fatal("accepted source was not registered and consumed") }
	extraction := knowledgeLatestJob(t, store, id, "knowledge-extraction")
	knowledgeLatestJob(t, store, id, "knowledge-summary")
	reopened := NewStore(store.Root)
	reopened.AuxiliaryServices = store.AuxiliaryServices
	if err := reopened.RegisterTaskKnowledge(id); err != nil { t.Fatal(err) }
	after, err := reopened.Load(id)
	if err != nil { t.Fatal(err) }
	if before.StateRevision != after.StateRevision || !equalJSON(before.Protocol.Auxiliary, after.Protocol.Auxiliary) {
		t.Fatal("restart duplicated jobs or changed the source cursor")
	}
	auxiliaryTestChangeTitle(t, reopened, id)
	if err := reopened.RegisterTaskKnowledge(id); err != nil { t.Fatal(err) }
	after, err = reopened.Load(id)
	if err != nil { t.Fatal(err) }
	count := 0
	for _, job := range after.Protocol.Auxiliary.Jobs { if job.Kind == "knowledge-extraction" { count++; if job.Key != extraction.Key { t.Fatal("unchanged acceptance received a new extraction key") } } }
	if count != 1 || len(after.Protocol.Auxiliary.Knowledge.Extractions) != 1 { t.Fatal("later progress re-extracted the same acceptance") }
	resources, err := reopened.readAuxiliaryResources()
	if err != nil { t.Fatal(err) }
	if len(resources.Reservations) != 0 { t.Fatal("registration dispatched a model synchronously") }
}

func TestKnowledgePartialDraftPublishesDurablyThenAdvances(t *testing.T) {
	store, id := knowledgeRegistrationFixture(t)
	before, err := store.Load(id)
	if err != nil { t.Fatal(err) }
	partial := knowledgeLatestJob(t, store, id, "knowledge-summary")
	knowledgePublishFixtureOutput(t, store, partial, "Partial draft: the accepted Work Item extraction is still running.")
	reopened := NewStore(store.Root)
	reopened.AuxiliaryServices = store.AuxiliaryServices
	state, err := reopened.Load(id)
	if err != nil { t.Fatal(err) }
	drafts := state.Protocol.Auxiliary.Knowledge.Drafts
	if len(drafts) != 1 || len(drafts[0].Coverage) != 1 || drafts[0].Coverage[0].State != "running" { t.Fatal("durable partial draft hid the running extraction") }
	oldDraft := drafts[0]
	extraction := knowledgeLatestJob(t, reopened, id, "knowledge-extraction")
	knowledgePublishFixtureOutput(t, reopened, extraction, `{"entries":[],"no_new_reason":"The accepted source repeats existing knowledge."}`)
	if err := reopened.RegisterTaskKnowledge(id); err != nil { t.Fatal(err) }
	updated := knowledgeLatestJob(t, reopened, id, "knowledge-summary")
	if updated == partial { t.Fatal("changed extraction coverage reused the old summary input") }
	knowledgePublishFixtureOutput(t, reopened, updated, "Complete extraction coverage: no new reusable knowledge, with a recorded reason.")
	finalStore := NewStore(store.Root)
	after, err := finalStore.Load(id)
	if err != nil { t.Fatal(err) }
	k := after.Protocol.Auxiliary.Knowledge
	if len(k.Drafts) != 2 || !equalJSON(k.Drafts[0], oldDraft) || k.Drafts[0].Version == k.Drafts[1].Version { t.Fatal("new coverage overwrote the earlier draft") }
	row := k.Drafts[1].Coverage[0]
	if row.State != "success" || row.Output == nil || len(k.Versions) != 0 { t.Fatal("explained no-new extraction became missing coverage or a fabricated rule") }
	var output AuxiliaryOutput
	if err := finalStore.ReadExecutionArtifact(id, k.Drafts[1].Output, &output); err != nil { t.Fatal(err) }
	if output.JobKey != updated.Key || output.Text == "" { t.Fatal("draft lacks its exact durable output") }
	if !equalJSON(before.WorkItems, after.WorkItems) || !equalJSON(before.Gates, after.Gates) || before.Delivery != after.Delivery { t.Fatal("auxiliary publication changed accepted work or delivery") }
}
