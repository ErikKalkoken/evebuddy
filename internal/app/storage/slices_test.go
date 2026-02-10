package storage

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestConvertNumericSlice(t *testing.T) {
	t.Run("can convert int64 to int", func(t *testing.T) {
		want := []int{1, 2, 3}
		got := convertNumericSlice[int]([]int64{1, 2, 3})
		xassert.Equal(t, want, got)
	})
	t.Run("can convert int64 to int64", func(t *testing.T) {
		want := []int64{1, 2, 3}
		got := convertNumericSlice[int64]([]int64{1, 2, 3})
		xassert.Equal(t, want, got)
	})
	t.Run("can convert int to int64", func(t *testing.T) {
		want := []int64{1, 2, 3}
		got := convertNumericSlice[int64]([]int{1, 2, 3})
		xassert.Equal(t, want, got)
	})
	t.Run("can convert empty slice", func(t *testing.T) {
		want := []int64{}
		got := convertNumericSlice[int64]([]int{})
		xassert.Equal(t, want, got)
	})
}
