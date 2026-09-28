package xsync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskGroup(t *testing.T) {
	t.Run("should not start after stop without prior start", func(t *testing.T) {
		var s TaskGroup
		assert.False(t, s.Stop())
		_, err := s.start(context.Background())
		assert.ErrorIs(t, err, ErrTaskGroupStopped)
	})
	t.Run("should not start twice", func(t *testing.T) {
		var s TaskGroup
		_, err := s.start(context.Background())
		require.NoError(t, err)
		_, err = s.start(context.Background())
		assert.ErrorIs(t, err, ErrTaskGroupRunning)
	})
	t.Run("should cancel and wait for tasks when stopped", func(t *testing.T) {
		var s TaskGroup
		ctx, err := s.start(context.Background())
		require.NoError(t, err)
		taskDone := make(chan struct{})
		go func() {
			<-ctx.Done()
			time.Sleep(50 * time.Millisecond)
			close(taskDone)
			s.markStopped()
		}()
		assert.True(t, s.Stop())
		select {
		case <-taskDone:
		default:
			t.Fatal("Stop returned before tasks finished")
		}
		_, err = s.start(context.Background())
		assert.ErrorIs(t, err, ErrTaskGroupStopped)
	})
	t.Run("should allow marking stopped more than once", func(t *testing.T) {
		var s TaskGroup
		_, err := s.start(context.Background())
		require.NoError(t, err)
		s.markStopped()
		assert.NotPanics(t, s.markStopped)
		assert.True(t, stopWithin(t, &s, time.Second))
	})
	t.Run("should ignore marking stopped before start", func(t *testing.T) {
		var s TaskGroup
		assert.NotPanics(t, s.markStopped)
		ctx, err := s.start(context.Background())
		require.NoError(t, err)
		go func() {
			<-ctx.Done()
			s.markStopped()
		}()
		assert.True(t, stopWithin(t, &s, time.Second))
	})
}

func TestTaskGroupRun(t *testing.T) {
	t.Run("should cancel all tasks and wait for them when stopped", func(t *testing.T) {
		var s TaskGroup
		var finished atomic.Int64
		task := func(delay time.Duration) func(context.Context) {
			return func(ctx context.Context) {
				<-ctx.Done()
				time.Sleep(delay)
				finished.Add(1)
			}
		}
		err := s.Run(context.Background(), task(0), task(50*time.Millisecond))
		require.NoError(t, err)
		assert.True(t, stopWithin(t, &s, time.Second))
		assert.EqualValues(t, 2, finished.Load())
	})
	t.Run("should not run tasks after stop", func(t *testing.T) {
		var s TaskGroup
		s.Stop()
		var called atomic.Bool
		err := s.Run(context.Background(), func(context.Context) {
			called.Store(true)
		})
		assert.ErrorIs(t, err, ErrTaskGroupStopped)
		time.Sleep(20 * time.Millisecond)
		assert.False(t, called.Load())
	})
	t.Run("should not run twice", func(t *testing.T) {
		var s TaskGroup
		block := func(ctx context.Context) { <-ctx.Done() }
		require.NoError(t, s.Run(context.Background(), block))
		err := s.Run(context.Background(), block)
		assert.ErrorIs(t, err, ErrTaskGroupRunning)
		assert.True(t, stopWithin(t, &s, time.Second))
	})
	t.Run("should stop when run without tasks", func(t *testing.T) {
		var s TaskGroup
		require.NoError(t, s.Run(context.Background()))
		assert.True(t, stopWithin(t, &s, time.Second))
	})
}

// stopWithin calls Stop and fails the test if it doesn't return within timeout.
func stopWithin(t *testing.T, s *TaskGroup, timeout time.Duration) bool {
	t.Helper()
	result := make(chan bool, 1)
	go func() {
		result <- s.Stop()
	}()
	select {
	case ok := <-result:
		return ok
	case <-time.After(timeout):
		t.Fatal("Stop did not return")
		return false
	}
}
