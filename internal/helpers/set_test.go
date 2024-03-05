package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetHas(t *testing.T) {
	s := NewSet([]int{3, 7, 9})

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
	got := NewSet([]int{})
	got.Add(3)
	want := NewSet([]int{3})
	assert.Equal(t, want, got)
}

func TestSetCanAddToFull(t *testing.T) {
	got := NewSet([]int{1, 2})
	got.Add(3)
	want := NewSet([]int{1, 2, 3})
	assert.Equal(t, want, got)
}

func TestSetSize(t *testing.T) {
	s1 := NewSet([]int{1, 2, 3})
	assert.Equal(t, 3, s1.Size())

	s2 := NewSet([]int{})
	assert.Equal(t, 0, s2.Size())
}

func TestSetCanRemoveWhenExists(t *testing.T) {
	got := NewSet([]int{1, 2})
	got.Remove(2)
	want := NewSet([]int{1})
	assert.Equal(t, want, got)
}

func TestSetCanRemoveWhenNotExists(t *testing.T) {
	got := NewSet([]int{1, 2})
	got.Remove(3)
	want := NewSet([]int{1, 2})
	assert.Equal(t, want, got)
}

func TestSetCanClear(t *testing.T) {
	got := NewSet([]int{1, 2})
	got.Clear()
	want := NewSet([]int{})
	assert.Equal(t, want, got)
}

func TestSetCanConvertToSlice(t *testing.T) {
	s := NewSet([]int{1, 2})
	r := s.ToSlice()
	assert.Equal(t, []int{1, 2}, r)
}
