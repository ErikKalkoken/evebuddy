package set_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestSetAdd(t *testing.T) {
	t.Run("can add to empty", func(t *testing.T) {
		got := set.New[int]()
		got.Add(3)
		want := set.NewFromSlice([]int{3})
		assert.Equal(t, want, got)
	})
	t.Run("can add to full", func(t *testing.T) {
		got := set.NewFromSlice([]int{1, 2})
		got.Add(3)
		want := set.NewFromSlice([]int{1, 2, 3})
		assert.Equal(t, want, got)

	})
}

func TestSetRemove(t *testing.T) {
	t.Run("can remove item when it exists", func(t *testing.T) {
		got := set.NewFromSlice([]int{1, 2})
		got.Remove(2)
		want := set.NewFromSlice([]int{1})
		assert.Equal(t, want, got)
	})
	t.Run("can remove item when it doesn't exist", func(t *testing.T) {
		got := set.NewFromSlice([]int{1, 2})
		got.Remove(3)
		want := set.NewFromSlice([]int{1, 2})
		assert.Equal(t, want, got)
	})
}

func TestSetHas(t *testing.T) {
	s := set.NewFromSlice([]int{3, 7, 9})
	cases := []struct {
		in   int
		want bool
	}{
		{3, true},
		{7, true},
		{9, true},
		{1, false},
		{0, false},
		{-1, false},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("case: %v", c.in), func(t *testing.T) {
			result := s.Contains(c.in)
			assert.Equal(t, c.want, result)
		})
	}
}

func TestSetSize(t *testing.T) {
	t.Run("can determine size of filled set", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2, 3})
		assert.Equal(t, 3, s1.Size())
	})
	t.Run("can determine size of empty set", func(t *testing.T) {
		s2 := set.NewFromSlice([]int{})
		assert.Equal(t, 0, s2.Size())
	})
}

func TestSetOther(t *testing.T) {
	t.Run("can convert to string", func(t *testing.T) {
		x := []int{42}
		s := set.NewFromSlice(x)
		assert.Equal(t, fmt.Sprint(x), fmt.Sprint(s))
	})
	t.Run("can clear", func(t *testing.T) {
		x := []int{1, 2}
		got := set.NewFromSlice(x)
		got.Clear()
		want := set.NewFromSlice([]int{})
		assert.Equal(t, want, got)
	})
	t.Run("can convert to slice", func(t *testing.T) {
		s := set.NewFromSlice([]int{1, 2})
		got := s.ToSlice()
		assert.Len(t, got, 2)
		assert.Contains(t, got, 1)
		assert.Contains(t, got, 2)
	})
	t.Run("can union", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2})
		s2 := set.NewFromSlice([]int{2, 3})
		want := set.NewFromSlice([]int{1, 2, 3})
		got := s1.Union(s2)
		assert.Equal(t, want, got)
	})
	t.Run("can intersect", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2})
		s2 := set.NewFromSlice([]int{2, 3})
		want := set.NewFromSlice([]int{2})
		got := s1.Intersect(s2)
		assert.Equal(t, want, got)
	})
	t.Run("can calculate difference", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2})
		s2 := set.NewFromSlice([]int{2, 3})
		want := set.NewFromSlice([]int{1})
		got := s1.Difference(s2)
		assert.Equal(t, want, got)
	})
	t.Run("can iterate over set", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2, 3})
		s2 := set.New[int]()
		for e := range s1.All() {
			s2.Add(e)
		}
		assert.Equal(t, s1, s2)
	})

}

func TestSetEqual(t *testing.T) {
	t.Run("report true when sets are equal", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2})
		s2 := set.NewFromSlice([]int{1, 2})
		assert.True(t, s1.Equal(s2))
	})
	t.Run("report true when sets are equal and empty", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{})
		s2 := set.NewFromSlice([]int{})
		assert.True(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 1", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2})
		s2 := set.NewFromSlice([]int{1, 2, 3})
		assert.False(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 2", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2, 3})
		s2 := set.NewFromSlice([]int{1, 2})
		assert.False(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 3", func(t *testing.T) {
		s1 := set.NewFromSlice([]int{1, 2})
		s2 := set.NewFromSlice([]int{2, 3})
		assert.False(t, s1.Equal(s2))
	})
}
