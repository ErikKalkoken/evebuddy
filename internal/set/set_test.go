package set_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		s    set.Set[int]
		v    int
		want set.Set[int]
	}{
		{"add to empty", empty, 1, set.New(1)},
		{"add to zero", zero, 1, set.New(1)},
		{"add new to non-empty", set.New(1), 2, set.New(1, 2)},
		{"add existing to non-empty", set.New(1), 1, set.New(1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.s.Add(tc.v)
			assert.True(t, tc.s.Equal(tc.want))
		})
	}
}

func TestClone(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		s    set.Set[int]
		want set.Set[int]
	}{
		{"non-empty", set.New(1), set.New(1)},
		{"empty", empty, empty},
		{"zero", zero, empty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.s.Clone()
			assert.True(t, got.Equal(tc.want))
		})
	}
}

func TestNewFromSlice(t *testing.T) {
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

func TestRemove(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		s    set.Set[int]
		v    int
		want set.Set[int]
	}{
		{"element exists", set.New(1, 2), 1, set.New(2)},
		{"element does not exist", set.New(1, 2), 3, set.New(1, 2)},
		{"removing last element", set.New(1), 1, empty},
		{"empty set", empty, 1, empty},
		{"zero set", zero, 1, zero},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.s.Remove(tc.v)
			assert.True(t, tc.s.Equal(tc.want))
		})
	}
}

func TestContainsNormalSet(t *testing.T) {
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

func TestContainsEmptySet(t *testing.T) {
	s := set.New[int]()
	assert.False(t, s.Contains(5))
}

func TestContainsZeroSet(t *testing.T) {
	var s set.Set[int]
	assert.False(t, s.Contains(5))
}

func TestSize(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		s    set.Set[int]
		want int
	}{
		{"non-empty 2 elements", set.New(1, 2), 2},
		{"non-empty 1 element", set.New(1), 1},
		{"empty", empty, 0},
		{"zero", zero, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.s.Size()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEqual(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		a    set.Set[int]
		b    set.Set[int]
		want bool
	}{
		{"non-empty are equal", set.New(1, 2), set.New(1, 2), true},
		{"non-empty are not equal 1", set.New(1, 2), set.New(1), false},
		{"non-empty are not equal 2", set.New(1, 2), set.New(1, 2, 3), false},
		{"non-empty and empty", set.New(1), empty, false},
		{"non-empty and zero", set.New(1), zero, false},
		{"empty sets are equal", empty, empty, true},
		{"zero sets are equal", zero, zero, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got1 := tc.a.Equal(tc.b)
			assert.Equal(t, tc.want, got1)
			got2 := tc.b.Equal(tc.a)
			assert.Equal(t, tc.want, got2)
		})
	}
}

func TestIsSubset2(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		a    set.Set[int]
		b    set.Set[int]
		want bool
	}{
		{"non-empy set is subset of another", set.New(1, 2), set.New(1, 2, 3), true},
		{"non-empy set is not subset of another", set.New(1, 2), set.New(2, 3), false},
		{"sets are subsets of themselves", set.New(1, 2), set.New(1, 2), true},
		{"empty set is subset of non-empty", empty, set.New(1, 2), true},
		{"zero set is subset of non-empty", zero, set.New(1, 2), true},
		{"empty set is subset of itself", empty, empty, true},
		{"zero set is subset of itself", zero, zero, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.a.IsSubset(tc.b)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPop(t *testing.T) {
	t.Run("can pop element when set non-empty", func(t *testing.T) {
		s := set.New(1, 2, 3)
		x, err := s.Pop()
		if assert.NoError(t, err) {
			assert.True(t, set.New(1, 2, 3).Contains(x))
		}
	})
	t.Run("should return error when trying to remove from empty set", func(t *testing.T) {
		s := set.New[int]()
		_, err := s.Pop()
		assert.ErrorIs(t, err, set.ErrNotFound)
	})
	t.Run("should return error when trying to remove from zero set", func(t *testing.T) {
		var s set.Set[int]
		_, err := s.Pop()
		assert.ErrorIs(t, err, set.ErrNotFound)
	})
}

func TestMustPop(t *testing.T) {
	t.Run("can pop element when set non-empty", func(t *testing.T) {
		s := set.New(1, 2, 3)
		x := s.MustPop()
		assert.True(t, set.New(1, 2, 3).Contains(x))
	})
	t.Run("should panic when trying to remove from empty set", func(t *testing.T) {
		s := set.New[int]()
		assert.Panics(t, func() {
			s.MustPop()
		})
	})
	t.Run("should return error when trying to remove from zero set", func(t *testing.T) {
		var s set.Set[int]
		assert.Panics(t, func() {
			s.MustPop()
		})
	})
}

func TestCollect(t *testing.T) {
	t.Run("can create a new set from an iterable", func(t *testing.T) {
		x := set.Collect(slices.Values([]int{1, 2, 3}))
		assert.True(t, set.New(1, 2, 3).Equal(x))
	})
}

func TestConvert(t *testing.T) {
	t.Run("can convert non-empty set to string", func(t *testing.T) {
		s := set.New(42)
		assert.Equal(t, "[42]", fmt.Sprint(s))
	})
	t.Run("can convert empty set to string", func(t *testing.T) {
		s := set.New[int]()
		assert.Equal(t, "[]", fmt.Sprint(s))
	})
	t.Run("can convert zero set to string", func(t *testing.T) {
		var s set.Set[int]
		assert.Equal(t, "[]", fmt.Sprint(s))
	})
}

func TestClear(t *testing.T) {
	t.Run("can clear non-empty set", func(t *testing.T) {
		got := set.New(1, 2)
		got.Clear()
		want := set.New[int]()
		assert.Equal(t, want, got)
	})
	t.Run("can clear empty set", func(t *testing.T) {
		got := set.New[int]()
		got.Clear()
		want := set.New[int]()
		assert.Equal(t, want, got)
	})
	t.Run("can clear zero set", func(t *testing.T) {
		var got, want set.Set[int]
		got.Clear()
		assert.Equal(t, want, got)
	})
}

func TestConvertToSlice(t *testing.T) {
	t.Run("can convert non-empty set to slice", func(t *testing.T) {
		s := set.New(1, 2)
		got := s.ToSlice()
		want := []int{1, 2}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can convert empty set to slice", func(t *testing.T) {
		s := set.New[int]()
		got := s.ToSlice()
		want := []int{}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can convert zero set to slice", func(t *testing.T) {
		var s set.Set[int]
		got := s.ToSlice()
		want := []int{}
		assert.ElementsMatch(t, want, got)
	})
}

func TestIterate(t *testing.T) {
	t.Run("can iterate over non-empty set", func(t *testing.T) {
		s1 := set.New(1, 2, 3)
		s2 := set.New[int]()
		for e := range s1.Values() {
			s2.Add(e)
		}
		assert.Equal(t, s1, s2)
	})
	t.Run("can iterate over empty set", func(t *testing.T) {
		s1 := set.New[int]()
		s2 := set.New[int]()
		for e := range s1.Values() {
			s2.Add(e)
		}
		assert.Equal(t, 0, s2.Size())
	})
	t.Run("can iterate over zero set", func(t *testing.T) {
		var s1 set.Set[int]
		s2 := set.New[int]()
		for e := range s1.Values() {
			s2.Add(e)
		}
		assert.Equal(t, 0, s2.Size())
	})
}

func TestUnion(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		a    set.Set[int]
		b    set.Set[int]
		want set.Set[int]
	}{
		{"non-empy with of another", set.New(1, 2), set.New(2, 3), set.New(1, 2, 3)},
		{"non-empy with itself", set.New(1), set.New(1), set.New(1)},
		{"non-empy with empty", set.New(1), empty, set.New(1)},
		{"empy with itself", empty, empty, empty},
		{"non-empy with zero", set.New(1), zero, set.New(1)},
		{"zero with itself", zero, zero, empty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.a.Union(tc.b)
			assert.True(t, got.Equal(tc.want))
		})
	}
}

func TestIntersec(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		a    set.Set[int]
		b    set.Set[int]
		want set.Set[int]
	}{
		{"non-empy with of another", set.New(1, 2, 3), set.New(2, 3, 4), set.New(2, 3)},
		{"non-empy with itself", set.New(1), set.New(1), set.New(1)},
		{"non-empy with empty", set.New(1), empty, empty},
		{"empy with non-empty", empty, set.New(1), empty},
		{"empy with itself", empty, empty, empty},
		{"non-empy with zero", set.New(1), zero, empty},
		{"zero with non-empty", zero, set.New(1), empty},
		{"zero with itself", zero, zero, empty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got1 := tc.a.Intersect(tc.b)
			assert.True(t, got1.Equal(tc.want))
			got2 := tc.b.Intersect(tc.a)
			assert.True(t, got2.Equal(tc.want))
		})
	}
}

