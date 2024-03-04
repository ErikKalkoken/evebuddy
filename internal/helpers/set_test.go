package helpers

import (
	"testing"
)

func TestSet(t *testing.T) {
	items := []int{3, 7, 9}
	s := NewSetFromSlice(items)

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
		got := s.Has(c.in)
		if got != c.want {
			t.Errorf("Has(%q) == %v, want %v", c.in, got, c.want)
		}
	}
}
