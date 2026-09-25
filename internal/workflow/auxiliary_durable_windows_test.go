//go:build windows

package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This fixture exercises real temporary Store commits with explicit capability,
// authorization, inventory and capacity doubles. It is not activation evidence.
func auxiliaryDurableFixture(t *testing.T, queueLimit int) (*Store, TaskID) {
	t.Helper()
	store, state, proof := durableFixture(t)
	store.AuxiliaryServices = &AuxiliaryServices{
		FreeBytes: func(string) (int64, error) { return 8 * 1024 * 1024 * 1024, nil },
		Authorize: func(s RuntimeState, job AuxiliaryJob) error {
			if s.Task.ID != state.Task.ID || job.Owner != state.Task.ID || job.Kind != "memory" {
				return fmt.Errorf("outside fixture authorization")
			}
			return nil
		},
		VerifyCapability: func(cap AuxiliaryCapability) error {
			if cap.Reference != proof || cap.Selection.Provider != "fixture" || cap.Selection.Model != "fixture" {
				return fmt.Errorf("unexpected fixture capability")
			}
			return nil
		},
		VerifyInventory: func(inventory AuxiliaryInventory) error {
			if inventory.Evidence != proof || len(inventory.TaskBytes) != 1 {
				return fmt.Errorf("unexpected fixture inventory")
			}
			return nil
		},
		VerifyObservation: func(_ RuntimeState, _ AuxiliaryReservation, observation AuxiliaryObservation) error {
			var saved AuxiliaryObservation
			if err := store.ReadExecutionArtifact(state.Task.ID, observation.Evidence, &saved); err != nil {
				return err
			}
			observation.Evidence = ActorReference{}
			if !equalJSON(saved, observation) {
				return fmt.Errorf("fixture observation differs from its immutable proof")
			}
			return nil
		},
	}
	policy := DefaultAuxiliaryResourcePolicy()
	policy.TaskQueue = queueLimit
	data, err := json.Marshal(policy)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(store.Root, "resource-policy.json"), data, 0600); err != nil { t.Fatal(err) }
	if err := store.InitializeAuxiliaryResources(AuxiliaryInventory{
		Known: true, Evidence: proof, TaskBytes: map[TaskID]int64{state.Task.ID: 0},
	}); err != nil { t.Fatal(err) }
	if _, _, err := store.TaskMemoryContext(state.Task.ID); err != nil { t.Fatal(err) }
	if err := store.RegisterTaskMemory(state.Task.ID); err != nil { t.Fatal(err) }
	return store, state.Task.ID
}

func auxiliaryTestClaim(t *testing.T, store *Store, ref AuxiliaryReference) AuxiliaryReservation {
	t.Helper()
	job, err := store.ResolveAuxiliary(ref)
	if err != nil { t.Fatal(err) }
	state, err := store.Load(ref.SourceOwner)
	if err != nil { t.Fatal(err) }
	// Reuse the persisted activation proof from this test's durable fixture.
	// The verifier only accepts this fixture-specific identity.
	var proof ActorReference
	lock, err := store.lock(state.Task.ID)
	if err != nil { t.Fatal(err) }
	proof, err = store.persistProtocolArtifactLocked(state.Task.ID, "test-activation", map[string]string{"source": "test double"})
	unlock(lock)
	if err != nil { t.Fatal(err) }
	measurement := AuxiliaryMeasurement{
		PromptDigest: job.InputDigest, InputTokens: 100, OutputTokens: 32,
		Capability: AuxiliaryCapability{
			Selection: AISelection{Provider: "fixture", Model: "fixture", Digest: "fixed"},
			Reference: proof, CounterVersion: "fixture-v1", ContextTokens: 4096,
			OutputTokens: 32, EnforcedOutput: true, ClosedInput: true,
		},
	}
	call, err := store.ClaimAuxiliaryCall(ref, "fixture-worker", measurement, time.Now().Add(5*time.Minute))
	if err != nil { t.Fatal(err) }
	return call
}

