// Package syncqueue provides queues for cuncurrent use.
package syncqueue

import (
	"context"
	"errors"
	"slices"
	"sync"
)

var ErrEmpty = errors.New("empty queue")

// SyncQueue represents an unlimited FIFO queue which can be used to synchronize goroutines.
type SyncQueue[T any] struct {
	cmu  sync.Mutex
	cond *sync.Cond

	smu sync.RWMutex
	s   []T // new items are inserted at the beginning of the slice
}

// New returns a new [SyncQueue].
func New[T any]() *SyncQueue[T] {
	q := &SyncQueue[T]{s: make([]T, 0)}
	q.cond = sync.NewCond(&q.cmu)
	return q
}

// GetNoWait returns the item from the top of the queue or an error when the queue is empty.
func (q *SyncQueue[T]) GetNoWait() (T, error) {
	var v T
	q.smu.Lock()
	defer q.smu.Unlock()
	if len(q.s) == 0 {
		return v, ErrEmpty
	}
	v = q.s[len(q.s)-1]
	q.s = q.s[:len(q.s)-1]
	return v, nil
}

// Get returns an item from the queue or waits until an item is available.
// The operation will be aborted when the provided context is canceled.
//
// If there are multiple goroutines waiting it is undefined which one retrives a new item.
func (q *SyncQueue[T]) Get(ctx context.Context) (T, error) {
	stopf := context.AfterFunc(ctx, func() {
		q.cond.L.Lock()
		defer q.cond.L.Unlock()
		q.cond.Broadcast()
	})
	defer stopf()

	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	for {
		v, err := q.GetNoWait()
		if err == nil {
			return v, nil
		}
		q.cond.Wait()
		if ctx.Err() != nil {
			var x T
			return x, ctx.Err()
		}
	}
}

// IsEmpty reports wether the queue is empty.
func (q *SyncQueue[T]) IsEmpty() bool {
	return q.Size() == 0
}

// Put adds an item in the queue.
func (q *SyncQueue[T]) Put(v T) {
	q.smu.Lock()
	defer q.smu.Unlock()
	q.s = slices.Insert(q.s, 0, v)
	q.cond.Signal()
}

// Size returns the number of items in the queue.
func (q *SyncQueue[T]) Size() int {
	q.smu.RLock()
	defer q.smu.RUnlock()
	return len(q.s)
}
