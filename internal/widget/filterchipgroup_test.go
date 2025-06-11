package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/widget"
)

func TestFilterChipGroup(t *testing.T) {
	a := test.NewTempApp(t)
	t.Run("options are deduplicated and cleaned", func(t *testing.T) {
		x := widget.NewFilterChipGroup([]string{"a", "c", "c", "", "d"}, nil)
		want := []string{"a", "c", "d"}
		assert.Equal(t, want, x.Options)
	})
	t.Run("can set initial selection", func(t *testing.T) {
		x := widget.NewFilterChipGroup([]string{"a", "b"}, nil)
		x.Selected = []string{"a"}
		w := a.NewWindow("xxx")
		w.SetContent(x)
		w.Show()
		assert.Equal(t, []string{"a"}, x.Selected)
	})
}

func TestFilterChipGroupSetSelection(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can set new selection", func(t *testing.T) {
		x := widget.NewFilterChipGroup([]string{"a", "b"}, nil)
		x.SetSelected([]string{"b"})
		assert.Equal(t, []string{"b"}, x.Selected)
	})
	t.Run("can ignore invalid elements in selection", func(t *testing.T) {
		x := widget.NewFilterChipGroup([]string{"a", "b"}, nil)
		x.SetSelected([]string{"b", "c", ""})
		assert.Equal(t, []string{"b"}, x.Selected)
	})
}
