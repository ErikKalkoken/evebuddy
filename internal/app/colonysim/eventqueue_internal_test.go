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
