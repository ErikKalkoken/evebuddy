package set_test

import (
	"iter"
	"slices"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	cases := []struct {
		name string
		s    set.Set[int]
		v    int
		want set.Set[int]
	}{
		{"add to empty", set.Of[int](), 1, set.Of(1)},
		{"add to zero", set.Set[int]{}, 1, set.Of(1)},
		{"add new to non-empty", set.Of(1), 2, set.Of(1, 2)},
		{"add existing to non-empty", set.Of(1), 1, set.Of(1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.s.Add(tc.v)
			assert.True(t, tc.s.Equal(tc.want))
		})
	}
}

func TestAddSeq(t *testing.T) {
	empty := set.Of[int]()
	cases := []struct {
		name string
		s    set.Set[int]
		seq  iter.Seq[int]
		want set.Set[int]
	}{
		{"add many to non-empty", set.Of(1), set.Of(1, 2).All(), set.Of(1, 2)},
		{"add none to non-empty", set.Of(1), empty.All(), set.Of(1)},
		{"add many to empty", set.Of[int](), set.Of(1, 2).All(), set.Of(1, 2)},
		{"add none to empty", set.Of[int](), empty.All(), empty},
		{"add many to zero", set.Of[int](), set.Of(1, 2).All(), set.Of(1, 2)},
		{"add none to zero", set.Set[int]{}, empty.All(), empty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.s.AddSeq(tc.seq)
			assert.Equal(t, tc.want, tc.s)
		})
	}
}

func TestAll(t *testing.T) {
	t.Run("can iterate over non-empty set", func(t *testing.T) {
		s1 := set.Of(1, 2, 3)
		s2 := set.Of[int]()
		for e := range s1.All() {
			s2.Add(e)
		}
		assert.Equal(t, s1, s2)
	})
	t.Run("can iterate over empty set", func(t *testing.T) {
		s1 := set.Of[int]()
		s2 := set.Of[int]()
		for e := range s1.All() {
			s2.Add(e)
		}
		assert.Equal(t, 0, s2.Size())
	})
	t.Run("can iterate over zero set", func(t *testing.T) {
		var s1 set.Set[int]
		s2 := set.Of[int]()
		for e := range s1.All() {
			s2.Add(e)
		}
		assert.Equal(t, 0, s2.Size())
	})
}

func TestClone(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		s    set.Set[int]
		want set.Set[int]
	}{
		{"non-empty", set.Of(1), set.Of(1)},
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

func TestContains(t *testing.T) {
	cases := []struct {
		name string
		s    set.Set[int]
		v    int
		want bool
	}{
		{"non-empty contains element", set.Of(1, 2), 2, true},
		{"non-empty does not contain element", set.Of(1, 2), 3, false},
		{"empty", set.Of[int](), 3, false},
		{"zero", set.Set[int]{}, 3, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := tc.s.Contains(tc.v)
			assert.Equal(t, tc.want, x)
		})
	}
}

func TestContainsAny(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		s    set.Set[int]
		seq  iter.Seq[int]
		want bool
	}{
		{"non-empty contains one element", set.Of(1, 2), set.Of(2, 3).All(), true},
		{"non-empty contains many elements", set.Of(1, 2, 3), set.Of(2, 3, 4).All(), true},
		{"non-empty contains no elements", set.Of(1, 2), set.Of(3, 4).All(), false},
		{"seq with no elements", set.Of(1, 2), set.Of[int]().All(), false},
		{"empty set with non-empty", empty, set.Of(1).All(), false},
		{"zero set with non-empty", zero, set.Of(1).All(), false},
		{"empty set with empty seq", empty, empty.All(), false},
		{"zero set with empty seq", zero, empty.All(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := tc.s.ContainsAny(tc.seq)
			assert.Equal(t, tc.want, x)
		})
	}
}

