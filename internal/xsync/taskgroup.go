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

// A TaskGroup manages the starting and stopping of goroutines.
// Start it with Run and stop it with Stop.
// Once stopped it can't be started again.
// The zero value is ready to use.
type TaskGroup struct {
	mu      sync.Mutex
	cancel  func()
	done    chan struct{}
	stopped bool
}

// Run starts the task group and runs each task in its own goroutine.
// Tasks must return when ctx is canceled. Stop waits for all of them.
func (g *TaskGroup) Run(ctx context.Context, tasks ...func(context.Context)) error {
	ctx, err := g.start(ctx)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Go(func() {
			t(ctx)
		})
	}
	go func() {
		wg.Wait()
		g.markStopped()
	}()
	return nil
}

// start starts the task group and returns the context for its tasks.
func (g *TaskGroup) start(ctx context.Context) (context.Context, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopped {
		return nil, ErrTaskGroupStopped
	}
	if g.cancel != nil {
		return nil, ErrTaskGroupRunning
	}
	g.done = make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)
	g.cancel = cancel
	return ctx, nil
}

// markStopped reports that all tasks have stopped.
// It is a no-op before start or when already marked.
func (g *TaskGroup) markStopped() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.done == nil {
		return
	}
	select {
	case <-g.done:
		return // already marked
	default:
	}
	close(g.done)
	g.cancel()
}

// Stop stops the task group, waits for its tasks to finish
// and reports whether it was running.
// Stop before Run also prevents any later Run.
func (g *TaskGroup) Stop() bool {
	g.mu.Lock()
	g.stopped = true
	cancel := g.cancel
	done := g.done
	g.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	<-done
	return true
}
