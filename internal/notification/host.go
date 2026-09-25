package notification

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"aiw/internal/workflow"
)

// Run shares E05's exclusive helper lock and bounded lifetime, but never its
// model recovery counters. Waiting-only work exits; due network retries wait
// until the original persisted time. No foreground acceptance waits here.
func Run(ctx context.Context, store *workflow.Store, id workflow.TaskID) error {
	dispatcher := PluginDispatcher{ProjectRoot: filepath.Dir(store.Root)}
	tried := map[string]bool{}
	var failures []error
	for processed := 0; processed < 32 && ctx.Err() == nil; {
		state, err := store.Load(id)
		if err != nil { return err }
		if state.PendingEvent != nil { return errors.New("notification source commit requires reconciliation") }
		progress := false
		var earliest time.Time
		for _, n := range state.Notifications {
			if n.Managed == nil { continue }
			if n.State == workflow.NotificationDispatching {
				// The exclusive helper lock means an older host no longer owns
				// this reservation. Its external outcome remains unknown.
				if _, err := store.RecoverNotificationDispatch(id, n.ID); err != nil { return err }
				progress = true
				break
			}
			if state.Protocol != nil && state.Protocol.Stop != nil { continue }
			if n.State != workflow.NotificationPending && n.State != "waiting" && n.State != workflow.NotificationFailed { continue }
			if n.DispatchAttempts >= 2 || (n.State == workflow.NotificationFailed && n.Managed.LastOutcome != "network_failure") { continue }
			key := fmt.Sprintf("%s:%d", n.ID, len(n.Managed.Attempts))
			if tried[key] { continue }
			if n.Managed.RetryDueAt != "" {
				due, err := time.Parse(time.RFC3339, n.Managed.RetryDueAt)
				if err != nil { continue }
				if time.Now().Before(due) {
					if earliest.IsZero() || due.Before(earliest) { earliest = due }
					continue
				}
			}
			tried[key] = true
			if _, err := store.SendNotification(ctx, id, n.ID, dispatcher); err != nil { failures = append(failures, err) }
			processed++
			progress = true
			break
		}
		if progress { continue }
		if earliest.IsZero() { return errors.Join(failures...) }
		// Re-read Stop/config periodically without moving the saved due time.
		wait := time.Until(earliest)
		if wait > time.Second { wait = time.Second }
		timer := time.NewTimer(wait)
		select { case <-ctx.Done(): timer.Stop(); return errors.Join(append(failures, ctx.Err())...); case <-timer.C: }
	}
	return errors.Join(append(failures, ctx.Err())...)
}
