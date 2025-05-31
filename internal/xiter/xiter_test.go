package xiter_test

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/stretchr/testify/assert"
)

func TestChain(t *testing.T) {
	t.Run("can return the elements of all sequences", func(t *testing.T) {
		s1 := []int{1, 2}
		s2 := []int{3, 4}
		got := slices.Collect(xiter.Chain(slices.Values(s1), slices.Values(s2)))
		want := []int{1, 2, 3, 4}
		assert.Equal(t, want, got)
	})
	t.Run("can stop iterating", func(t *testing.T) {
		s1 := []int{1, 2}
		s2 := []int{3, 4}
		it := xiter.Chain(slices.Values(s1), slices.Values(s2))
		next, stop := iter.Pull(it)
		defer stop()
		got := make([]int, 0)
		for range 3 {
			v, ok := next()
			if ok {
				got = append(got, v)
			}
		}
		want := []int{1, 2, 3}
		assert.Equal(t, want, got)
	})
}

func TestCount(t *testing.T) {
	t.Run("should return with index starting at 0", func(t *testing.T) {
		s := []string{"a", "b", "c"}
		got := maps.Collect(xiter.Count(slices.Values(s), 0))
		want := map[int]string{0: "a", 1: "b", 2: "c"}
		assert.Equal(t, want, got)
	})
	t.Run("should return with index starting at 3", func(t *testing.T) {
		s := []string{"a", "b", "c"}
		got := maps.Collect(xiter.Count(slices.Values(s), 3))
		want := map[int]string{3: "a", 4: "b", 5: "c"}
		assert.Equal(t, want, got)
	})
	t.Run("can stop counting", func(t *testing.T) {
		s := []string{"a", "b", "c"}
		seq := xiter.Count(slices.Values(s), 0)
		next, stop := iter.Pull2(seq)
		defer stop()
		got := make(map[int]string)
		for range 2 {
			k, v, ok := next()
			if ok {
				got[k] = v
			}
		}
		want := map[int]string{0: "a", 1: "b"}
		assert.Equal(t, want, got)
	})

}

func TestFilter(t *testing.T) {
	t.Run("can filter a sequence", func(t *testing.T) {
		s := []int{1, 2, 3}
		got := slices.Collect(xiter.Filter(slices.Values(s), func(x int) bool {
			return x%2 != 0
		}))
		want := []int{1, 3}
		assert.Equal(t, want, got)
	})
	t.Run("can stop filtering", func(t *testing.T) {
		s := []int{1, 2, 3}
		seq := xiter.Filter(slices.Values(s), func(x int) bool {
			return x%2 != 0
		})
		next, stop := iter.Pull(seq)
		defer stop()
		got := make([]int, 0)
		for range 1 {
			v, ok := next()
			if ok {
				got = append(got, v)
			}
		}
		want := []int{1}
		assert.Equal(t, want, got)
	})
}

func TestFilterSlice(t *testing.T) {
	s := []int{1, 2, 3}
	got := slices.Collect(xiter.FilterSlice(s, func(x int) bool {
		return x%2 == 0
	}))
	want := []int{2}
	assert.Equal(t, want, got)
}

func TestMap(t *testing.T) {
	s := []int{1, 2, 3}
	it := xiter.Map(slices.Values(s), func(x int) string {
		return fmt.Sprint(x)
	})
	next, stop := iter.Pull(it)
	defer stop()
	got := make([]string, 0)
	for range 2 {
		v, ok := next()
		if ok {
			got = append(got, v)
		}
	}
	want := []string{"1", "2"}
	assert.Equal(t, want, got)
}

func TestMapSlice(t *testing.T) {
	s := []int{1, 2, 3}
	got := slices.Collect(xiter.MapSlice(s, func(x int) string {
		return fmt.Sprint(x)
	}))
	want := []string{"1", "2", "3"}
	assert.Equal(t, want, got)
}

func TestMapSlice2(t *testing.T) {
	t.Run("can map a slice", func(t *testing.T) {
		s := []int{1, 2, 3}
		it := xiter.MapSlice2(s, func(v int) (int, int) {
			return v, v * 2
		})
		got := maps.Collect(it)
		want := map[int]int{1: 2, 2: 4, 3: 6}
		assert.Equal(t, want, got)
	})
	t.Run("can stop mapping", func(t *testing.T) {
		s := []int{1, 2, 3}
		it := xiter.MapSlice2(s, func(v int) (int, int) {
			return v, v * 2
		})
		next, stop := iter.Pull2(it)
		defer stop()
		got := make(map[int]int)
		for range 2 {
			k, v, ok := next()
			if ok {
				got[k] = v
			}
		}
		want := map[int]int{1: 2, 2: 4}
		assert.Equal(t, want, got)
	})
}