func TestContainsAll(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		s    set.Set[int]
		seq  iter.Seq[int]
		want bool
	}{
		{"non-empty contains some elements", set.Of(1, 2), set.Of(2, 3).All(), false},
		{"non-empty contains all elements", set.Of(1, 2), set.Of(1, 2).All(), true},
		{"non-empty contains no elements", set.Of(1, 2), set.Of(3, 4).All(), false},
		{"seq with no elements", set.Of(1, 2), empty.All(), true},
		{"empty set with non-empty", empty, set.Of(1).All(), false},
		{"zero set with non-empty", zero, set.Of(1).All(), false},
		{"empty set with empty seq", empty, empty.All(), true},
		{"zero set with empty seq", zero, empty.All(), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := tc.s.ContainsAll(tc.seq)
			assert.Equal(t, tc.want, x)
		})
	}
}

func TestContainsFunc(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	f := func(i int) bool {
		return i%2 == 0
	}
	cases := []struct {
		name string
		s    set.Set[int]
		f    func(int) bool
		want bool
	}{
		{"non-empty contains one", set.Of(1, 2), f, true},
		{"non-empty contains many", set.Of(1, 2, 4), f, true},
		{"non-empty contains none", set.Of(1), f, false},
		{"empty set", empty, f, false},
		{"zero set", zero, f, false},
		{"f is nil", set.Of(1), nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := tc.s.ContainsFunc(tc.f)
			assert.Equal(t, tc.want, x)
		})
	}
}

func TestEqual(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		a    set.Set[int]
		b    set.Set[int]
		want bool
	}{
		{"non-empty equal", set.Of(1, 2), set.Of(1, 2), true},
		{"non-empty not equal 1", set.Of(1, 2), set.Of(1), false},
		{"non-empty not equal 2", set.Of(1, 2), set.Of(1, 2, 3), false},
		{"non-empty not equal 2", set.Of(1, 2), set.Of(2, 3), false},
		{"non-empty and empty", set.Of(1), empty, false},
		{"non-empty and zero", set.Of(1), zero, false},
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

func TestClear(t *testing.T) {
	cases := []struct {
		name string
		s    set.Set[int]
		want set.Set[int]
	}{
		{"non-empty", set.Of(1, 2), set.Of[int]()},
		{"empty", set.Of[int](), set.Of[int]()},
		{"zero", set.Set[int]{}, set.Of[int]()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.s.Clear()
			assert.Equal(t, tc.want, tc.s)
		})
	}
}

func TestDelete(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name     string
		s        set.Set[int]
		v        int
		wantSet  set.Set[int]
		wantBool bool
	}{
		{"element exists", set.Of(1, 2), 1, set.Of(2), true},
		{"element does not exist", set.Of(1, 2), 3, set.Of(1, 2), false},
		{"removing last element", set.Of(1), 1, empty, true},
		{"empty set", empty, 1, empty, false},
		{"zero set", zero, 1, zero, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := tc.s.Delete(tc.v)
			assert.True(t, tc.s.Equal(tc.wantSet))
			assert.Equal(t, tc.wantBool, x)
		})
	}
}

func TestDeleteFunc(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	f := func(i int) bool {
		return i%2 == 0
	}
	cases := []struct {
		name      string
		s         set.Set[int]
		f         func(int) bool
		wantSet   set.Set[int]
		wantCount int
	}{
		{"delete one", set.Of(1, 2), f, set.Of(1), 1},
		{"delete many", set.Of(1, 2, 4), f, set.Of(1), 2},
		{"delete none", set.Of(1), f, set.Of(1), 0},
		{"empty set", empty, f, empty, 0},
		{"zero set", zero, f, zero, 0},
		{"f is nil", set.Of(1), nil, set.Of(1), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := tc.s.DeleteFunc(tc.f)
			assert.True(t, tc.s.Equal(tc.wantSet))
			assert.Equal(t, tc.wantCount, x)
		})
	}
}

