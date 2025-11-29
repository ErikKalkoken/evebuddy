package xslices_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func TestDeduplicate(t *testing.T) {
	t.Run("can remove duplicate elements", func(t *testing.T) {
		s := []string{"b", "a", "b"}
		got := xslices.Deduplicate(s)
		want := []string{"b", "a"}
		assert.Equal(t, want, got)
	})
	t.Run("can process slices with no duplicates", func(t *testing.T) {
		s := []string{"b", "a"}
		got := xslices.Deduplicate(s)
		want := []string{"b", "a"}
		assert.Equal(t, want, got)
	})
	t.Run("can processs empty slice", func(t *testing.T) {
		s := []string{}
		got := xslices.Deduplicate(s)
		want := []string{}
		assert.Equal(t, want, got)
	})
}

func TestFilter(t *testing.T) {
	t.Run("should filter a slice", func(t *testing.T) {
		s1 := []int{1, 2, 3, 4}
		s2 := xslices.Filter(s1, func(x int) bool {
			return x%2 == 0
		})
		assert.Equal(t, []int{2, 4}, s2)
	})
}

func TestMap(t *testing.T) {
	t.Run("should return mapped slice", func(t *testing.T) {
		s1 := []int{1, 2, 3}
		s2 := xslices.Map(s1, func(x int) string {
			return fmt.Sprint(x)
		})
		assert.Equal(t, []string{"1", "2", "3"}, s2)
	})
}

func TestPop(t *testing.T) {
	t.Run("NormalIntegerPop", func(t *testing.T) {
		s := []int{10, 20, 30}
		sPtr := &s
		popped, ok := xslices.Pop(sPtr)
		assert.True(t, ok)
		assert.Equal(t, 30, popped)
		assert.Equal(t, []int{10, 20}, s)
	})

	t.Run("PopStringFromSingleElementSlice", func(t *testing.T) {
		s := []string{"last"}
		sPtr := &s
		popped, ok := xslices.Pop(sPtr)
		assert.True(t, ok)
		assert.Equal(t, "last", popped)
		assert.Empty(t, s)
	})

	t.Run("PopFromEmptySlice", func(t *testing.T) {
		s := []float64{}
		sPtr := &s
		popped, ok := xslices.Pop(sPtr)
		assert.False(t, ok)
		var zero float64
		assert.Equal(t, zero, popped)
		assert.Empty(t, s)
	})

	t.Run("PopStructs", func(t *testing.T) {
		type Item struct {
			ID   int
			Name string
		}
		s := []Item{{1, "A"}, {2, "B"}}
		sPtr := &s
		expectedPopped := Item{2, "B"}
		expectedRemaining := []Item{{1, "A"}}
		popped, ok := xslices.Pop(sPtr)
		assert.True(t, ok)
		assert.Equal(t, expectedPopped, popped)
		assert.Equal(t, expectedRemaining, s)
	})

	t.Run("PopWithNilPointer", func(t *testing.T) {
		var sPtr *[]int // A nil pointer to a slice of ints
		popped, ok := xslices.Pop(sPtr)
		assert.False(t, ok)
		var zero int // Expected zero value for int
		assert.Equal(t, zero, popped)
		// The original pointer remains nil, and the function does not panic.
	})
}

func TestReduce(t *testing.T) {
	t.Run("should return result when there are multple items", func(t *testing.T) {
		s := []int{1, 2, 3, 4}
		got := xslices.Reduce(s, func(x, y int) int {
			return x + y
		})
		assert.Equal(t, 10, got)
	})
	t.Run("should return result when there is one item", func(t *testing.T) {
		s := []int{1}
		got := xslices.Reduce(s, func(x, y int) int {
			return x + y
		})
		assert.Equal(t, 1, got)
	})
	t.Run("should return zero value when there are no items", func(t *testing.T) {
		s := []int{}
		got := xslices.Reduce(s, func(x, y int) int {
			return x + y
		})
		assert.Equal(t, 0, got)
	})
}
