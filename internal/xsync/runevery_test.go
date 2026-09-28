package xsync_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

func TestRunEvery(t *testing.T) {
	t.Run("should run at once before first tick", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ran := make(chan struct{}, 1)
		go xsync.RunEvery(ctx, time.Hour, func(context.Context) {
			select {
			case ran <- struct{}{}:
			default:
			}
		})
		select {
		case <-ran:
		case <-time.After(time.Second):
			t.Fatal("f did not run at once")
		}
	})
	t.Run("should run repeatedly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var count atomic.Int64
		go xsync.RunEvery(ctx, 10*time.Millisecond, func(context.Context) {
			count.Add(1)
		})
		assert.Eventually(t, func() bool {
			return count.Load() >= 3
		}, time.Second, 5*time.Millisecond)
	})
	t.Run("should return when canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			xsync.RunEvery(ctx, time.Hour, func(context.Context) {})
			close(done)
		}()
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("RunEvery did not return after cancel")
		}
	})
	t.Run("should not overlap runs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var running, overlaps, count atomic.Int64
		done := make(chan struct{})
		go func() {
			xsync.RunEvery(ctx, 5*time.Millisecond, func(context.Context) {
				if running.Add(1) > 1 {
					overlaps.Add(1)
				}
				time.Sleep(20 * time.Millisecond)
				running.Add(-1)
				count.Add(1)
			})
			close(done)
		}()
		assert.Eventually(t, func() bool {
			return count.Load() >= 3
		}, time.Second, 5*time.Millisecond)
		cancel()
		<-done
		assert.Zero(t, overlaps.Load())
	})
}
