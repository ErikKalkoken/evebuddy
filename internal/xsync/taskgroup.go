package xsync

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrTaskGroupRunning = errors.New("task group already running")
	ErrTaskGroupStopped = errors.New("task group stopped")
)

// A TaskGroup runs goroutines that are canceled and awaited together.
// Long-running tasks are started with Run and one-off jobs with Go.
// Once stopped it can't be started again.
// Use [NewTaskGroup] to create one.
type TaskGroup struct {
	mu      sync.Mutex
	ctx     context.Context
	cancel  func()
	wg      sync.WaitGroup
	running bool
}

// NewTaskGroup returns a task group whose goroutines are canceled with ctx or Stop.
func NewTaskGroup(ctx context.Context) *TaskGroup {
	ctx, cancel := context.WithCancel(ctx)
	return &TaskGroup{ctx: ctx, cancel: cancel}
}

// Run starts the long-running tasks, each in its own goroutine.
// Tasks must return when ctx is canceled.
// Returns [ErrTaskGroupStopped] after Stop and [ErrTaskGroupRunning] when called twice.
func (g *TaskGroup) Run(tasks ...func(context.Context)) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ctx.Err() != nil {
		return ErrTaskGroupStopped
	}
	if g.running {
		return ErrTaskGroupRunning
	}
	g.running = true
	for _, t := range tasks {
		g.wg.Go(func() {
			t(g.ctx)
		})
	}
	return nil
}

// Go starts a one-off job and reports whether it was started.
// The job must return when ctx is canceled.
// Can be used before, during or after Run, but not after Stop.
func (g *TaskGroup) Go(job func(context.Context)) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ctx.Err() != nil {
		return false
	}
	g.wg.Go(func() {
		job(g.ctx)
	})
	return true
}

// Stop cancels all tasks and jobs and waits for them to finish.
// Reports whether Run had been called, so callers can log only when it was.
func (g *TaskGroup) Stop() bool {
	g.mu.Lock()
	wasRunning := g.running
	g.running = false
	g.cancel()
	g.mu.Unlock()
	g.wg.Wait()
	return wasRunning
}
