package xsync

import (
	"context"
	"time"
)

// Throttle runs a function never overlapping and with at least an interval between runs.
// Triggers while a run is running or waiting are merged into one run.
// Use [NewThrottle] to create one and Stop to release it.
type Throttle struct {
	trigger chan struct{}
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewThrottle returns a throttle that runs f with at least interval d between runs.
func NewThrottle(d time.Duration, f func()) *Throttle {
	ctx, cancel := context.WithCancel(context.Background())
	t := &Throttle{trigger: make(chan struct{}, 1), cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(t.done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.trigger:
			}
			if ctx.Err() != nil {
				return // select picks randomly when both are ready
			}
			f()
			select {
			case <-ctx.Done():
				return
			case <-time.After(d):
			}
		}
	}()
	return t
}

// Trigger requests a run of f. Does nothing after Stop.
func (t *Throttle) Trigger() {
	select {
	case t.trigger <- struct{}{}:
	default: // a run is already pending
	}
}

// Stop cancels a pending run and waits for a running one to finish.
// Safe to call more than once. Must not be called from f.
func (t *Throttle) Stop() {
	t.cancel()
	<-t.done
}
