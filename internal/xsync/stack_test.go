package xsync_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

func TestStack_Pop(t *testing.T) {
	t.Run("should return items in LIFO order", func(t *testing.T) {
		st := new(xsync.Stack[int])
		st.Push(99)
		st.Push(42)

		v, ok := st.Pop()
		require.True(t, ok)
		assert.Equal(t, 42, v)
		v, ok = st.Pop()
		require.True(t, ok)
		assert.Equal(t, 99, v)
	})

	t.Run("should return false and zero value when trying to pop from empty stack", func(t *testing.T) {
		st := new(xsync.Stack[int])
		v, ok := st.Pop()
		assert.False(t, ok)
		assert.Zero(t, v)
	})
}

func TestStack_Size(t *testing.T) {
	t.Run("should return stack size when not empty", func(t *testing.T) {
		st := new(xsync.Stack[int])
		st.Push(99)
		st.Push(42)
		assert.Equal(t, 2, st.Size())
	})

	t.Run("should return stack size when empty", func(t *testing.T) {
		st := new(xsync.Stack[int])
		assert.Equal(t, 0, st.Size())
	})
}

func TestStack_Lifecycle(t *testing.T) {
	t.Run("should accurately reflect Size during push/pop lifecycle", func(t *testing.T) {
		st := new(xsync.Stack[string])
		require.Equal(t, 0, st.Size())

		st.Push("first")
		st.Push("second")
		require.Equal(t, 2, st.Size())

		v1, ok := st.Pop()
		require.True(t, ok)
		assert.Equal(t, "second", v1)
		require.Equal(t, 1, st.Size())

		v2, ok := st.Pop()
		require.True(t, ok)
		assert.Equal(t, "first", v2)
		require.Equal(t, 0, st.Size())

		// Extra pop should fail
		_, ok = st.Pop()
		require.False(t, ok)
	})
}

func TestStack_Nil(t *testing.T) {
	t.Run("should do nothing when stack is nil", func(t *testing.T) {
		var st *xsync.Stack[int]
		st.Push(42)
		v, ok := st.Pop()
		assert.False(t, ok)
		assert.Zero(t, v)
		assert.Equal(t, 0, st.Size())
	})
}

func TestStack_Concurrent(t *testing.T) {
	t.Run("should handle concurrent pushes and pops safely", func(t *testing.T) {
		const goroutines = 100
		const itemsPerGoroutine = 100
		const total = goroutines * itemsPerGoroutine

		st := new(xsync.Stack[int])

		// Concurrently push items while calling Size
		var pushers sync.WaitGroup
		for i := range goroutines {
			pushers.Go(func() {
				base := i * itemsPerGoroutine
				for j := range itemsPerGoroutine {
					st.Push(base + j)
				}
			})
		}
		done := make(chan struct{})
		var reader sync.WaitGroup
		reader.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = st.Size()
				}
			}
		})
		pushers.Wait()
		close(done)
		reader.Wait()
		require.Equal(t, total, st.Size())

		// Concurrently pop all items
		popped := make(chan int, total)
		var poppers sync.WaitGroup
		for range goroutines {
			poppers.Go(func() {
				for range itemsPerGoroutine {
					if v, ok := st.Pop(); ok {
						popped <- v
					}
				}
			})
		}
		poppers.Wait()
		close(popped)

		// Verify every pushed item was popped exactly once
		seen := make(map[int]int, total)
		for v := range popped {
			seen[v]++
		}
		assert.Len(t, seen, total)
		for v := range total {
			assert.Equal(t, 1, seen[v], "item %d", v)
		}
		assert.Equal(t, 0, st.Size())

		// Ensure popping on empty stack reports no item
		_, ok := st.Pop()
		require.False(t, ok)
	})
}