func TestIsDisjoint(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		a    set.Set[int]
		b    set.Set[int]
		want bool
	}{
		{"non-empy set is disjoint of another", set.New(1, 2), set.New(3, 4), true},
		{"non-empy set is not disjoint of another", set.New(1, 2), set.New(2, 3), false},
		{"empty set is disjoint with non-empty set", empty, set.New(1), true},
		{"empty set is disjoint with itself", empty, empty, true},
		{"zero set is disjoint with non-empty set", zero, set.New(1), true},
		{"zero set is disjoint with itself", zero, zero, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got1 := tc.a.IsDisjoint(tc.b)
			assert.Equal(t, tc.want, got1)
			got2 := tc.b.IsDisjoint(tc.a)
			assert.Equal(t, tc.want, got2)
		})
	}
}

func TestSuperset(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		a    set.Set[int]
		b    set.Set[int]
		want bool
	}{
		{"non-empy set is supserset of another", set.New(1, 2, 3), set.New(1, 2), true},
		{"non-empy set is not supserset of another", set.New(1, 2), set.New(1, 2, 3), false},
		{"non-empy set is not supserset of itself", set.New(1), set.New(1), true},
		{"non-empty set is superset of empty set", set.New(1), empty, true},
		{"non-empty set is superset of zero set", set.New(1), zero, true},
		{"empty set is not superset of non-empty set", empty, set.New(1), false},
		{"zero set is not superset of non-empty set", zero, set.New(1), false},
		{"empty set is superset of itself", empty, empty, true},
		{"zero set is superset of itself", zero, zero, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.a.IsSuperset(tc.b)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDifference(t *testing.T) {
	empty := set.New[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		a    set.Set[int]
		b    set.Set[int]
		want set.Set[int]
	}{
		{"non-empy sets without intersection", set.New(1), set.New(2), set.New(1)},
		{"non-empy sets with intersection", set.New(1, 2), set.New(2, 3), set.New(1)},
		{"non-empy sets with empty set", set.New(1), empty, set.New(1)},
		{"empy sets with non-empty set", empty, set.New(1), empty},
		{"non-empy sets with zero set", set.New(1), zero, set.New(1)},
		{"zero sets with non-empty set", zero, set.New(1), empty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.a.Difference(tc.b)
			assert.True(t, got.Equal(tc.want))
		})
	}
}
