package execution

import (
	"context"
	"errors"
	"time"

	"aiw/internal/workflow"
)

// AuxiliaryWorker is a controlled, read-only host adapter. Measure counts the
// complete frozen prompt (including its provider envelope). Start enforces the
// saved capability, output limits, deadline, and the whole child-process tree.
// It must journal the exact request before returning. Reconcile may only read
// that journal or inspect the original executor; it cannot launch new work.
// A context timeout alone is not proof that an executor has terminated.
type AuxiliaryWorker interface {
	Identity() string
	Measure(workflow.AuxiliaryJob) (workflow.AuxiliaryMeasurement, error)
	Start(context.Context, workflow.AuxiliaryJob, workflow.AuxiliaryReservation) (*workflow.AuxiliaryObservation, error)
	Reconcile(context.Context, workflow.AuxiliaryReservation) (*workflow.AuxiliaryObservation, error)
}

type AuxiliaryHost struct {
	Store *workflow.Store
	Workers map[string]AuxiliaryWorker
}

// Run is the bounded process entry for a managed detached host. Its caller must
// provide a host-owned context, not a foreground Session context. Task completion
// and delivery are deliberately not exit conditions. There is no polling daemon:
// pending, executable work is drained; waiting-only and unknown work are saved
// and left for the next managed launch. Notifications keep their own recovery
// policy and must not be routed through these four model-operation workers.
func (h AuxiliaryHost) Run(parent context.Context, id workflow.TaskID) error {
	if h.Store == nil { return errors.New("auxiliary Store is unavailable") }
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	deadline, _ := ctx.Deadline()
	tried := map[string]bool{}
	var waits []error
	for ctx.Err() == nil {
		state, err := h.Store.Load(id)
		if err != nil { return err }
		if state.Protocol == nil { return nil }
		if state.PendingEvent != nil {
			if _, err := h.Store.RecoverPendingEvent(id); err != nil { return err }
			continue
		}
		// Reconcile the project slot before looking for new work, including an
		// executor from a different source Task or one with a persisted Stop.
		active, err := h.Store.AuxiliaryActiveCall()
		if err != nil { return err }
		if active != nil {
			var worker AuxiliaryWorker
			for _, candidate := range h.Workers { if candidate != nil && candidate.Identity() == active.Executor { worker = candidate; break } }
			if worker == nil { return errors.New("original auxiliary executor unavailable; keep its slot and reservation") }
			callContext, stop := context.WithTimeout(ctx, 120*time.Second)
			observation, runErr := worker.Reconcile(callContext, *active)
			stop()
			if observation == nil { return errors.Join(runErr, errors.New("auxiliary dispatch remains unknown")) }
			if err := h.Store.ObserveAuxiliary(active.Owner, *observation); err != nil { return errors.Join(runErr, err) }
			if observation.State == "unknown" || observation.State == "running" { return runErr }
			continue
		}
		// Registration errors preserve unconsumed facts; already registered
		// work can still make progress independently of this projection gap.
		registrationErr := errors.Join(h.Store.RegisterTaskVerifier(id), h.Store.RegisterTaskMemory(id), h.Store.RegisterTaskKnowledge(id))
		state, err = h.Store.Load(id)
		if err != nil { return err }
		if state.Protocol.Auxiliary == nil { return registrationErr }
		progress := false
		for _, job := range state.Protocol.Auxiliary.Jobs {
			ref := workflow.AuxiliaryReference{SourceOwner: id, Key: job.Key}
			if job.State == "completed" || job.State == "unavailable" { continue }
			if job.Result != nil && job.Result.State == "valid" {
				if tried[job.Key+":publish"] { continue }
				tried[job.Key+":publish"] = true
				if err := h.Store.PublishAuxiliaryOutput(ref); err != nil { return err }
				progress = true
				break
			}
			if state.Protocol.Stop != nil || tried[job.Key] || time.Now().Add(120*time.Second).After(deadline) { continue }
			if job.State == "running" || job.State == "reconcile" { return errors.New("source call has no settled project observation") }
			worker := h.Workers[job.Kind]
			tried[job.Key] = true
			if worker == nil {
				if err := h.Store.RecordAuxiliaryWait(ref, "controlled auxiliary worker is unavailable"); err != nil { return err }
				waits = append(waits, errors.New("controlled auxiliary worker is unavailable"))
				continue
			}
			measurement, err := worker.Measure(job)
			if err != nil {
				if saveErr := h.Store.RecordAuxiliaryWait(ref, err.Error()); saveErr != nil { return errors.Join(err, saveErr) }
				waits = append(waits, err)
				continue
			}
			call, err := h.Store.ClaimAuxiliaryCall(ref, worker.Identity(), measurement, deadline)
			if err != nil {
				// Claim may have committed an intent. Do not start or retry here;
				// a subsequent host reconciles that exact persisted request.
				return err
			}
			current, err := h.Store.CheckAuxiliaryDispatch(call)
			if err != nil { return err }
			callContext, stop := context.WithTimeout(ctx, 120*time.Second)
			observation, runErr := worker.Start(callContext, current, call)
			stop()
			if observation == nil { return errors.Join(runErr, errors.New("auxiliary result unknown; reconcile the original executor")) }
			if err := h.Store.ObserveAuxiliary(id, *observation); err != nil { return errors.Join(runErr, err) }
			if observation.State == "unknown" || observation.State == "running" { return runErr }
			// At most the source ledger's single additional recovery is allowed.
			delete(tried, job.Key)
			progress = true
			break
		}
		if !progress { return errors.Join(append(waits, registrationErr)... ) }
	}
	return ctx.Err()
}

// AuxiliaryLauncher owns the OS/process boundary. Implementations launch the
// bounded host outside the foreground process lifetime, with no window, using
// the same project Store and fixed Task identity. A goroutine is insufficient.
// Missing production capability/launcher registration is an explicit auxiliary
// gap; callers must not roll back accepted development or substitute raw LLMs.
type AuxiliaryLauncher func(*workflow.Store, workflow.TaskID) error

// StartAuxiliary is called by managed startup even for completed Tasks. It does
// not consume a Work Item or the foreground write lease.
func StartAuxiliary(store *workflow.Store, id workflow.TaskID, launch AuxiliaryLauncher) error {
	state, err := store.Load(id)
	if err != nil { return err }
	if state.Protocol == nil { return nil }
	if state.PendingEvent != nil { if _, err := store.RecoverPendingEvent(id); err != nil { return err } }
	registrationErr := errors.Join(store.RegisterTaskVerifier(id), store.RegisterTaskMemory(id), store.RegisterTaskKnowledge(id))
	if launch == nil { return errors.Join(registrationErr, errors.New("managed detached auxiliary launcher is unavailable")) }
	return errors.Join(registrationErr, launch(store, id))
}
