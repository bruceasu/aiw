//go:build windows

package workflow

import (
 "strings"
 "testing"
)

func verifierRegistrationFixture(t *testing.T, snapshot bool) (*Store, TaskID) {
 t.Helper()
 store, id := auxiliaryDurableFixture(t, 128)
 state, err := store.Load(id); if err != nil { t.Fatal(err) }
 // Construct acceptance facts in a temporary Store, not the production acceptance pipeline.
 _, err = store.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.test.verifier-source"}, func(s *RuntimeState) error {
  ref, err := store.persistProtocolArtifactLocked(id, "test-acceptance", map[string]string{"fixture":"accepted"}); if err != nil { return err }
  item := ItemExecution{WorkItemID:s.WorkItems[0].ID, AttemptID:"attempt-verifier", Phase:PhaseAccepted, Accepted:&ref}
  s.Attempts = append(s.Attempts, Attempt{ID:item.AttemptID, WorkItemID:item.WorkItemID, Workspace:s.Task.Workspace, State:AttemptCompleted})
  s.WorkItems[0].State, s.WorkItems[0].AcceptedReference = WorkItemCompleted, &ref
  if snapshot {
   v := VerifierSnapshot{Version:1, TaskID:id, WorkItemID:item.WorkItemID, AttemptID:item.AttemptID, Criteria:[]VerifierCriterion{{ID:"criterion-one", Text:"fixed requirement", SourceSHA256:strings.Repeat("a",64)}}, Diff:InputSource{Content:"fixed fixture diff", SHA256:strings.Repeat("b",64)}}
   fixed, err := store.persistProtocolArtifactLocked(id, "verifier-snapshot", v); if err != nil { return err }; item.VerifierSnapshot = &fixed
  }
  s.Protocol.Items = append(s.Protocol.Items, item)
  return nil
 }); if err != nil { t.Fatal(err) }
 return store,id
}

func TestVerifierRegistrationReusesFrozenSourceAfterRestart(t *testing.T) {
 store,id := verifierRegistrationFixture(t,true)
 if err := store.RegisterTaskVerifier(id); err != nil { t.Fatal(err) }
 before,err := store.Load(id); if err != nil { t.Fatal(err) }
 var original AuxiliaryJob
 count := 0
 for _,job := range before.Protocol.Auxiliary.Jobs { if job.Kind=="verifier" { count++; original=job } }
 if count!=1 { t.Fatalf("registered %d verifier jobs",count) }
 c,err := verifierJobContext(original); if err != nil { t.Fatal(err) }
 if c.Snapshot != *before.Protocol.Items[0].VerifierSnapshot { t.Fatal("snapshot identity lost") }
 reopened := NewStore(store.Root); reopened.AuxiliaryServices=store.AuxiliaryServices
 if err := reopened.RegisterTaskVerifier(id); err != nil { t.Fatal(err) }
 after,err := reopened.Load(id); if err != nil { t.Fatal(err) }
 if after.StateRevision!=before.StateRevision { t.Fatal("restart rewrote registration") }
 auxiliaryTestChangeTitle(t,reopened,id)
 // Compare registration against the intentional source change, not the old title.
 beforeRegistration,err := reopened.Load(id); if err != nil { t.Fatal(err) }
 if err := reopened.RegisterTaskVerifier(id); err != nil { t.Fatal(err) }
 after,err=reopened.Load(id); if err != nil { t.Fatal(err) }
 count=0
 for _,job := range after.Protocol.Auxiliary.Jobs { if job.Kind=="verifier" { count++; if !equalJSON(job,original) { t.Fatal("later progress replaced fixed job") } } }
 if count!=1 { t.Fatal("later progress duplicated verifier") }
 if !equalJSON(beforeRegistration,after) { t.Fatal("registration changed state after source update") }
}

func TestVerifierHistoricalAcceptanceDoesNotInventSnapshot(t *testing.T) {
 store,id := verifierRegistrationFixture(t,false)
 before,err := store.Load(id); if err != nil { t.Fatal(err) }
 err=store.RegisterTaskVerifier(id)
 if err==nil || !strings.Contains(err.Error(),"historical acceptance has no fixed snapshot") { t.Fatalf("missing explicit gap: %v",err) }
 after,err := store.Load(id); if err != nil { t.Fatal(err) }
 if !equalJSON(before,after) { t.Fatal("missing snapshot changed accepted state or fabricated a job") }
}
