package xsync_test

import (
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

func TestThrottle(t *testing.T) {
	t.Run("should run at once on first trigger", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			start := time.Now()
			var runs []time.Duration
			th := xsync.NewThrottle(time.Second, func() {
				runs = append(runs, time.Since(start))
			})

			// when
			th.Trigger()

			// then
			time.Sleep(time.Millisecond)
			th.Stop()
			xassert.Equal(t, []time.Duration{0}, runs)
		})
	})
	t.Run("should merge a burst of triggers into one run", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			var count atomic.Int64
			th := xsync.NewThrottle(time.Second, func() {
				count.Add(1)
			})
			defer th.Stop()
			th.Trigger()
			time.Sleep(100 * time.Millisecond)

			// when
			for range 10 {
				th.Trigger()
			}

			// then
			time.Sleep(5 * time.Second)
			xassert.Equal(t, 2, count.Load())
		})
	})
	t.Run("should wait for the interval after the last run", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			start := time.Now()
			var runs []time.Duration
			th := xsync.NewThrottle(time.Second, func() {
				runs = append(runs, time.Since(start))
			})
			th.Trigger()
			time.Sleep(100 * time.Millisecond)

			// when
			th.Trigger()

			// then
			time.Sleep(5 * time.Second)
			th.Stop()
			xassert.Equal(t, []time.Duration{0, time.Second}, runs)
		})
	})
	t.Run("should run once more when triggered during a run", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			var count atomic.Int64
			release := make(chan struct{})
			th := xsync.NewThrottle(time.Second, func() {
				if count.Add(1) == 1 {
					<-release
				}
			})
			defer th.Stop()
			th.Trigger()
			synctest.Wait()

			// when
			for range 3 {
				th.Trigger()
			}
			close(release)

			// then
			time.Sleep(5 * time.Second)
			xassert.Equal(t, 2, count.Load())
		})
	})
	t.Run("should not overlap runs", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			var running, overlaps, count atomic.Int64
			th := xsync.NewThrottle(10*time.Millisecond, func() {
				if running.Add(1) > 1 {
					overlaps.Add(1)
				}
				time.Sleep(50 * time.Millisecond)
				running.Add(-1)
				count.Add(1)
			})
			defer th.Stop()

			// when
			for range 20 {
				th.Trigger()
				time.Sleep(5 * time.Millisecond)
			}

			// then
			time.Sleep(time.Second)
			xassert.Equal(t, 0, overlaps.Load())
			xassert.Equal(t, 3, count.Load())
		})
	})
	t.Run("should not run a pending run after stop", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			var count atomic.Int64
			th := xsync.NewThrottle(time.Second, func() {
				count.Add(1)
			})
			th.Trigger()
			time.Sleep(100 * time.Millisecond)
			th.Trigger()

			// when
			th.Stop()

			// then
			time.Sleep(5 * time.Second)
			xassert.Equal(t, 1, count.Load())
		})
	})
	t.Run("should ignore triggers after stop", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			var count atomic.Int64
			th := xsync.NewThrottle(time.Second, func() {
				count.Add(1)
			})
			th.Stop()

			// when
			th.Trigger()

			// then
			time.Sleep(5 * time.Second)
			xassert.Equal(t, 0, count.Load())
		})
	})
	t.Run("should allow stop to be called twice", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			th := xsync.NewThrottle(time.Second, func() {})
			th.Stop()

			// when
			th.Stop()

			// then: returned instead of blocking (synctest fails on deadlock)
		})
	})
	t.Run("should wait for a running run when stopped", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			var finished atomic.Bool
			th := xsync.NewThrottle(time.Second, func() {
				time.Sleep(time.Second)
				finished.Store(true)
			})
			th.Trigger()
			synctest.Wait()

			// when
			th.Stop()

			// then
			xassert.Equal(t, true, finished.Load())
		})
	})
}
