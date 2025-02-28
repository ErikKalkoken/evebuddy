package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/queue"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestQueue(t *testing.T) {
	t.Run("should return items in FIFO order", func(t *testing.T) {
		q := queue.New[int]()
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
		q := queue.New[int]()
		_, err := q.GetNoWait()
		assert.ErrorIs(t, queue.ErrEmpty, err)
	})
	t.Run("should return correct queue size", func(t *testing.T) {
		q := queue.New[int]()
		q.Put(99)
		q.Put(42)
		v := q.Size()
		assert.Equal(t, 2, v)
	})
	t.Run("should report wether the queue is empty", func(t *testing.T) {
		q := queue.New[int]()
		assert.True(t, q.IsEmpty())
		q.Put(99)
		assert.False(t, q.IsEmpty())
	})
	t.Run("should wait until there is an item in the queue", func(t *testing.T) {
		q := queue.New[int]()
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
		q := queue.New[int]()
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
