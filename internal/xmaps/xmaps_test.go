package xmaps_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
)

func TestXMaps(t *testing.T) {
	m := xmaps.OrderedMap[string, int]{"bravo": 1, "alpha": 2}
	var keys []string
	var values []int
	for k, v := range m.All() {
		keys = append(keys, k)
		values = append(values, v)
	}
	assert.Equal(t, []string{"alpha", "bravo"}, keys)
	assert.Equal(t, []int{2, 1}, values)
}

func TestInvert(t *testing.T) {
	t.Run("swaps keys and values", func(t *testing.T) {
		in := map[string]int{"one": 1, "two": 2, "three": 3}

		got := xmaps.Invert(in)

		assert.Equal(t, map[int]string{1: "one", 2: "two", 3: "three"}, got)
	})

	t.Run("empty map returns empty non-nil map", func(t *testing.T) {
		got := xmaps.Invert(map[string]int{})

		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("nil map returns empty non-nil map", func(t *testing.T) {
		var in map[string]int

		got := xmaps.Invert(in)

		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("duplicate values keep exactly one key", func(t *testing.T) {
		in := map[string]int{"a": 1, "b": 1, "c": 2}

		got := xmaps.Invert(in)

		require.Len(t, got, 2)
		assert.Contains(t, []string{"a", "b"}, got[1])
		assert.Equal(t, "c", got[2])
	})

	t.Run("does not modify the input", func(t *testing.T) {
		in := map[string]int{"x": 10, "y": 20}
		want := map[string]int{"x": 10, "y": 20}

		_ = xmaps.Invert(in)

		assert.Equal(t, want, in)
	})

	t.Run("result is independent of the input", func(t *testing.T) {
		in := map[string]int{"x": 10}

		got := xmaps.Invert(in)
		in["y"] = 20

		assert.Equal(t, map[int]string{10: "x"}, got)
	})

	t.Run("works with named map types", func(t *testing.T) {
		type Codes map[string]rune
		in := Codes{"alpha": 'a', "bravo": 'b'}

		got := xmaps.Invert(in)

		assert.Equal(t, map[rune]string{'a': "alpha", 'b': "bravo"}, got)
	})

	t.Run("round trip restores a bijective map", func(t *testing.T) {
		in := map[int]string{1: "a", 2: "b", 3: "c"}

		assert.Equal(t, in, xmaps.Invert(xmaps.Invert(in)))
	})
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		in   []map[string]int
		want map[string]int
	}{
		{
			name: "no maps",
			in:   nil,
			want: map[string]int{},
		},
		{
			name: "single map",
			in:   []map[string]int{{"a": 1}},
			want: map[string]int{"a": 1},
		},
		{
			name: "disjoint keys",
			in:   []map[string]int{{"a": 1}, {"b": 2}},
			want: map[string]int{"a": 1, "b": 2},
		},
		{
			name: "later map wins on conflict",
			in:   []map[string]int{{"a": 1, "b": 2}, {"b": 20}, {"b": 200, "c": 3}},
			want: map[string]int{"a": 1, "b": 200, "c": 3},
		},
		{
			name: "nil and empty maps are skipped",
			in:   []map[string]int{nil, {"a": 1}, {}, nil},
			want: map[string]int{"a": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xmaps.Merge(tt.in...)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMerge_DoesNotModifyInputs(t *testing.T) {
	a := map[string]int{"a": 1, "b": 2}
	b := map[string]int{"b": 20}

	got := xmaps.Merge(a, b)
	got["new"] = 99

	assert.Equal(t, map[string]int{"a": 1, "b": 2}, a)
	assert.Equal(t, map[string]int{"b": 20}, b)
}
