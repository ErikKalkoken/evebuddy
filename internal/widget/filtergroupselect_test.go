package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/stretchr/testify/assert"
)

func TestFilterChipSelect_CanCreateEnabledOff(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChipSelect("Test", []string{"Alpha", "Bravo"}, nil)
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.AssertImageMatches(t, "filterchipselect/enabled_off.png", w.Canvas().Capture())
}

func TestFilterChipSelect_CanCreateEnabledOn(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChipSelect("Test", []string{"Alpha", "Bravo"}, nil)
	chip.Selected = "Alpha"
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.AssertImageMatches(t, "filterchipselect/enabled_on.png", w.Canvas().Capture())
}

func TestFilterChipSelect_CanCreateDisabledOff(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChipSelect("Test", []string{"Alpha", "Bravo"}, nil)
	chip.Disable()
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.AssertImageMatches(t, "filterchipselect/disabled_off.png", w.Canvas().Capture())
}

func TestFilterChipSelect_CanCreateDisabledOn(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChipSelect("Test", []string{"Alpha", "Bravo"}, nil)
	chip.Selected = "Alpha"
	chip.Disable()
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.AssertImageMatches(t, "filterchipselect/disabled_on.png", w.Canvas().Capture())
}
func TestFilterChipSelect(t *testing.T) {
	test.NewTempApp(t)
	t.Run("options are deduplicated", func(t *testing.T) {
		x := widget.NewFilterChipSelect("placeholder", []string{"b", "a", "b"}, nil)
		assert.Equal(t, []string{"b", "a"}, x.Options)
	})
}

func TestFilterChipSelectSetSelected(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can select an option", func(t *testing.T) {
		x := widget.NewFilterChipSelect("placeholder", []string{"a", "b"}, nil)
		x.SetSelected("a")
		assert.Equal(t, "a", x.Selected)
		assert.Equal(t, []string{"a", "b"}, x.Options)
	})
	t.Run("selecting invalid option is ignored", func(t *testing.T) {
		x := widget.NewFilterChipSelect("placeholder", []string{"a", "b"}, nil)
		x.SetSelected("a")
		x.SetSelected("x")
		assert.Equal(t, "a", x.Selected)
	})
	t.Run("can clear selection", func(t *testing.T) {
		// given
		x := widget.NewFilterChipSelect("placeholder", []string{"a", "b"}, nil)
		x.SetSelected("a")
		// when
		x.SetSelected("")
		// then
		assert.Equal(t, "", x.Selected)
	})
	t.Run("can not clear selection when no placeholder", func(t *testing.T) {
		// given
		x := widget.NewFilterChipSelect("", []string{"a", "b"}, nil)
		x.SetSelected("a")
		// when
		x.SetSelected("")
		// then
		assert.Equal(t, "a", x.Selected)
	})
	t.Run("selecting an option triggers callback when changed", func(t *testing.T) {
		var isCalled bool
		var v string
		x := widget.NewFilterChipSelect("placeholder", []string{"a", "b"}, func(selected string) {
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
		x := widget.NewFilterChipSelect("placeholder", []string{"a", "b"}, func(selected string) {
			isCalled = true
		})
		x.SetSelected("a")
		isCalled = false
		// when
		x.SetSelected("a")
		// then
		assert.False(t, isCalled)
	})
	t.Run("options are deduplicated, but not sorted when disabled", func(t *testing.T) {
		x := widget.NewFilterChipSelect("", []string{"b", "a", "b"}, nil)
		assert.Equal(t, []string{"b", "a"}, x.Options)
	})
}

func TestFilterChipSelectClearSelected(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can clear selection", func(t *testing.T) {
		// given
		x := widget.NewFilterChipSelect("placeholder", []string{"a", "b"}, nil)
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
		x := widget.NewFilterChipSelect("placeholder", []string{"a", "b", "c"}, func(selected string) {
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
	t.Run("options are deduplicated when set", func(t *testing.T) {
		x := widget.NewFilterChipSelect("placeholder", []string{}, nil)
		x.SetOptions([]string{"b", "a", "b", "a"})
		assert.Equal(t, []string{"b", "a"}, x.Options)
	})
	t.Run("selection is cleared when no longer valid", func(t *testing.T) {
		// given
		x := widget.NewFilterChipSelect("placeholder", []string{"c"}, nil)
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
	t.Run("options are deduplicated", func(t *testing.T) {
		x := widget.NewFilterChipSelectWithSearch("placeholder", []string{"b", "a", "b", "a"}, nil, w)
		assert.Equal(t, []string{"b", "a"}, x.Options)
	})
}

func TestFilterChipSelect_CanShowDropDown(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChipSelect("Test", []string{"Bravo", "Alpha"}, nil)
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()
	w.Resize(fyne.NewSize(100, 200))

	test.Tap(chip)

	test.AssertImageMatches(t, "filterchipselect/dropdown_sorted.png", w.Canvas().Capture())
}

func TestFilterChipSelect_CanDisableOptionSort(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChipSelect("Test", []string{"Bravo", "Alpha"}, nil)
	chip.SortDisabled = true
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()
	w.Resize(fyne.NewSize(100, 200))

	test.Tap(chip)

	test.AssertImageMatches(t, "filterchipselect/dropdown_unsorted.png", w.Canvas().Capture())
}

func TestFilterChipSelect_CanShowSearchBox(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(500, 200))
	chip := widget.NewFilterChipSelectWithSearch("Test", []string{"Bravo", "Alpha"}, nil, w)
	w.SetContent(container.NewCenter(chip))

	test.Tap(chip)

	test.AssertImageMatches(t, "filterchipselect/search_sorted.png", w.Canvas().Capture())
}

func TestFilterChipSelect_CanShowSearchBoxUnsorted(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(500, 200))
	chip := widget.NewFilterChipSelectWithSearch("Test", []string{"Bravo", "Alpha"}, nil, w)
	chip.SortDisabled = true
	w.SetContent(container.NewCenter(chip))

	test.Tap(chip)

	test.AssertImageMatches(t, "filterchipselect/search_unsorted.png", w.Canvas().Capture())
}
