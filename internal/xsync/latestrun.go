package xsync

import "sync/atomic"

// LatestRun identifies the newest of overlapping async runs,
// so results from older runs can be discarded.
type LatestRun struct{ n atomic.Int64 }

// Start begins a new run and returns a function reporting whether it is still the newest.
func (l *LatestRun) Start() func() bool {
	n := l.n.Add(1)
	return func() bool { return l.n.Load() == n }
}

// Discard marks all started runs as outdated.
func (l *LatestRun) Discard() { l.n.Add(1) }
