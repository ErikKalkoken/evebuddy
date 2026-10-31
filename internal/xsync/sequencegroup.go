package xsync

import (
	"sync"
)

// SequenceGroup runs functions one after another in a background goroutine,
// in the order they were added. The zero value is ready to use.
type SequenceGroup struct {
	mu      sync.Mutex
	idle    *sync.Cond // signaled when the worker stops; created on first use
	jobs    []func()
	running bool
}

// Go queues f to run after all previously queued functions.
func (g *SequenceGroup) Go(f func()) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.jobs = append(g.jobs, f)
	if !g.running {
		g.running = true // set under lock so only one worker is started
		go g.work()
	}
}

// Wait blocks until no functions are queued or running.
// It is safe to call concurrently with Go.
func (g *SequenceGroup) Wait() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.idle == nil {
		g.idle = sync.NewCond(&g.mu) // lazy so the zero value stays usable
	}
	for g.running {
		g.idle.Wait()
	}
}

func (g *SequenceGroup) work() {
	for {
		g.mu.Lock()
		if len(g.jobs) == 0 {
			g.running = false
			if g.idle != nil {
				g.idle.Broadcast()
			}
			g.mu.Unlock()
			return
		}
		f := g.jobs[0]
		g.jobs[0] = nil
		g.jobs = g.jobs[1:]
		g.mu.Unlock()
		f()
	}
}
