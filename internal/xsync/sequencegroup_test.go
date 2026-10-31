package xsync_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

func TestSequenceGroup(t *testing.T) {
	t.Run("should run jobs in the order they were added", func(t *testing.T) {
		// given
		g := new(xsync.SequenceGroup)
		var got []int
		// when
		g.Go(func() {
			time.Sleep(50 * time.Millisecond)
			got = append(got, 1)
		})
		g.Go(func() {
			time.Sleep(5 * time.Millisecond)
			got = append(got, 2)
		})
		g.Go(func() {
			time.Sleep(10 * time.Millisecond)
			got = append(got, 3)
		})
		g.Wait()
		// then
		xassert.Equal(t, []int{1, 2, 3}, got)
	})
	t.Run("should never run jobs at the same time", func(t *testing.T) {
		// given
		g := new(xsync.SequenceGroup)
		var running, overlaps atomic.Int64
		// when
		for range 20 {
			g.Go(func() {
				if running.Add(1) > 1 {
					overlaps.Add(1)
				}
				time.Sleep(time.Millisecond)
				running.Add(-1)
			})
		}
		g.Wait()
		// then
		xassert.Equal(t, int64(0), overlaps.Load())
	})
	t.Run("should return immediately from Wait when no jobs were added", func(t *testing.T) {
		// given
		g := new(xsync.SequenceGroup)
		// when
		g.Wait()
		// then: did not block
	})
	t.Run("can be reused after Wait", func(t *testing.T) {
		// given
		g := new(xsync.SequenceGroup)
		var c atomic.Int64
		g.Go(func() { c.Add(1) })
		g.Wait()
		// when
		g.Go(func() { c.Add(1) })
		g.Wait()
		// then
		xassert.Equal(t, int64(2), c.Load())
	})
	t.Run("can call Wait concurrently with Go", func(t *testing.T) {
		// given
		g := new(xsync.SequenceGroup)
		var c atomic.Int64
		var wg sync.WaitGroup
		// when
		for range 10 {
			wg.Go(func() {
				for range 100 {
					g.Go(func() { c.Add(1) })
					g.Wait()
				}
			})
		}
		wg.Wait()
		g.Wait()
		// then
		xassert.Equal(t, int64(1000), c.Load())
	})
}