func auxiliaryTestObserve(t *testing.T, store *Store, call AuxiliaryReservation, status, text string) AuxiliaryObservation {
	t.Helper()
	observation := AuxiliaryObservation{
		RequestID: call.ID, Key: call.Key, Executor: call.Executor,
		InputDigest: call.Measurement.PromptDigest, State: status, Text: text,
		Reason: "fixture observation",
	}
	proof, err := store.PersistAuxiliaryEvidence(call, observation)
	if err != nil { t.Fatal(err) }
	observation.Evidence = proof
	if err := store.ObserveAuxiliary(call.Owner, observation); err != nil { t.Fatal(err) }
	return observation
}

func auxiliaryTestLatest(t *testing.T, store *Store, id TaskID) (RuntimeState, AuxiliaryReference) {
	t.Helper()
	state, err := store.Load(id)
	if err != nil { t.Fatal(err) }
	jobs := state.Protocol.Auxiliary.Jobs
	if len(jobs) == 0 { t.Fatal("memory registration produced no job") }
	return state, AuxiliaryReference{SourceOwner: id, Key: jobs[len(jobs)-1].Key}
}

func auxiliaryTestChangeTitle(t *testing.T, store *Store, id TaskID) {
	t.Helper()
	state, err := store.Load(id)
	if err != nil { t.Fatal(err) }
	_, err = store.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.test.source-change"}, func(current *RuntimeState) error {
		current.WorkItems[0].Title = "New material decision"
		return nil
	})
	if err != nil { t.Fatal(err) }
}

func TestAuxiliaryRecoverySurvivesRestartAndConsumerReuse(t *testing.T) {
	store, id := auxiliaryDurableFixture(t, 128)
	_, ref := auxiliaryTestLatest(t, store, id)
	first := auxiliaryTestClaim(t, store, ref)
	auxiliaryTestObserve(t, store, first, "unknown", "")
	reopened := NewStore(store.Root)
	reopened.AuxiliaryServices = store.AuxiliaryServices
	active, err := reopened.AuxiliaryActiveCall()
	if err != nil || active == nil || active.ID != first.ID || active.Tokens != 132 {
		t.Fatalf("unknown dispatch lost its slot or reservation: %+v %v", active, err)
	}
	if _, err := reopened.ClaimAuxiliaryCall(ref, first.Executor, first.Measurement, time.Now().Add(5*time.Minute)); err == nil {
		t.Fatal("unknown dispatch permitted a replacement request")
	}
	terminated := auxiliaryTestObserve(t, reopened, first, "terminated", "")
	before, _ := auxiliaryTestLatest(t, reopened, id)
	if err := reopened.ObserveAuxiliary(id, terminated); err != nil { t.Fatal(err) }
	after, _ := auxiliaryTestLatest(t, reopened, id)
	if before.StateRevision != after.StateRevision { t.Fatal("replayed result committed another Task revision") }
	second := auxiliaryTestClaim(t, reopened, ref)
	if second.ID == first.ID { t.Fatal("recovery reused the original dispatch identity") }
	auxiliaryTestObserve(t, reopened, second, "invalid", "")
	job, err := reopened.ResolveAuxiliary(ref)
	if err != nil { t.Fatal(err) }
	if job.State != "unavailable" || !job.RecoveryUsed || len(job.Calls) != 2 {
		t.Fatal("fixed-input recovery did not terminate after its one extra call")
	}
	other, err := reopened.EnqueueAuxiliary(id, "other-consumer", job.SourceID, "memory", memoryInstruction)
	if err != nil || other != ref { t.Fatalf("consumer created a separate source ledger: %+v %v", other, err) }
	job, err = reopened.ResolveAuxiliary(other)
	if err != nil { t.Fatal(err) }
	if !job.RecoveryUsed || job.Sponsor != id || len(job.Calls) != 2 { t.Fatal("consumer reset budget or sponsor") }
	if _, err := reopened.ClaimAuxiliaryCall(other, first.Executor, first.Measurement, time.Now().Add(5*time.Minute)); err == nil {
		t.Fatal("consumer received a third attempt")
	}
	resources, err := reopened.readAuxiliaryResources()
	if err != nil { t.Fatal(err) }
	var tokens int64
	for _, call := range resources.Reservations { tokens += call.Tokens }
	if len(resources.Reservations) != 2 || tokens != 264 || resources.Active != "" {
		t.Fatal("unknown usage was released or restart reset accumulated reservations")
	}
}