func TestDeleteSeq(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name      string
		s         set.Set[int]
		seq       iter.Seq[int]
		wantSet   set.Set[int]
		wantCount int
	}{
		{"non-empty contains one match", set.Of(1, 2), set.Of(2, 3).All(), set.Of(1), 1},
		{"non-empty contains many matches", set.Of(1, 2, 3), set.Of(2, 3, 4).All(), set.Of(1), 2},
		{"non-empty contains no matches", set.Of(1, 2), set.Of(3, 4).All(), set.Of(1, 2), 0},
		{"seq with no elements", set.Of(1, 2), empty.All(), set.Of(1, 2), 0},
		{"empty set with non-empty seq", empty, set.Of(1).All(), empty, 0},
		{"zero set with non-empty seq", zero, set.Of(1).All(), zero, 0},
		{"empty set with empty seq", empty, empty.All(), empty, 0},
		{"zero set with empty seq", zero, empty.All(), zero, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := tc.s.DeleteSeq(tc.seq)
			assert.Equal(t, tc.wantSet, tc.s)
			assert.Equal(t, tc.wantCount, x)
		})
	}
}

func TestOf(t *testing.T) {
	t.Run("can create empty", func(t *testing.T) {
		s := set.Of[int]()
		assert.ElementsMatch(t, []int{}, s.Slice())
	})
	t.Run("can create empty with initial elements", func(t *testing.T) {
		s := set.Of(3, 2, 1)
		assert.ElementsMatch(t, []int{1, 2, 3}, s.Slice())
	})
	t.Run("can create from slice", func(t *testing.T) {
		got := set.Of([]int{1, 2}...)
		want := set.Of(1, 2)
		assert.Equal(t, want, got)
	})
	t.Run("can create from empty slice", func(t *testing.T) {
		got := set.Of([]int{}...)
		want := set.Of[int]()
		assert.Equal(t, want, got)
	})
}

func TestSize(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		s    set.Set[int]
		want int
	}{
		{"non-empty 2 elements", set.Of(1, 2), 2},
		{"non-empty 1 element", set.Of(1), 1},
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

func TestString(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name  string
		s     set.Set[int]
		want1 string
		want2 string
	}{
		{"non-empty 1", set.Of(1), "{1}", ""},
		{"non-empty 2", set.Of(1, 2), "{1 2}", "{2 1}"},
		{"empty", empty, "{}", ""},
		{"zero", zero, "{}", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.s.String()
			assert.Condition(t, func() (success bool) {
				return got == tc.want1 || got == tc.want2
			})
		})
	}
}

func TestSlice(t *testing.T) {
	t.Run("can convert non-empty set to slice", func(t *testing.T) {
		s := set.Of(1, 2)
		got := s.Slice()
		want := []int{1, 2}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can convert empty set to slice", func(t *testing.T) {
		s := set.Of[int]()
		got := s.Slice()
		want := []int{}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can convert zero set to slice", func(t *testing.T) {
		var s set.Set[int]
		got := s.Slice()
		want := []int{}
		assert.ElementsMatch(t, want, got)
	})
}

func TestCollect(t *testing.T) {
	t.Run("can create a new set from an iterable with items", func(t *testing.T) {
		s := set.Collect(slices.Values([]int{1, 2, 3}))
		assert.Equal(t, set.Of(1, 2, 3), s)
	})
	t.Run("can create a new set from an iterable without items", func(t *testing.T) {
		s := set.Collect(slices.Values([]int{}))
		assert.Equal(t, set.Of[int](), s)
	})
}

