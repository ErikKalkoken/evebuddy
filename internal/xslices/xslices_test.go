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
