package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuxiliarySourceTracksMaterialFactsOnly(t *testing.T) {
	state := RuntimeState{
		Task: TaskReference{ID: "owner"},
		WorkItems: []WorkItem{{ID: "item", Title: "Fixed requirement", State: WorkItemReady}},
		Protocol: &ExecutionProtocol{},
	}
	captureAuxiliarySource(&state)
	if len(state.Protocol.Auxiliary.Sources) != 1 {
		t.Fatal("initial facts did not produce exactly one source")
	}
	original := state.Protocol.Auxiliary.Sources[0]
	originalBytes, err := json.Marshal(original)
	if err != nil { t.Fatal(err) }
	state.StateRevision++
	state.LastEventSequence++
	state.Protocol.Requests = append(state.Protocol.Requests, StageRecord{
		Request: StageRequest{ActorRequest: ActorRequest{ID: "new-session-request"}},
		Dispatch: "intent",
	})
	captureAuxiliarySource(&state)
	if len(state.Protocol.Auxiliary.Sources) != 1 {
		t.Fatal("revision or unexecuted Session request created new memory work")
	}
	state.WorkItems[0].State = WorkItemCompleted
	captureAuxiliarySource(&state)
	sources := state.Protocol.Auxiliary.Sources
	if len(sources) != 2 || sources[1].Kind != "progress" || sources[1].Version == original.Version {
		t.Fatal("changed facts were lost or completion without acceptance was trusted")
	}
	accepted := ActorReference{Kind: "acceptance", Path: "reports/accepted.json", SHA256: strings.Repeat("a", 64)}
	state.WorkItems[0].AcceptedReference = &accepted
	captureAuxiliarySource(&state)
	sources = state.Protocol.Auxiliary.Sources
	if len(sources) != 3 || sources[2].Kind != "completed" || len(sources[2].References) != 1 || sources[2].References[0] != accepted {
		t.Fatal("accepted completion did not retain its exact evidence")
	}
	preserved, err := json.Marshal(sources[0])
	if err != nil { t.Fatal(err) }
	if string(preserved) != string(originalBytes) {
		t.Fatal("later facts overwrote an earlier source snapshot")
	}
}

func TestAuxiliaryJobIdentitySharedAcrossSponsors(t *testing.T) {
	store := &Store{}
	source := AuxiliarySource{ID: "owner:fixed", Owner: "owner", Version: "fixed", Kind: "progress"}
	first, err := store.buildAuxiliaryJob(source, "consumer-one", "memory", "Summarize the fixed source.")
	if err != nil { t.Fatal(err) }
	second, err := store.buildAuxiliaryJob(source, "consumer-two", "memory", "Summarize the fixed source.")
	if err != nil { t.Fatal(err) }
	if first.Key == "" || first.Key != second.Key || first.InputDigest != second.InputDigest || first.Prompt != second.Prompt {
		t.Fatal("another consumer created a new identity for the same fixed input")
	}
	if first.Owner != source.Owner || second.Owner != source.Owner || first.Sponsor == second.Sponsor {
		t.Fatal("source ownership or distinct consumer attribution was lost")
	}
	changedInstruction, err := store.buildAuxiliaryJob(source, "consumer-one", "memory", "Summarize risks from the fixed source.")
	if err != nil { t.Fatal(err) }
	changedKind, err := store.buildAuxiliaryJob(source, "consumer-one", "knowledge-summary", "Summarize the fixed source.")
	if err != nil { t.Fatal(err) }
	source.ID, source.Version = "owner:changed", "changed"
	changedSource, err := store.buildAuxiliaryJob(source, "consumer-one", "memory", "Summarize the fixed source.")
	if err != nil { t.Fatal(err) }
	for _, changed := range []AuxiliaryJob{changedInstruction, changedKind, changedSource} {
		if changed.Key == first.Key {
			t.Fatal("changed operation or input reused the old job identity")
		}
	}
}

func TestAuxiliaryOversizeInputIsUnavailableWithoutTruncation(t *testing.T) {
	store := &Store{}
	source := AuxiliarySource{ID: "owner:fixed", Owner: "owner", Version: "fixed", Kind: "progress"}
	for _, kind := range []string{"memory", "knowledge-extraction", "knowledge-summary", "verifier"} {
		t.Run(kind, func(t *testing.T) {
			short, err := store.buildAuxiliaryJob(source, "consumer", kind, "Read the fixed source.")
			if err != nil || short.State != "pending" { t.Fatalf("valid input was not eligible: %s %v", short.State, err) }
			instruction := strings.Repeat("x", AuxiliaryInputBytes)
			job, err := store.buildAuxiliaryJob(source, "consumer", kind, instruction)
			if err != nil { t.Fatal(err) }
			if job.State != "unavailable" || job.Reason == "" {
				t.Fatal("oversized complete input remained eligible")
			}
			if len(job.Prompt) <= AuxiliaryInputBytes || !strings.HasPrefix(job.Prompt, instruction+"\n") || job.InputDigest != contentDigest([]byte(job.Prompt)) {
				t.Fatal("oversized input was truncated or detached from its digest")
			}
		})
	}
	for _, kind := range []string{"notification", "report", "diagnosis"} {
		if _, err := store.buildAuxiliaryJob(source, "consumer", kind, "Read the fixed source."); err == nil {
			t.Fatalf("unrelated operation %q entered the shared auxiliary ledger", kind)
		}
	}
}
