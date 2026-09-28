package xsync_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

func TestTaskGroupRun(t *testing.T) {
	t.Run("should cancel all tasks and wait for them when stopped", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		var finished atomic.Int64
		task := func(delay time.Duration) func(context.Context) {
			return func(ctx context.Context) {
				<-ctx.Done()
				time.Sleep(delay)
				finished.Add(1)
			}
		}
		require.NoError(t, g.Run(task(0), task(50*time.Millisecond)))
		assert.True(t, stopWithin(t, g, time.Second))
		assert.EqualValues(t, 2, finished.Load())
	})
	t.Run("should not run twice", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		block := func(ctx context.Context) { <-ctx.Done() }
		require.NoError(t, g.Run(block))
		assert.ErrorIs(t, g.Run(block), xsync.ErrTaskGroupRunning)
		assert.True(t, stopWithin(t, g, time.Second))
	})
	t.Run("should not run tasks after stop", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		g.Stop()
		var called atomic.Bool
		err := g.Run(func(context.Context) {
			called.Store(true)
		})
		assert.ErrorIs(t, err, xsync.ErrTaskGroupStopped)
		time.Sleep(20 * time.Millisecond)
		assert.False(t, called.Load())
	})
	t.Run("should keep running without tasks until stopped", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		require.NoError(t, g.Run())
		assert.ErrorIs(t, g.Run(), xsync.ErrTaskGroupRunning)
		assert.True(t, stopWithin(t, g, time.Second))
	})
}

func TestTaskGroupGo(t *testing.T) {
	t.Run("should run job", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		done := make(chan struct{})
		assert.True(t, g.Go(func(context.Context) {
			close(done)
		}))
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("job did not run")
		}
		stopWithin(t, g, time.Second)
	})
	t.Run("should allow run after go", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		block := func(ctx context.Context) { <-ctx.Done() }
		assert.True(t, g.Go(block))
		require.NoError(t, g.Run(block))
		assert.True(t, stopWithin(t, g, time.Second))
	})
	t.Run("should cancel jobs and wait for them when stopped", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		var finished atomic.Bool
		g.Go(func(ctx context.Context) {
			<-ctx.Done()
			time.Sleep(50 * time.Millisecond)
			finished.Store(true)
		})
		stopWithin(t, g, time.Second)
		assert.True(t, finished.Load())
	})
	t.Run("should not run job after stop", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		g.Stop()
		var called atomic.Bool
		assert.False(t, g.Go(func(context.Context) {
			called.Store(true)
		}))
		time.Sleep(20 * time.Millisecond)
		assert.False(t, called.Load())
	})
}

func TestTaskGroupStop(t *testing.T) {
	t.Run("should report running only once after run", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		require.NoError(t, g.Run())
		assert.True(t, stopWithin(t, g, time.Second))
		assert.False(t, stopWithin(t, g, time.Second))
	})
	t.Run("should report not running without run", func(t *testing.T) {
		g := xsync.NewTaskGroup(context.Background())
		g.Go(func(context.Context) {})
		assert.False(t, stopWithin(t, g, time.Second))
	})
	t.Run("should stop when parent context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		g := xsync.NewTaskGroup(ctx)
		cancel()
		assert.False(t, g.Go(func(context.Context) {}))
		assert.ErrorIs(t, g.Run(), xsync.ErrTaskGroupStopped)
	})
}

// stopWithin calls Stop and fails the test if it doesn't return within timeout.
func stopWithin(t *testing.T, g *xsync.TaskGroup, timeout time.Duration) bool {
	t.Helper()
	result := make(chan bool, 1)
	go func() {
		result <- g.Stop()
	}()
	select {
	case ok := <-result:
		return ok
	case <-time.After(timeout):
		t.Fatal("Stop did not return")
		return false
	}
}
