package xiter_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/stretchr/testify/assert"
)

func TestCount(t *testing.T) {
	t.Run("should return with index starting at 0", func(t *testing.T) {
		s := []string{"a", "b", "c"}
		got := maps.Collect(xiter.Count(slices.Values(s), 0))
		expected := map[int]string{0: "a", 1: "b", 2: "c"}
		assert.Equal(t, expected, got)
	})
	t.Run("should return with index starting at 3", func(t *testing.T) {
		s := []string{"a", "b", "c"}
		got := maps.Collect(xiter.Count(slices.Values(s), 3))
		expected := map[int]string{3: "a", 4: "b", 5: "c"}
		assert.Equal(t, expected, got)
	})
}

func TestFilter(t *testing.T) {
	s := []int{1, 2, 3}
	got := slices.Collect(xiter.Filter(slices.Values(s), func(x int) bool {
		return x%2 == 0
	}))
	expected := []int{2}
	assert.Equal(t, expected, got)
}

func TestFilterSlice(t *testing.T) {
	s := []int{1, 2, 3}
	got := slices.Collect(xiter.FilterSlice(s, func(x int) bool {
		return x%2 == 0
	}))
	expected := []int{2}
	assert.Equal(t, expected, got)
}

func TestMap(t *testing.T) {
	s := []int{1, 2, 3}
	got := slices.Collect(xiter.Map(slices.Values(s), func(x int) string {
		return fmt.Sprint(x)
	}))
	expected := []string{"1", "2", "3"}
	assert.Equal(t, expected, got)
}

func TestMapSlice(t *testing.T) {
	s := []int{1, 2, 3}
	got := slices.Collect(xiter.MapSlice(s, func(x int) string {
		return fmt.Sprint(x)
	}))
	expected := []string{"1", "2", "3"}
	assert.Equal(t, expected, got)
}
