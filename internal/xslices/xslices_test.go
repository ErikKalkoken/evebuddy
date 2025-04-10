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
		z := xslices.Reduce(s, func(x, y int) int {
			return x + y
		})
		assert.Equal(t, 10, z)
	})
	t.Run("should return result when there is one item", func(t *testing.T) {
		s := []int{1}
		z := xslices.Reduce(s, func(x, y int) int {
			return x + y
		})
		assert.Equal(t, 1, z)
	})
	t.Run("should panic if there are no items", func(t *testing.T) {
		s := []int{}
		assert.Panics(t, func() {
			xslices.Reduce(s, func(x, y int) int {
				return x + y
			})
		})
	})
}
