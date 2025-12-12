package syncqueue_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/syncqueue"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestSyncQueue_Get(t *testing.T) {
	t.Run("should wait until there is an item in the queue", func(t *testing.T) {
		q := syncqueue.New[int]()
		g := new(errgroup.Group)
		ctx := context.Background()
		g.Go(func() error {
			v, err := q.Get(ctx)
			if err != nil {
				return err
			}
			assert.Equal(t, 42, v)
			return nil
		})
		time.Sleep(250 * time.Millisecond)
		q.Put(42)
		err := g.Wait()
		if assert.NoError(t, err) {
			assert.True(t, q.IsEmpty())
		}
	})
	t.Run("should abort while waiting for a new item the queue", func(t *testing.T) {
		q := syncqueue.New[int]()
		g := new(errgroup.Group)
		ctx, cancel := context.WithCancel(context.Background())
		g.Go(func() error {
			_, err := q.Get(ctx)
			if err != nil {
				return err
			}
			return nil
		})
		time.Sleep(10 * time.Millisecond)
		cancel()
		err := g.Wait()
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestSyncQueue_GetNoWait(t *testing.T) {
	t.Run("should return items in FIFO order", func(t *testing.T) {
		q := syncqueue.New[int]()
		q.Put(99)
		q.Put(42)
		v, err := q.GetNoWait()
		if assert.NoError(t, err) {
			assert.Equal(t, 99, v)
		}
		v, err = q.GetNoWait()
		if assert.NoError(t, err) {
			assert.Equal(t, 42, v)
		}
	})
	t.Run("should return specific error when trying to pop from empty queue", func(t *testing.T) {
		q := syncqueue.New[int]()
		_, err := q.GetNoWait()
		assert.ErrorIs(t, syncqueue.ErrEmpty, err)
	})
}

func TestSyncQueue_IsEmpty(t *testing.T) {
	t.Run("should report when the queue is empty", func(t *testing.T) {
		q := syncqueue.New[int]()
		assert.True(t, q.IsEmpty())
	})
	t.Run("should report when the queue is not empty", func(t *testing.T) {
		q := syncqueue.New[int]()
		q.Put(99)
		assert.False(t, q.IsEmpty())
	})
}

func TestSyncQueue_Size(t *testing.T) {
	t.Run("should return queue size when not empty", func(t *testing.T) {
		q := syncqueue.New[int]()
		q.Put(99)
		q.Put(42)
		v := q.Size()
		assert.Equal(t, 2, v)
	})
	t.Run("should return queue size when empty", func(t *testing.T) {
		q := syncqueue.New[int]()
		v := q.Size()
		assert.Equal(t, 0, v)
	})
}
