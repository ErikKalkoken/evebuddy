package slices_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/helper/slices"

	"github.com/stretchr/testify/assert"
)

func TestConvertNumeric(t *testing.T) {
	t.Run("can convert int32 to int", func(t *testing.T) {
		want := []int{1, 2, 3}
		got := slices.ConvertNumeric[int32, int]([]int32{1, 2, 3})
		assert.Equal(t, want, got)
	})
	t.Run("can convert int32 to int64", func(t *testing.T) {
		want := []int64{1, 2, 3}
		got := slices.ConvertNumeric[int32, int64]([]int32{1, 2, 3})
		assert.Equal(t, want, got)
	})
	t.Run("can convert int to int32", func(t *testing.T) {
		want := []int32{1, 2, 3}
		got := slices.ConvertNumeric[int, int32]([]int{1, 2, 3})
		assert.Equal(t, want, got)
	})
	t.Run("can convert empty slice", func(t *testing.T) {
		want := []int32{}
		got := slices.ConvertNumeric[int, int32]([]int{})
		assert.Equal(t, want, got)
	})
}