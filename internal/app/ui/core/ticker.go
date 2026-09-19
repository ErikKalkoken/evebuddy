package core

import (
	"context"
	"sync"
	"time"
)

// cancelableTicker runs a periodic background task that can be canceled and waited on.
type cancelableTicker struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// Start launches fire on a ticker of interval d, until Stop is called. If immediate
// is true, fire also runs once right away, before the first tick.
func (t *cancelableTicker) Start(d time.Duration, immediate bool, fire func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	t.mu.Lock()
	t.cancel = cancel
	t.done = done
	t.mu.Unlock()
	go func() {
		defer close(done)
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
	}()
}

// Stop cancels the ticker and waits for it to finish. Safe to call even if Start was never called.
func (t *cancelableTicker) Stop() {
	t.mu.Lock()
	cancel := t.cancel
	done := t.done
	t.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}
