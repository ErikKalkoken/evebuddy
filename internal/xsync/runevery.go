package xsync

import (
	"context"
	"time"
)

// RunEvery runs f at once and then every d until ctx is canceled.
// Runs never overlap.
func RunEvery(ctx context.Context, d time.Duration, f func(context.Context)) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		f(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
