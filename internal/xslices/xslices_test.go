package xslices_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func TestSlices(t *testing.T) {
	t.Run("should return mapped slice", func(t *testing.T) {
		s1 := []int{1, 2, 3}
		s2 := xslices.Map(s1, func(x int) string {
			return fmt.Sprint(x)
		})
		assert.Equal(t, []string{"1", "2", "3"}, s2)
	})
	t.Run("should filter a slice", func(t *testing.T) {
		s1 := []int{1, 2, 3, 4}
		s2 := xslices.Filter(s1, func(x int) bool {
			return x%2 == 0
		})
		assert.Equal(t, []int{2, 4}, s2)
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

func TestDeduplicate(t *testing.T) {
	t.Run("can remove duplucate elements", func(t *testing.T) {
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
