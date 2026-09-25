//go:build windows

package workflow

import "testing"

// Reconstruct a committed interruption boundary without a disk-fault hook:
// publication reservation exists, but the following output/Task commit did not
// finish. This is not evidence of a real process kill or failed disk operation.
func auxiliaryInterruptedSaveFixture(t *testing.T, modelRecovery, outputStored bool) (*Store, AuxiliaryReference, *ActorReference) {
	t.Helper()
	store, id := auxiliaryDurableFixture(t, 128)
	_, ref := auxiliaryTestLatest(t, store, id)
	call := auxiliaryTestClaim(t, store, ref)
	if modelRecovery {
		auxiliaryTestObserve(t, store, call, "terminated", "")
		call = auxiliaryTestClaim(t, store, ref)
	}
	auxiliaryTestObserve(t, store, call, "valid", "Preserve this exact reusable output.")
	state, err := store.Load(id)
	if err != nil { t.Fatal(err) }
	_, err = store.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.test.interrupted-publication"}, func(current *RuntimeState) error {
		job, err := mutableAuxiliaryJob(current, ref.Key)
		if err != nil { return err }
		job.PublishAttempts = 1
		return nil
	})
	if err != nil { t.Fatal(err) }
	var stored *ActorReference
	if outputStored {
		job, err := store.ResolveAuxiliary(ref)
		if err != nil { t.Fatal(err) }
		lock, err := store.lock(id)
		if err != nil { t.Fatal(err) }
		output, err := store.persistProtocolArtifactLocked(id, "auxiliary-output", AuxiliaryOutput{
			JobKey: job.Key, SourceID: job.SourceID, SourceVersion: job.SourceVersion,
			InputDigest: job.InputDigest, Text: job.Result.Text,
		})
		unlock(lock)
		if err != nil { t.Fatal(err) }
		stored = &output
	}
	reopened := NewStore(store.Root)
	reopened.AuxiliaryServices = store.AuxiliaryServices
	return reopened, ref, stored
}

func TestAuxiliarySaveRecoverySurvivesReopen(t *testing.T) {
	store, ref, _ := auxiliaryInterruptedSaveFixture(t, false, false)
	before, err := store.ResolveAuxiliary(ref)
	if err != nil { t.Fatal(err) }
	if before.PublishAttempts != 1 || before.RecoveryUsed || before.Output != nil {
		t.Fatal("fixture did not persist the interrupted first-save boundary")
	}
	other, err := store.EnqueueAuxiliary(ref.SourceOwner, "another-consumer", before.SourceID, "memory", memoryInstruction)
	if err != nil || other != ref { t.Fatalf("consumer did not reuse the interrupted job: %+v %v", other, err) }
	if err := store.PublishAuxiliaryOutput(other); err != nil { t.Fatal(err) }
	job, err := store.ResolveAuxiliary(ref)
	if err != nil { t.Fatal(err) }
	if job.State != "completed" || job.PublishAttempts != 2 || !job.RecoveryUsed || len(job.Calls) != 1 || job.Output == nil {
		t.Fatal("save recovery did not consume exactly the shared extra allowance")
	}
	var output AuxiliaryOutput
	if err := store.ReadExecutionArtifact(ref.SourceOwner, *job.Output, &output); err != nil { t.Fatal(err) }
	if output.Text != before.Result.Text || output.InputDigest != before.InputDigest { t.Fatal("save recovery changed the fixed output") }
	state, err := store.Load(ref.SourceOwner)
	if err != nil { t.Fatal(err) }
	if err := store.PublishAuxiliaryOutput(ref); err != nil { t.Fatal(err) }
	after, err := store.Load(ref.SourceOwner)
	if err != nil { t.Fatal(err) }
	if state.StateRevision != after.StateRevision { t.Fatal("completed output replay consumed another commit") }
	resources, err := store.readAuxiliaryResources()
	if err != nil { t.Fatal(err) }
	if len(resources.Reservations) != 1 || !resources.Queue[ref.Key].Terminal { t.Fatal("saving output created another model dispatch or left the queue open") }
}

func TestAuxiliaryModelRecoveryLeavesNoSaveRetry(t *testing.T) {
	store, ref, _ := auxiliaryInterruptedSaveFixture(t, true, false)
	before, err := store.Load(ref.SourceOwner)
	if err != nil { t.Fatal(err) }
	if err := store.PublishAuxiliaryOutput(ref); err != nil { t.Fatal(err) }
	job, err := store.ResolveAuxiliary(ref)
	if err != nil { t.Fatal(err) }
	if job.State != "unavailable" || job.Reason == "" || job.PublishAttempts != 1 || !job.RecoveryUsed || len(job.Calls) != 2 || job.Output != nil {
		t.Fatal("model and save recovery received separate extra allowances")
	}
	if err := store.PublishAuxiliaryOutput(ref); err == nil { t.Fatal("unavailable output was retried") }
	after, err := store.Load(ref.SourceOwner)
	if err != nil { t.Fatal(err) }
	if !equalJSON(before.WorkItems, after.WorkItems) || !equalJSON(before.Attempts, after.Attempts) || !equalJSON(before.Gates, after.Gates) || before.Planning != after.Planning || before.Delivery != after.Delivery {
		t.Fatal("auxiliary exhaustion changed development or delivery")
	}
	resources, err := store.readAuxiliaryResources()
	if err != nil { t.Fatal(err) }
	if len(resources.Reservations) != 2 || resources.Active != "" || !resources.Queue[ref.Key].Terminal { t.Fatal("exhaustion lost the existing dispatch history or reopened the queue") }
}

func TestAuxiliaryStoredOutputReconcilesWithoutExtraRecovery(t *testing.T) {
	store, ref, stored := auxiliaryInterruptedSaveFixture(t, true, true)
	if stored == nil { t.Fatal("fixture did not save its immutable orphan output") }
	if err := store.PublishAuxiliaryOutput(ref); err != nil { t.Fatal(err) }
	job, err := store.ResolveAuxiliary(ref)
	if err != nil { t.Fatal(err) }
	if job.State != "completed" || job.Output == nil || *job.Output != *stored || job.PublishAttempts != 1 || len(job.Calls) != 2 || !job.RecoveryUsed {
		t.Fatal("reconciling already stored bytes required another save or model call")
	}
	state, err := store.Load(ref.SourceOwner)
	if err != nil { t.Fatal(err) }
	if state.Protocol.Auxiliary.Memory == nil || state.Protocol.Auxiliary.Memory.Output != *stored {
		t.Fatal("publication reconciliation did not bind the exact stored output")
	}
	resources, err := store.readAuxiliaryResources()
	if err != nil { t.Fatal(err) }
	if len(resources.Reservations) != 2 || !resources.Queue[ref.Key].Terminal { t.Fatal("reconciliation changed model accounting") }
}
