// Package syncqueue provides queues for concurrent use.
package syncqueue

import (
	"context"
	"errors"
	"sync"
)

var ErrEmpty = errors.New("empty queue")

// SyncQueue represents an unlimited FIFO queue which can be used to synchronize goroutines.
type SyncQueue[T any] struct {
	mu   sync.Mutex
	cond *sync.Cond
	s    []T
}

// New returns a new [SyncQueue].
func New[T any]() *SyncQueue[T] {
	q := &SyncQueue[T]{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// GetNoWait returns the item from the top of the queue or an error when the queue is empty.
func (q *SyncQueue[T]) GetNoWait() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.popLocked()
}

// Get returns an item from the queue or waits until an item is available.
// The operation will be aborted when the provided context is canceled.
//
// If there are multiple goroutines waiting it is undefined which one retrieves a new item.
func (q *SyncQueue[T]) Get(ctx context.Context) (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	stop := context.AfterFunc(ctx, func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.cond.Broadcast()
	})
	defer stop()

	for len(q.s) == 0 {
		if err := ctx.Err(); err != nil {
			var zero T
			return zero, err
		}
		q.cond.Wait()
	}

	return q.popLocked()
}

func (q *SyncQueue[T]) popLocked() (T, error) {
	if len(q.s) == 0 {
		var zero T
		return zero, ErrEmpty
	}
	v := q.s[0]
	var zero T
	q.s[0] = zero // avoid memory leak / retain reference
	q.s = q.s[1:]

	if len(q.s) == 0 {
		q.s = nil // reset backing array allocation to prevent memory leak in slice
	}
	return v, nil
}

// IsEmpty reports whether the queue is empty.
func (q *SyncQueue[T]) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.s) == 0
}

// Put adds an item in the queue.
func (q *SyncQueue[T]) Put(v T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.s = append(q.s, v)
	q.cond.Signal()
}

// Size returns the number of items in the queue.
func (q *SyncQueue[T]) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.s)
}
