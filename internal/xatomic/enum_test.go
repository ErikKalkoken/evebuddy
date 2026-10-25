package xatomic_test

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/constraints"

	"github.com/ErikKalkoken/evebuddy/internal/xatomic"
)

type State int

const (
	Idle State = iota
	Running
	Stopped
)

func (s State) String() string {
	switch s {
	case Idle:
		return "Idle"
	case Running:
		return "Running"
	case Stopped:
		return "Stopped"
	}
	return "Unknown"
}

// plain has no String method.
type plain uint8

func TestZeroValue(t *testing.T) {
	// given
	var a xatomic.Enum[State]
	// when
	got := a.Load()
	// then
	assert.Equal(t, Idle, got)
}

func TestStoreAndLoad(t *testing.T) {
	// given
	var a xatomic.Enum[State]
	a.Store(Stopped)
	// when
	a.Store(Running)
	// then
	assert.Equal(t, Running, a.Load())
}

func TestSwap(t *testing.T) {
	// given
	var a xatomic.Enum[State]
	a.Store(Idle)
	// when
	old := a.Swap(Running)
	// then
	assert.Equal(t, Idle, old)
	assert.Equal(t, Running, a.Load())
}

func TestCompareAndSwap(t *testing.T) {
	cases := []struct {
		name        string
		initial     State
		old, new    State
		wantSwapped bool
		want        State
	}{
		{"matches", Idle, Idle, Running, true, Running},
		{"does not match", Idle, Running, Stopped, false, Idle},
		{"same value", Stopped, Stopped, Stopped, true, Stopped},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var a xatomic.Enum[State]
			a.Store(tc.initial)
			// when
			swapped := a.CompareAndSwap(tc.old, tc.new)
			// then
			assert.Equal(t, tc.wantSwapped, swapped)
			assert.Equal(t, tc.want, a.Load())
		})
	}
}

func TestString(t *testing.T) {
	t.Run("uses Stringer of T", func(t *testing.T) {
		// given
		var a xatomic.Enum[State]
		a.Store(Running)
		// when
		got := a.String()
		// then
		assert.Equal(t, "Running", got)
	})
	t.Run("falls back to number", func(t *testing.T) {
		// given
		var a xatomic.Enum[plain]
		a.Store(plain(42))
		// when
		got := a.String()
		// then
		assert.Equal(t, "42", got)
	})
}

// roundTrip checks that each value survives Store/Load, Swap and CompareAndSwap.
func roundTrip[T constraints.Integer](t *testing.T, values ...T) {
	t.Helper()
	for _, v := range values {
		var a xatomic.Enum[T]
		a.Store(v)
		assert.Equal(t, v, a.Load(), "Store/Load")
		assert.Equal(t, v, a.Swap(0), "Swap")
		a.Store(v)
		assert.True(t, a.CompareAndSwap(v, 0), "CompareAndSwap(%v, 0)", v)
		assert.Equal(t, T(0), a.Load())
	}
}

func TestAllIntegerKindsAndExtremeValues(t *testing.T) {
	t.Run("int", func(t *testing.T) { roundTrip(t, -1, 1, math.MinInt, math.MaxInt) })
	t.Run("int8", func(t *testing.T) { roundTrip[int8](t, -1, math.MinInt8, math.MaxInt8) })
	t.Run("int16", func(t *testing.T) { roundTrip[int16](t, -1, math.MinInt16, math.MaxInt16) })
	t.Run("int32", func(t *testing.T) { roundTrip[int32](t, -1, math.MinInt32, math.MaxInt32) })
	t.Run("int64", func(t *testing.T) { roundTrip[int64](t, -1, math.MinInt64, math.MaxInt64) })
	t.Run("uint", func(t *testing.T) { roundTrip[uint](t, 1, math.MaxUint) })
	t.Run("uint8", func(t *testing.T) { roundTrip[uint8](t, 1, math.MaxUint8) })
	t.Run("uint16", func(t *testing.T) { roundTrip[uint16](t, 1, math.MaxUint16) })
	t.Run("uint32", func(t *testing.T) { roundTrip[uint32](t, 1, math.MaxUint32) })
	t.Run("uint64", func(t *testing.T) { roundTrip[uint64](t, 1, math.MaxUint64) })
	t.Run("uintptr", func(t *testing.T) { roundTrip(t, 1, ^uintptr(0)) })
	t.Run("named type", func(t *testing.T) { roundTrip(t, Idle, Stopped, State(-1)) })
}

func TestCompareAndSwapFailsForWrongNegativeOld(t *testing.T) {
	// -1 is stored with all bits set; other edge values must not match it.
	cases := []struct {
		name string
		old  int8
	}{
		{"one", 1},
		{"min", math.MinInt8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var a xatomic.Enum[int8]
			a.Store(-1)
			// when
			swapped := a.CompareAndSwap(tc.old, 0)
			// then
			assert.False(t, swapped)
			assert.Equal(t, int8(-1), a.Load())
		})
	}
}

func TestConcurrentCompareAndSwapOnlyOneWins(t *testing.T) {
	// given
	const goroutines = 100
	var a xatomic.Enum[State]
	a.Store(Idle)
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		wins  int
		start = make(chan struct{})
	)
	for range goroutines {
		wg.Go(func() {
			<-start
			if a.CompareAndSwap(Idle, Running) {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		})
	}
	// when
	close(start)
	wg.Wait()
	// then
	assert.Equal(t, 1, wins)
	assert.Equal(t, Running, a.Load())
}

func TestConcurrentStoreAndLoadNeverSeesInvalidValue(t *testing.T) {
	// given
	var a xatomic.Enum[State]
	a.Store(Idle)
	valid := map[State]bool{Idle: true, Running: true, Stopped: true}
	// when
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Go(func() {
			for j := range 1000 {
				a.Store(State((i + j) % 3))
			}
		})
		wg.Go(func() {
			for range 1000 {
				v := a.Load()
				// then
				if !valid[v] {
					assert.Failf(t, "invalid value", "got %d", v)
					return
				}
			}
		})
	}
	wg.Wait()
}

func TestConcurrentSwapsPreserveAllValues(t *testing.T) {
	// Each goroutine swaps in a unique value. Every value written must be
	// observed exactly once, either as a returned old value or as the final value.
	// given
	const n = 1000
	type id int32
	var a xatomic.Enum[id]
	a.Store(id(-1))
	// when
	results := make(chan id, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			results <- a.Swap(id(i))
		})
	}
	wg.Wait()
	close(results)
	// then
	seen := make(map[id]int, n+1)
	for v := range results {
		seen[v]++
	}
	seen[a.Load()]++

	require.Len(t, seen, n+1)
	for v, count := range seen {
		assert.Equalf(t, 1, count, "value %d seen %d times", v, count)
	}
}
