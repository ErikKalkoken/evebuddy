package xsync

import (
	"context"
	"sync"
	"time"
)

// BackgroundGroup tracks a set of cancelable background goroutines sharing one
// lifetime: a ctx that is canceled by Stop, and a WaitGroup that Stop waits on.
// The zero value is ready to use.
type BackgroundGroup struct {
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (g *BackgroundGroup) init() {
	g.once.Do(func() {
		g.ctx, g.cancel = context.WithCancel(context.Background())
	})
}

// Context returns the group's ctx, valid for the group's entire lifetime until
// Stop cancels it.
func (g *BackgroundGroup) Context() context.Context {
	g.init()
	return g.ctx
}

// Go runs f in a tracked goroutine.
func (g *BackgroundGroup) Go(f func()) {
	g.init()
	g.wg.Go(f)
}

// StartTicker runs fire on a ticker of interval d, tracked by the group, until
// Stop is called. If immediate is true, fire also runs once right away, before
// the first tick.
func (g *BackgroundGroup) StartTicker(d time.Duration, immediate bool, fire func(ctx context.Context)) {
	ctx := g.Context()
	g.Go(func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		if immediate {
			fire(ctx)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fire(ctx)
			}
		}
	})
}

// Wait blocks until all currently tracked goroutines have finished, without
// canceling the group's ctx.
func (g *BackgroundGroup) Wait() {
	g.wg.Wait()
}

// Stop cancels the group's ctx and waits for all tracked goroutines to finish.
// It is safe to call even if nothing was ever started.
func (g *BackgroundGroup) Stop() {
	g.init()
	g.cancel()
	g.wg.Wait()
}