func TestAuxiliaryQueueLimitPreservesSourceCursor(t *testing.T) {
	store, id := auxiliaryDurableFixture(t, 1)
	before, _ := auxiliaryTestLatest(t, store, id)
	auxiliaryTestChangeTitle(t, store, id)
	if err := store.RegisterTaskMemory(id); err == nil { t.Fatal("full queue admitted another source") }
	reopened := NewStore(store.Root)
	state, _ := auxiliaryTestLatest(t, reopened, id)
	a := state.Protocol.Auxiliary
	if a.Cursor != before.Protocol.Auxiliary.Cursor || len(a.Jobs) != 1 || len(a.Sources) <= a.Cursor {
		t.Fatal("failed registration lost its pending source or advanced the cursor")
	}
	resources, err := reopened.readAuxiliaryResources()
	if err != nil { t.Fatal(err) }
	if len(resources.Queue) != 1 || len(resources.Reservations) != 0 { t.Fatal("queue rejection created dispatch or extra reservation") }
}

func TestAuxiliaryLateMemoryPublicationAndStop(t *testing.T) {
	store, id := auxiliaryDurableFixture(t, 128)
	_, oldRef := auxiliaryTestLatest(t, store, id)
	oldCall := auxiliaryTestClaim(t, store, oldRef)
	auxiliaryTestChangeTitle(t, store, id)
	auxiliaryTestObserve(t, store, oldCall, "valid", "Old summary must stay historical.")
	if err := store.PublishAuxiliaryOutput(oldRef); err != nil { t.Fatal(err) }
	state, _ := auxiliaryTestLatest(t, store, id)
	oldJob, err := store.ResolveAuxiliary(oldRef)
	if err != nil { t.Fatal(err) }
	if oldJob.State != "completed" || oldJob.Output == nil || state.Protocol.Auxiliary.Memory != nil {
		t.Fatal("late output was lost or replaced current memory")
	}
	text, _, err := store.TaskMemoryContext(id)
	if err != nil || !strings.Contains(text, "New material decision") || !strings.Contains(text, "[Degraded:") || strings.Contains(text, "Old summary must stay historical.") {
		t.Fatalf("stale summary replaced original current facts: %q %v", text, err)
	}
	if err := store.RegisterTaskMemory(id); err != nil { t.Fatal(err) }
	_, currentRef := auxiliaryTestLatest(t, store, id)
	currentCall := auxiliaryTestClaim(t, store, currentRef)
	auxiliaryTestObserve(t, store, currentCall, "valid", "Current summary.")
	if err := store.PublishAuxiliaryOutput(currentRef); err != nil { t.Fatal(err) }
	state, _ = auxiliaryTestLatest(t, store, id)
	if err := store.PublishAuxiliaryOutput(currentRef); err != nil { t.Fatal(err) }
	replayed, _ := auxiliaryTestLatest(t, store, id)
	if state.StateRevision != replayed.StateRevision { t.Fatal("publication replay consumed another commit") }
	text, _, err = store.TaskMemoryContext(id)
	if err != nil || !strings.Contains(text, "Current summary.") || strings.Contains(text, "[Degraded:") {
		t.Fatalf("current version-bound summary was not projected: %q %v", text, err)
	}
	if _, err := store.RequestExecutionStop(id, replayed.StateRevision, "operator stop", "fixture"); err != nil { t.Fatal(err) }
	if err := store.RegisterTaskMemory(id); err != nil { t.Fatal(err) }
	_, stoppedRef := auxiliaryTestLatest(t, store, id)
	stoppedJob, err := store.ResolveAuxiliary(stoppedRef)
	if err != nil { t.Fatal(err) }
	measurement := currentCall.Measurement
	measurement.PromptDigest = stoppedJob.InputDigest
	if _, err := store.ClaimAuxiliaryCall(stoppedRef, currentCall.Executor, measurement, time.Now().Add(5*time.Minute)); err == nil {
		t.Fatal("Stop allowed a new auxiliary dispatch")
	}
	reopened := NewStore(store.Root)
	state, _ = auxiliaryTestLatest(t, reopened, id)
	if state.Protocol.Stop == nil { t.Fatal("reopening erased Stop") }
}
