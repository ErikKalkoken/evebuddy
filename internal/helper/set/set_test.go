package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetHas(t *testing.T) {
	s := NewFromSlice([]int{3, 7, 9})

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
		result := s.Has(c.in)
		assert.Equalf(t, c.want, result, "case: %v", c.in)
	}
}

func TestSetCanAddToEmpty(t *testing.T) {
	got := New[int]()
	got.Add(3)
	want := NewFromSlice([]int{3})
	assert.Equal(t, want, got)
}

func TestSetCanAddToFull(t *testing.T) {
	got := NewFromSlice([]int{1, 2})
	got.Add(3)
	want := NewFromSlice([]int{1, 2, 3})
	assert.Equal(t, want, got)
}

func TestSetSize(t *testing.T) {
	s1 := NewFromSlice([]int{1, 2, 3})
	assert.Equal(t, 3, s1.Size())

	s2 := NewFromSlice([]int{})
	assert.Equal(t, 0, s2.Size())
}

func TestSetCanRemoveWhenExists(t *testing.T) {
	got := NewFromSlice([]int{1, 2})
	got.Remove(2)
	want := NewFromSlice([]int{1})
	assert.Equal(t, want, got)
}

func TestSetCanRemoveWhenNotExists(t *testing.T) {
	got := NewFromSlice([]int{1, 2})
	got.Remove(3)
	want := NewFromSlice([]int{1, 2})
	assert.Equal(t, want, got)
}

func TestSetCanClear(t *testing.T) {
	got := NewFromSlice([]int{1, 2})
	got.Clear()
	want := NewFromSlice([]int{})
	assert.Equal(t, want, got)
}

func TestSetCanConvertToSlice(t *testing.T) {
	s := NewFromSlice([]int{1, 2})
	got := s.ToSlice()
	assert.Len(t, got, 2)
	assert.Contains(t, got, 1)
	assert.Contains(t, got, 2)
}

func TestSetCanUnion(t *testing.T) {
	s1 := NewFromSlice([]int{1, 2})
	s2 := NewFromSlice([]int{2, 3})
	want := NewFromSlice([]int{1, 2, 3})
	got := s1.Union(s2)
	assert.Equal(t, want, got)
}

func TestSetCanIntersect(t *testing.T) {
	s1 := NewFromSlice([]int{1, 2})
	s2 := NewFromSlice([]int{2, 3})
	want := NewFromSlice([]int{2})
	got := s1.Intersect(s2)
	assert.Equal(t, want, got)
}

func TestSetCanDifference(t *testing.T) {
	s1 := NewFromSlice([]int{1, 2})
	s2 := NewFromSlice([]int{2, 3})
	want := NewFromSlice([]int{1})
	got := s1.Difference(s2)
	assert.Equal(t, want, got)
}

func TestSetEqual(t *testing.T) {
	t.Run("report true when sets are equal", func(t *testing.T) {
		s1 := NewFromSlice([]int{1, 2})
		s2 := NewFromSlice([]int{1, 2})
		assert.True(t, s1.Equal(s2))
	})
	t.Run("report true when sets are equal and empty", func(t *testing.T) {
		s1 := NewFromSlice([]int{})
		s2 := NewFromSlice([]int{})
		assert.True(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 1", func(t *testing.T) {
		s1 := NewFromSlice([]int{1, 2})
		s2 := NewFromSlice([]int{1, 2, 3})
		assert.False(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 2", func(t *testing.T) {
		s1 := NewFromSlice([]int{1, 2, 3})
		s2 := NewFromSlice([]int{1, 2})
		assert.False(t, s1.Equal(s2))
	})
	t.Run("report false when sets are not equal 3", func(t *testing.T) {
		s1 := NewFromSlice([]int{1, 2})
		s2 := NewFromSlice([]int{2, 3})
		assert.False(t, s1.Equal(s2))
	})
}
