package main

import (
	"context"
	"errors"
	"sync"
)

type appSlot struct {
	app *appProcess
}

type appPool struct {
	config Config
	slots  chan *appSlot
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

var errAppProcessCleanup = errors.New("app-server process cleanup failed")

func newAppPool(c Config) *appPool {
	p := &appPool{config: c, slots: make(chan *appSlot, c.GlobalConcurrency), done: make(chan struct{})}
	for range c.GlobalConcurrency { p.slots <- &appSlot{} }
	return p
}

func (p *appPool) acquire(ctx context.Context) (*appSlot, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.done:
		return nil, errors.New("app-server pool is closed")
	case slot := <-p.slots:
		p.mu.Lock(); closed := p.closed; p.mu.Unlock()
		if closed { p.release(slot, false); return nil, errors.New("app-server pool is closed") }
		if slot.app != nil {
			if err := slot.app.verifySafety(ctx); err != nil { if p.release(slot, false) != nil { return nil, errAppProcessCleanup }; return nil, err }
			return slot, nil
		}
		for attempt := 0; attempt < 2; attempt++ {
			app, err := startAppProcess(p.config)
			if err != nil {
				if attempt == 0 { continue }
				p.release(slot, false)
				return nil, err
			}
			if err = app.preflight(ctx); err != nil {
				stopErr := app.stop()
				if p.release(slot, false) != nil || stopErr != nil { return nil, errAppProcessCleanup }
				return nil, err
			}
			slot.app = app
			return slot, nil
		}
		p.release(slot, false)
		return nil, errors.New("app-server startup failed")
	}
}

func (p *appPool) release(slot *appSlot, healthy bool) error {
	var stopErr error
	if !healthy && slot.app != nil { stopErr = slot.app.stop(); slot.app = nil }
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		if slot.app != nil { stopErr = slot.app.stop(); slot.app = nil }
		return stopErr
	}
	p.slots <- slot
	p.mu.Unlock()
	return stopErr
}

func (p *appPool) close() error {
	p.mu.Lock()
	if p.closed { p.mu.Unlock(); return nil }
	p.closed = true
	close(p.done)
	p.mu.Unlock()
	var stopping []*appSlot
	for {
		select {
		case slot := <-p.slots:
			stopping = append(stopping, slot)
		default:
			var wait sync.WaitGroup
			failures := make(chan struct{}, len(stopping))
			for _, slot := range stopping {
				if slot.app == nil { continue }
				wait.Add(1)
				go func(slot *appSlot) { defer wait.Done(); if slot.app.stop() != nil { failures <- struct{}{} }; slot.app = nil }(slot)
			}
			wait.Wait()
			close(failures)
			if len(failures) > 0 { return errAppProcessCleanup }
			return nil
		}
	}
}
