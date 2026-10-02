package colonysim

import (
	"container/heap"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventQueue(t *testing.T) {
	popAll := func(q *eventQueue) []event {
		var got []event
		for q.Len() > 0 {
			got = append(got, heap.Pop(q).(event))
		}
		return got
	}
	t.Run("should return events ordered by time", func(t *testing.T) {
		var q eventQueue
		e1 := event{time: t0, pinID: 1, seq: 3}
		e2 := event{time: t0.Add(time.Minute), pinID: 2, seq: 1}
		e3 := event{time: t0.Add(time.Hour), pinID: 3, seq: 2}
		heap.Push(&q, e3)
		heap.Push(&q, e1)
		heap.Push(&q, e2)
		assert.Equal(t, []event{e1, e2, e3}, popAll(&q))
	})
	t.Run("should return events at the same time in insertion order", func(t *testing.T) {
		var q eventQueue
		e1 := event{time: t0, pinID: 3, seq: 1}
		e2 := event{time: t0, pinID: 1, seq: 2}
		e3 := event{time: t0, pinID: 2, seq: 3}
		heap.Push(&q, e3)
		heap.Push(&q, e1)
		heap.Push(&q, e2)
		assert.Equal(t, []event{e1, e2, e3}, popAll(&q))
	})
	t.Run("should order by time before sequence", func(t *testing.T) {
		var q eventQueue
		e1 := event{time: t0, pinID: 1, seq: 2}
		e2 := event{time: t0.Add(time.Second), pinID: 2, seq: 1}
		heap.Push(&q, e2)
		heap.Push(&q, e1)
		assert.Equal(t, []event{e1, e2}, popAll(&q))
	})
	t.Run("should allow pushing while popping", func(t *testing.T) {
		var q eventQueue
		e1 := event{time: t0, pinID: 1, seq: 1}
		e2 := event{time: t0.Add(time.Minute), pinID: 2, seq: 2}
		e3 := event{time: t0.Add(30 * time.Second), pinID: 3, seq: 3}
		heap.Push(&q, e1)
		heap.Push(&q, e2)
		assert.Equal(t, e1, heap.Pop(&q))
		heap.Push(&q, e3)
		assert.Equal(t, []event{e3, e2}, popAll(&q))
	})
	t.Run("should be empty when new", func(t *testing.T) {
		var q eventQueue
		assert.Equal(t, 0, q.Len())
	})
}

func TestEventQueue_Methods(t *testing.T) {
	e1 := event{time: t0, pinID: 1, seq: 1}
	e2 := event{time: t0.Add(time.Minute), pinID: 2, seq: 2}
	t.Run("Len should return number of events", func(t *testing.T) {
		q := eventQueue{e1, e2}
		assert.Equal(t, 2, q.Len())
	})
	t.Run("Less should compare by time", func(t *testing.T) {
		q := eventQueue{e1, e2}
		assert.True(t, q.Less(0, 1))
		assert.False(t, q.Less(1, 0))
	})
	t.Run("Less should compare by sequence when time is equal", func(t *testing.T) {
		q := eventQueue{{time: t0, seq: 1}, {time: t0, seq: 2}}
		assert.True(t, q.Less(0, 1))
		assert.False(t, q.Less(1, 0))
	})
	t.Run("Less should report false for equal events", func(t *testing.T) {
		q := eventQueue{e1, e1}
		assert.False(t, q.Less(0, 1))
	})
	t.Run("Swap should exchange events", func(t *testing.T) {
		q := eventQueue{e1, e2}
		q.Swap(0, 1)
		assert.Equal(t, eventQueue{e2, e1}, q)
	})
	t.Run("Push should append event", func(t *testing.T) {
		q := eventQueue{e2}
		q.Push(e1)
		assert.Equal(t, eventQueue{e2, e1}, q)
	})
	t.Run("Push should panic for other types", func(t *testing.T) {
		var q eventQueue
		assert.Panics(t, func() { q.Push(42) })
	})
	t.Run("Pop should remove and return last event", func(t *testing.T) {
		q := eventQueue{e1, e2}
		assert.Equal(t, e2, q.Pop())
		assert.Equal(t, eventQueue{e1}, q)
	})
	t.Run("Pop should panic when empty", func(t *testing.T) {
		var q eventQueue
		assert.Panics(t, func() { q.Pop() })
	})
}
