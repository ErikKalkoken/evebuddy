package widget

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"
)

func TestFilterChipGroup(t *testing.T) {
	t.Run("can create initial widget", func(t *testing.T) {
		w := NewFilterChipGroup([]string{"a", "b"}, nil)
		window := test.NewWindow(w)
		defer window.Close()
		w.Refresh()
		assert.ElementsMatch(t, []string{"a", "b"}, w.Options)
		assert.ElementsMatch(t, []string{}, w.Selected)
		assert.False(t, w.chips[0].On)
		assert.False(t, w.chips[1].On)
	})
	t.Run("can create widget with selected", func(t *testing.T) {
		w := NewFilterChipGroup([]string{"a", "b"}, nil)
		w.Selected = []string{"b"}
		window := test.NewWindow(w)
		defer window.Close()
		w.Refresh()
		assert.ElementsMatch(t, []string{"a", "b"}, w.Options)
		assert.ElementsMatch(t, []string{"b"}, w.Selected)
		assert.False(t, w.chips[0].On)
		assert.True(t, w.chips[1].On)
	})
	t.Run("can set selected", func(t *testing.T) {
		w := NewFilterChipGroup([]string{"a", "b", "c"}, nil)
		window := test.NewWindow(w)
		defer window.Close()
		w.Refresh()
		w.SetSelected([]string{"b", "c"})
		assert.False(t, w.chips[0].On)
		assert.True(t, w.chips[1].On)
		assert.True(t, w.chips[2].On)
	})
}

func TestDeduplicateSlice(t *testing.T) {
	t.Run("can remove duplicate elements", func(t *testing.T) {
		s := []string{"b", "a", "b"}
		got := deduplicateSlice(s)
		want := []string{"b", "a"}
		assert.Equal(t, want, got)
	})
	t.Run("can process slices with no duplicates", func(t *testing.T) {
		s := []string{"b", "a"}
		got := deduplicateSlice(s)
		want := []string{"b", "a"}
		assert.Equal(t, want, got)
	})
	t.Run("can process empty slice", func(t *testing.T) {
		s := []string{}
		got := deduplicateSlice(s)
		want := []string{}
		assert.Equal(t, want, got)
	})
}
