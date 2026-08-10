package syncqueue_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/syncqueue"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

type result struct {
	value int
	err   error
}

func TestSyncQueue_Get(t *testing.T) {
	t.Run("should wait until there is an item in the queue", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			q := syncqueue.New[int]()
			c := make(chan result)
			go func() {
				v, err := q.Get(t.Context())
				c <- result{
					value: v,
					err:   err,
				}
			}()
			q.Put(42)
			synctest.Wait()
			r := <-c
			require.NoError(t, r.err)
			xassert.Equal(t, 42, r.value)
		})
	})
	t.Run("should abort waiting for a new item when context is canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			q := syncqueue.New[int]()
			c := make(chan result)
			ctx, cancel := context.WithCancel(t.Context())
			go func() {
				v, err := q.Get(ctx)
				c <- result{
					value: v,
					err:   err,
				}
			}()
			cancel()
			synctest.Wait()
			r := <-c
			assert.ErrorIs(t, r.err, context.Canceled)
		})
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

func TestSyncQueue_Get_ContextPreCanceled(t *testing.T) {
	t.Run("should fail immediately if context is already canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			q := syncqueue.New[int]()
			ctx, cancel := context.WithCancel(t.Context())
			cancel() // Pre-cancel before calling Get

			v, err := q.Get(ctx)
			require.ErrorIs(t, err, context.Canceled)
			assert.Zero(t, v)
		})
	})
}

func TestSyncQueue_Get_MultipleWaiters(t *testing.T) {
	t.Run("should wake up the correct waiter and leave others waiting", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			q := syncqueue.New[int]()

			resCh1 := make(chan result, 1)
			resCh2 := make(chan result, 1)

			ctx1, cancel1 := context.WithCancel(t.Context())
			defer cancel1()

			// Start Waiter 1
			go func() {
				v, err := q.Get(ctx1)
				resCh1 <- result{value: v, err: err}
			}()

			// Start Waiter 2
			go func() {
				v, err := q.Get(t.Context())
				resCh2 <- result{value: v, err: err}
			}()

			synctest.Wait()

			// Cancel Waiter 1 specifically
			cancel1()
			synctest.Wait()

			r1 := <-resCh1
			require.ErrorIs(t, r1.err, context.Canceled)

			// Waiter 2 should still be waiting until an item is pushed
			q.Put(100)
			synctest.Wait()

			r2 := <-resCh2
			require.NoError(t, r2.err)
			assert.Equal(t, 100, r2.value)
		})
	})

	t.Run("should fulfill multiple waiters sequentially as items are added", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			q := syncqueue.New[int]()
			results := make(chan int, 2)

			for range 2 {
				go func() {
					v, err := q.Get(t.Context())
					if err == nil {
						results <- v
					}
				}()
			}

			synctest.Wait()

			q.Put(1)
			q.Put(2)
			synctest.Wait()

			require.Len(t, results, 2)
			val1 := <-results
			val2 := <-results
			assert.ElementsMatch(t, []int{1, 2}, []int{val1, val2})
		})
	})
}

func TestSyncQueue_LifecycleAndMemoryReset(t *testing.T) {
	t.Run("should accurately reflect Size and IsEmpty during push/pop lifecycle", func(t *testing.T) {
		q := syncqueue.New[string]()
		require.True(t, q.IsEmpty())
		require.Equal(t, 0, q.Size())

		q.Put("first")
		q.Put("second")
		require.False(t, q.IsEmpty())
		require.Equal(t, 2, q.Size())

		v1, err := q.GetNoWait()
		require.NoError(t, err)
		assert.Equal(t, "first", v1)
		require.Equal(t, 1, q.Size())

		v2, err := q.GetNoWait()
		require.NoError(t, err)
		assert.Equal(t, "second", v2)

		require.True(t, q.IsEmpty())
		require.Equal(t, 0, q.Size())

		// Extra pop should fail
		_, err = q.GetNoWait()
		require.ErrorIs(t, err, syncqueue.ErrEmpty)
	})
}
