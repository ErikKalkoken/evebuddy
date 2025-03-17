package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/stretchr/testify/assert"
)

func TestFilterChipGroup(t *testing.T) {
	t.Run("can create widget without selected", func(t *testing.T) {
		w := widget.NewFilterChipGroup([]string{"a", "b"}, nil)
		window := test.NewWindow(w)
		defer window.Close()
		w.Refresh()
		assert.ElementsMatch(t, []string{"a", "b"}, w.Options())
		assert.ElementsMatch(t, []string{}, w.Selected)
	})
	t.Run("can create widget with selected", func(t *testing.T) {
		w := widget.NewFilterChipGroup([]string{"a", "b"}, nil)
		w.Selected = []string{"b"}
		window := test.NewWindow(w)
		defer window.Close()
		w.Refresh()
		assert.ElementsMatch(t, []string{"a", "b"}, w.Options())
		assert.ElementsMatch(t, []string{"b"}, w.Selected)
	})
	t.Run("should panic when selected has invalid value", func(t *testing.T) {
		w := widget.NewFilterChipGroup([]string{"a", "b"}, nil)
		w.Selected = []string{"c"}
		assert.Panics(t, func() {
			test.NewWindow(w)
		})
	})
	t.Run("can set selected", func(t *testing.T) {
		w := widget.NewFilterChipGroup([]string{"a", "b", "c"}, nil)
		window := test.NewWindow(w)
		defer window.Close()
		w.Refresh()
		w.SetSelected([]string{"b", "c"})
		w.Refresh()
	})
}