func TestDifference(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name   string
		s      set.Set[int]
		others []set.Set[int]
		want   set.Set[int]
	}{
		{"non-empty sets without intersection", set.Of(1), []set.Set[int]{set.Of(2)}, set.Of(1)},
		{"non-empty sets with intersection", set.Of(1, 2), []set.Set[int]{set.Of(2, 3)}, set.Of(1)},
		{"non-empty sets with empty set", set.Of(1), []set.Set[int]{empty}, set.Of(1)},
		{"empty sets with non-empty set", empty, []set.Set[int]{set.Of(1)}, empty},
		{"non-empty sets with zero set", set.Of(1), []set.Set[int]{zero}, set.Of(1)},
		{"zero sets with non-empty set", zero, []set.Set[int]{set.Of(1)}, empty},
		{"non-empty sets multiple other non-empty", set.Of(1, 4), []set.Set[int]{set.Of(2, 3), set.Of(4, 5)}, set.Of(1)},
		{"no other set provided", set.Of(1), []set.Set[int]{}, set.Of(1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := set.Difference(tc.s, tc.others...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIntersection(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		sets []set.Set[int]
		want set.Set[int]
	}{
		{"non-empty with another", []set.Set[int]{set.Of(1, 2, 3), set.Of(2, 3, 4)}, set.Of(2, 3)},
		{"non-empty with itself", []set.Set[int]{set.Of(1), set.Of(1)}, set.Of(1)},
		{"non-empty with empty", []set.Set[int]{set.Of(1), empty}, empty},
		{"empty with non-empty", []set.Set[int]{empty, set.Of(1)}, empty},
		{"empty with itself", []set.Set[int]{empty, empty}, empty},
		{"non-empty with zero", []set.Set[int]{set.Of(1), zero}, empty},
		{"zero with non-empty", []set.Set[int]{zero, set.Of(1)}, empty},
		{"zero with itself", []set.Set[int]{zero, zero}, empty},
		{"non-empty with 2 other", []set.Set[int]{set.Of(1, 2, 3), set.Of(2, 3, 4), set.Of(2, 5, 6)}, set.Of(2)},
		{"only one non-empty set provided", []set.Set[int]{set.Of(1, 2, 3)}, empty},
		{"no sets provided", []set.Set[int]{}, empty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := set.Intersection(tc.sets...)
			assert.True(t, got.Equal(tc.want))
			assert.Condition(t, func() (success bool) {
				return got.Equal(tc.want)
			},
				"got=%s, want=%s", got, tc.want,
			)
		})
	}
}

func TestUnion(t *testing.T) {
	empty := set.Of[int]()
	zero := set.Set[int]{}
	cases := []struct {
		name string
		sets []set.Set[int]
		want set.Set[int]
	}{
		{"non-empty with of another", []set.Set[int]{set.Of(1, 2), set.Of(2, 3)}, set.Of(1, 2, 3)},
		{"non-empty with itself", []set.Set[int]{set.Of(1), set.Of(1)}, set.Of(1)},
		{"non-empty with empty", []set.Set[int]{set.Of(1), empty}, set.Of(1)},
		{"empty with itself", []set.Set[int]{empty, empty}, empty},
		{"non-empty with zero", []set.Set[int]{set.Of(1), zero}, set.Of(1)},
		{"zero with itself", []set.Set[int]{zero, zero}, empty},
		{"multiple non-empty", []set.Set[int]{set.Of(1, 2), set.Of(2, 3), set.Of(3, 4)}, set.Of(1, 2, 3, 4)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := set.Union(tc.sets...)
			assert.True(t, got.Equal(tc.want))
		})
	}
}

// TestAssert tests that testify's assert functions work correctly with the Set type.
func TestAssert(t *testing.T) {
	cases := []struct {
		name  string
		a     set.Set[int]
		b     set.Set[int]
		equal bool
	}{
		{"non-empty equal", set.Of(1, 2), set.Of(2, 1), true},
		{"non-empty not-equal 1", set.Of(1, 2), set.Of(2, 1, 3), false},
		{"non-empty not-equal 1", set.Of(1, 2), set.Of(1), false},
		{"non-empty vs. empty", set.Of(1, 2), set.Of[int](), false},
		{"empty", set.Of[int](), set.Of[int](), true},
		{"zero", set.Set[int]{}, set.Set[int]{}, true},
		{"empty vs. zero", set.Of[int](), set.Set[int]{}, false}, // note that this is false
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.equal {
				assert.Equal(t, tc.b, tc.a)
			} else {
				assert.NotEqual(t, tc.b, tc.a)
			}
		})
	}
}
