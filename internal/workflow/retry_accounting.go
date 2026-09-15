package workflow

import "fmt"

// ConsumeOperationalRetry records one retry for a recovery, notification, or
// delivery operation.  The update is event-backed, so an interrupted caller
// can recover the state transition without replaying its external operation.
// It never reads or changes Work Item Attempt counters.
func (s *Store) ConsumeOperationalRetry(id TaskID, kind OperationalRetryKind) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "retry." + string(kind) + ".consumed", Detail: string(kind)}, func(state *RuntimeState) error {
		counter, err := retryCounter(&state.OperationalRetries, kind)
		if err != nil {
			return err
		}
		if counter.Used >= counter.MaxAttempts {
			return fmt.Errorf("%s retry limit of %d is exhausted", kind, counter.MaxAttempts)
		}
		counter.Used++
		return nil
	})
}

// ResetOperationalRetries begins a new independently-accounted operational
// cycle.  It is intentionally explicit: completing a Work Item does not erase
// delivery, recovery, or notification audit history.
func (s *Store) ResetOperationalRetries(id TaskID, kind OperationalRetryKind) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "retry." + string(kind) + ".reset", Detail: string(kind)}, func(state *RuntimeState) error {
		counter, err := retryCounter(&state.OperationalRetries, kind)
		if err != nil {
			return err
		}
		counter.Used = 0
		return nil
	})
}

// SetOperationalRetryLimit changes one task-level category without widening a
// Work Item policy.  Limits follow the same small, auditable 1..5 range as
// implementation Attempts.
func (s *Store) SetOperationalRetryLimit(id TaskID, kind OperationalRetryKind, limit int) (RuntimeState, error) {
	if limit < MinRetryLimit || limit > MaxRetryLimit {
		return RuntimeState{}, fmt.Errorf("retry limit must be between %d and %d", MinRetryLimit, MaxRetryLimit)
	}
	return s.UpdateWithEvent(id, Event{Type: "retry." + string(kind) + ".limit-set", Detail: string(kind)}, func(state *RuntimeState) error {
		counter, err := retryCounter(&state.OperationalRetries, kind)
		if err != nil {
			return err
		}
		if limit < counter.Used {
			return fmt.Errorf("%s retry limit %d is below already-used retries %d", kind, limit, counter.Used)
		}
		counter.MaxAttempts = limit
		return nil
	})
}

func normalizeOperationalRetries(accounting *OperationalRetryAccounting) error {
	for _, counter := range []*RetryCounter{&accounting.Recovery, &accounting.Notification, &accounting.Delivery} {
		if counter.MaxAttempts == 0 {
			counter.MaxAttempts = DefaultRetryLimit
		}
		if counter.MaxAttempts < MinRetryLimit || counter.MaxAttempts > MaxRetryLimit {
			return fmt.Errorf("retry limit must be between %d and %d", MinRetryLimit, MaxRetryLimit)
		}
		if counter.Used < 0 || counter.Used > counter.MaxAttempts {
			return fmt.Errorf("retry usage must be between 0 and its retry limit")
		}
	}
	return nil
}

func retryCounter(accounting *OperationalRetryAccounting, kind OperationalRetryKind) (*RetryCounter, error) {
	switch kind {
	case RetryRecovery:
		return &accounting.Recovery, nil
	case RetryNotification:
		return &accounting.Notification, nil
	case RetryDelivery:
		return &accounting.Delivery, nil
	default:
		return nil, fmt.Errorf("unsupported operational retry kind: %s", kind)
	}
}
