package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/stretchr/testify/assert"
)

func TestFilterChip(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can set state", func(t *testing.T) {
		x := iwidget.NewFilterChip("dummy", nil)
		x.SetState(true)
		assert.True(t, x.On)
		x.SetState(false)
		assert.False(t, x.On)
	})
	t.Run("cb is called on change", func(t *testing.T) {
		var isCalled, v bool
		x := iwidget.NewFilterChip("dummy", func(on bool) {
			isCalled = true
			v = on
		})
		x.SetState(true)
		assert.True(t, isCalled)
		assert.True(t, v)
	})
}

func TestFilterChipSelect(t *testing.T) {
	test.NewTempApp(t)
	t.Run("options are deduplicated and sorted", func(t *testing.T) {
		x := iwidget.NewFilterChipSelect("placeholder", []string{"b", "a", "b"}, nil)
		assert.Equal(t, []string{"a", "b"}, x.Options)
	})
}

func TestFilterChipSelectSetSelected(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can select an option", func(t *testing.T) {
		x := iwidget.NewFilterChipSelect("placeholder", []string{"a", "b"}, nil)
		x.SetSelected("a")
		assert.Equal(t, "a", x.Selected)
		assert.Equal(t, []string{"a", "b"}, x.Options)
	})
	t.Run("selecting invalid option is ignored", func(t *testing.T) {
		x := iwidget.NewFilterChipSelect("placeholder", []string{"a", "b"}, nil)
		x.SetSelected("a")
		x.SetSelected("x")
		assert.Equal(t, "a", x.Selected)
	})
	t.Run("can clear selection", func(t *testing.T) {
		// given
		x := iwidget.NewFilterChipSelect("placeholder", []string{"a", "b"}, nil)
		x.SetSelected("a")
		// when
		x.SetSelected("")
		// then
		assert.Equal(t, "", x.Selected)
	})
	t.Run("can not clear selection when no placeholder", func(t *testing.T) {
		// given
		x := iwidget.NewFilterChipSelect("", []string{"a", "b"}, nil)
		x.SetSelected("a")
		// when
		x.SetSelected("")
		// then
		assert.Equal(t, "a", x.Selected)
	})
	t.Run("selecting an option triggers callback when changed", func(t *testing.T) {
		var isCalled bool
		var v string
		x := iwidget.NewFilterChipSelect("placeholder", []string{"a", "b"}, func(selected string) {
			isCalled = true
			v = selected
		})
		x.SetSelected("a")
		assert.True(t, isCalled)
		assert.Equal(t, "a", v)
	})
	t.Run("selecting an option does not trigger callback when not changed", func(t *testing.T) {
		// given
		var isCalled bool
		x := iwidget.NewFilterChipSelect("placeholder", []string{"a", "b"}, func(selected string) {
			isCalled = true
		})
		x.SetSelected("a")
		isCalled = false
		// when
		x.SetSelected("a")
		// then
		assert.False(t, isCalled)
	})
	t.Run("options are deduplicated, but not sorted when there is no placeholder", func(t *testing.T) {
		x := iwidget.NewFilterChipSelect("", []string{"b", "a", "b"}, nil)
		assert.Equal(t, []string{"b", "a"}, x.Options)
	})

}

func TestFilterChipSelectClearSelected(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can clear selection", func(t *testing.T) {
		// given
		x := iwidget.NewFilterChipSelect("placeholder", []string{"a", "b"}, nil)
		x.SetSelected("a")
		// when
		x.ClearSelected()
		// then
		assert.Equal(t, "", x.Selected)
	})
	t.Run("clearing selection triggers callback", func(t *testing.T) {
		// given
		var isCalled bool
		var v string
		x := iwidget.NewFilterChipSelect("placeholder", []string{"a", "b", "c"}, func(selected string) {
			isCalled = true
			v = selected
		})
		x.SetSelected("a")
		isCalled = false
		v = "xx"
		// when
		x.ClearSelected()
		// then
		assert.True(t, isCalled)
		assert.Equal(t, "", v)
	})
}

func TestFilterChipSelectedSetOptions(t *testing.T) {
	test.NewTempApp(t)
	t.Run("options are sorted and deduplicated when set", func(t *testing.T) {
		x := iwidget.NewFilterChipSelect("placeholder", []string{}, nil)
		x.SetOptions([]string{"b", "a", "b", "a"})
		assert.Equal(t, []string{"a", "b"}, x.Options)
	})
	t.Run("selection is cleared when no longer valid", func(t *testing.T) {
		// given
		x := iwidget.NewFilterChipSelect("placeholder", []string{"c"}, nil)
		x.SetSelected("c")
		// when
		x.SetOptions([]string{"a"})
		// then
		assert.Equal(t, "", x.Selected)
	})
}

func TestFilterChipSelectedWithSearch(t *testing.T) {
	a := test.NewTempApp(t)
	w := a.NewWindow("Dummy")
	t.Run("options are sorted and deduplicated", func(t *testing.T) {
		x := iwidget.NewFilterChipSelectWithSearch("placeholder", []string{"b", "a", "b", "a"}, nil, w)
		assert.Equal(t, []string{"a", "b"}, x.Options)
	})
}

func TestFilterChipGroup(t *testing.T) {
	a := test.NewTempApp(t)
	t.Run("options are deduplicated and cleaned", func(t *testing.T) {
		x := iwidget.NewFilterChipGroup([]string{"a", "c", "c", "", "d"}, nil)
		want := []string{"a", "c", "d"}
		assert.Equal(t, want, x.Options)
	})
	t.Run("can set initial selection", func(t *testing.T) {
		x := iwidget.NewFilterChipGroup([]string{"a", "b"}, nil)
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
		x := iwidget.NewFilterChipGroup([]string{"a", "b"}, nil)
		x.SetSelected([]string{"b"})
		assert.Equal(t, []string{"b"}, x.Selected)
	})
	t.Run("can ignore invalid elements in selection", func(t *testing.T) {
		x := iwidget.NewFilterChipGroup([]string{"a", "b"}, nil)
		x.SetSelected([]string{"b", "c", ""})
		assert.Equal(t, []string{"b"}, x.Selected)
	})
}
