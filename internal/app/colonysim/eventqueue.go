package colonysim

import "time"

type event struct {
	time  time.Time
	pinID int64
	seq   uint64 // also breaks ties in insertion order
}

// eventQueue is a priority queue of events ordered by time.
type eventQueue []event

func (q eventQueue) Len() int { return len(q) }
func (q eventQueue) Less(i, j int) bool {
	if c := q[i].time.Compare(q[j].time); c != 0 {
		return c < 0
	}
	return q[i].seq < q[j].seq
}
func (q eventQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *eventQueue) Push(x any)   { *q = append(*q, x.(event)) }
func (q *eventQueue) Pop() any {
	old := *q
	n := len(old)
	x := old[n-1]
	*q = old[:n-1]
	return x
}
