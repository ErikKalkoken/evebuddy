package set_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	t.Parallel()
	t.Run("can convert to string", func(t *testing.T) {
		s := set.New(42)
		assert.Equal(t, "[42]", fmt.Sprint(s))
	})
	t.Run("can clear", func(t *testing.T) {
		got := set.New(1, 2)
		got.Clear()
		want := set.New[int]()
		assert.Equal(t, want, got)
	})
	t.Run("can convert to slice", func(t *testing.T) {
		s := set.New(1, 2)
		got := s.ToSlice()
		assert.Len(t, got, 2)
		assert.Contains(t, got, 1)
		assert.Contains(t, got, 2)
	})
	t.Run("can union", func(t *testing.T) {
		s1 := set.New(1, 2)
		s2 := set.New(2, 3)
		want := set.New(1, 2, 3)
		got := s1.Union(s2)
		assert.Equal(t, want, got)
	})
	t.Run("can intersect", func(t *testing.T) {
		s1 := set.New(1, 2)
		s2 := set.New(2, 3)
		want := set.New(2)
		got := s1.Intersect(s2)
		assert.Equal(t, want, got)
	})
	t.Run("can calculate difference", func(t *testing.T) {
		s1 := set.New(1, 2)
		s2 := set.New(2, 3)
		want := set.New(1)
		got := s1.Difference(s2)
		assert.Equal(t, want, got)
	})
	t.Run("can iterate over set", func(t *testing.T) {
		s1 := set.New(1, 2, 3)
		s2 := set.New[int]()
		for e := range s1.Values() {
			s2.Add(e)
		}
		assert.Equal(t, s1, s2)
	})
	t.Run("can assert equality", func(t *testing.T) {
		a1 := set.New(1, 2, 3)
		a2 := set.New(1, 2, 3)
		b := set.New(2, 3, 4)
		c := set.New[int]()
		assert.Equal(t, a1, a2)
		assert.NotEqual(t, b, a2)
		assert.NotEqual(t, b, c)
	})
}

func TestSetAdd(t *testing.T) {
	t.Parallel()
	t.Run("can add to empty", func(t *testing.T) {
		got := set.New[int]()
		got.Add(3)
		want := set.New(3)
		assert.Equal(t, want, got)
	})
	t.Run("can add to full", func(t *testing.T) {
		got := set.New(1, 2)
		got.Add(3)
		want := set.New(1, 2, 3)
		assert.Equal(t, want, got)
	})
}

func TestSetClone(t *testing.T) {
	t.Parallel()
	t.Run("can clone a set", func(t *testing.T) {
		a := set.New(1, 2)
		b := a.Clone()
		assert.True(t, b.Equal(a))
		a.Add(3)
		b.Add(4)
		assert.False(t, b.Equal(a))
	})
}

func TestNewFromSlice(t *testing.T) {
	t.Parallel()
	t.Run("can create from slice", func(t *testing.T) {
		got := set.NewFromSlice([]int{1, 2})
		want := set.New(1, 2)
		assert.Equal(t, want, got)
	})
	t.Run("can create from empty slice", func(t *testing.T) {
		got := set.NewFromSlice([]int{})
		want := set.New[int]()
		assert.Equal(t, want, got)
	})
}

func TestSetRemove(t *testing.T) {
	t.Parallel()
	t.Run("can remove item when it exists", func(t *testing.T) {
		got := set.New(1, 2)
		got.Remove(2)
		want := set.New(1)
		assert.Equal(t, want, got)
	})
	t.Run("can remove item when it doesn't exist", func(t *testing.T) {
		got := set.New(1, 2)
		got.Remove(3)
		want := set.New(1, 2)
		assert.Equal(t, want, got)
	})
}

func TestSetHas(t *testing.T) {
	t.Parallel()
	s := set.New(3, 7, 9)
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
	t.Parallel()
	t.Run("can determine size of filled set", func(t *testing.T) {
		s1 := set.New(1, 2, 3)
		assert.Equal(t, 3, s1.Size())
	})
	t.Run("can determine size of empty set", func(t *testing.T) {
		s2 := set.New[int]()
		assert.Equal(t, 0, s2.Size())
	})
}

func TestSetEqual(t *testing.T) {
	t.Parallel()
	t.Run("report true when sets are equal", func(t *testing.T) {
		s1 := set.New(1, 2)
		s2 := set.New(1, 2)
		assert.True(t, s1.Equal(s2))
	})
	t.Run("report true when sets are equal and empty", func(t *testing.T) {
		s1 := set.New[int]()
		s2 := set.New[int]()
		assert.True(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 1", func(t *testing.T) {
		s1 := set.New(1, 2)
		s2 := set.New(1, 2, 3)
		assert.False(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 2", func(t *testing.T) {
		s1 := set.New(1, 2, 3)
		s2 := set.New(1, 2)
		assert.False(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 3", func(t *testing.T) {
		s1 := set.New(1, 2)
		s2 := set.New(2, 3)
		assert.False(t, s1.Equal(s2))
	})
}

func TestIsSubset(t *testing.T) {
	t.Parallel()
	t.Run("report true when a is subset of b", func(t *testing.T) {
		a := set.New(1, 2)
		b := set.New(1, 2, 3)
		assert.True(t, a.IsSubset(b))
	})
	t.Run("report true when a is same as b", func(t *testing.T) {
		a := set.New(1, 2)
		b := set.New(1, 2)
		assert.True(t, a.IsSubset(b))
	})
	t.Run("report false when a is not a subset of b", func(t *testing.T) {
		a := set.New(1, 3)
		b := set.New(1, 2, 4)
		assert.False(t, a.IsSubset(b))
	})
}

func TestPop(t *testing.T) {
	t.Parallel()
	t.Run("can remove element", func(t *testing.T) {
		s := set.New(1, 2, 3)
		x, err := s.Pop()
		if assert.NoError(t, err) {
			assert.True(t, set.New(1, 2, 3).Contains(x))
		}
	})
	t.Run("should return error when set is empty", func(t *testing.T) {
		s := set.New[int]()
		_, err := s.Pop()
		assert.ErrorIs(t, err, set.ErrNotFound)
	})
}
