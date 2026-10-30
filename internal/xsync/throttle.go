package xsync

import (
	"sync"
	"time"
)

// Throttle runs a function never overlapping and with at least an interval between runs.
// Triggers while a run is running or waiting are merged into one run.
// Use [NewThrottle] to create one.
type Throttle struct {
	d time.Duration
	f func()

	mu      sync.Mutex
	lastRun time.Time
	pending bool
	running bool
	stopped bool
	timer   *time.Timer
	wg      sync.WaitGroup
}

// NewThrottle returns a throttle that runs f with at least interval d between runs.
func NewThrottle(d time.Duration, f func()) *Throttle {
	return &Throttle{d: d, f: f}
}

// Trigger requests a run of f. Does nothing after Stop.
func (t *Throttle) Trigger() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped || t.pending {
		return
	}
	t.pending = true
	if t.running {
		return // run schedules the next one when done
	}
	t.schedule()
}

// schedule starts the timer for the next run. Must be called with mu held.
func (t *Throttle) schedule() {
	wait := max(t.d-time.Since(t.lastRun), 0)
	t.wg.Add(1)
	t.timer = time.AfterFunc(wait, t.run)
}

func (t *Throttle) run() {
	defer t.wg.Done()
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return
	}
	t.pending = false
	t.running = true
	t.mu.Unlock()

	t.f()

	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
	t.lastRun = time.Now()
	if t.pending && !t.stopped {
		t.schedule()
	}
}

// Stop cancels a pending run and waits for a running one to finish.
// Later triggers are ignored. Must not be called from f.
func (t *Throttle) Stop() {
	t.mu.Lock()
	t.stopped = true
	if t.timer != nil && t.timer.Stop() {
		t.wg.Done() // run will not be called for this timer
	}
	t.mu.Unlock()
	t.wg.Wait()
}
