package xwidget

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
)

func TestNavRail_TappingItemSelectsIt(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	nr := NewNavRail([]*NavRailItem{a, b})
	w := test.NewWindow(nr)
	defer w.Close()

	test.Tap(b.dest)
	assert.Equal(t, b, nr.Selected())

	nr.DisableItem(a)
	test.Tap(a.dest)
	assert.Equal(t, b, nr.Selected())
}

func TestNavRail_DestinationSizeIsStableOnHoverAndSelect(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	nr := NewNavRail([]*NavRailItem{a, b})
	w := test.NewWindow(nr)
	defer w.Close()

	want := b.dest.MinSize()
	b.dest.MouseIn(&desktop.MouseEvent{})
	assert.Equal(t, want, b.dest.MinSize())
	b.dest.MouseOut()
	nr.Select(b)
	assert.Equal(t, want, b.dest.MinSize())
}

func TestNavRail_TappingMenuItemShowsMenuAndKeepsSelection(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	m := NewNavRailMenuItem(theme.MenuIcon(), "Menu", fyne.NewMenu("", fyne.NewMenuItem("X", nil)))
	nr := NewNavRail([]*NavRailItem{a}, m)
	w := test.NewWindow(nr)
	defer w.Close()

	test.Tap(m.dest)
	assert.Equal(t, a, nr.Selected())
	assert.NotEmpty(t, w.Canvas().Overlays().List())
}

func TestNavRail_TappingActionItem(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var tapped int
	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	x := NewNavRailActionItem(theme.SearchIcon(), "Search", func() { tapped++ })
	nr := NewNavRail([]*NavRailItem{a}, x)
	w := test.NewWindow(nr)
	defer w.Close()

	test.Tap(x.dest)
	assert.Equal(t, 1, tapped)
	assert.Equal(t, a, nr.Selected())

	nr.DisableItem(x)
	test.Tap(x.dest)
	assert.Equal(t, 1, tapped)
}

func TestNavRail_IndicatorFollowsSelection(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	c := NewNavRailItem(theme.SettingsIcon(), "C", widget.NewLabel("C"))
	nr := NewNavRail([]*NavRailItem{a, b}, c)
	w := test.NewWindow(nr)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 400))

	assertIndicatorAt(t, nr, a)

	nr.Select(b)
	assertIndicatorAt(t, nr, b)

	nr.Select(c)
	assertIndicatorAt(t, nr, c)

	nr.DisableItem(c)
	assertIndicatorAt(t, nr, a)
}

func TestNavRail_Placeholder(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	t.Run("shown when all items are disabled", func(t *testing.T) {
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		x := NewNavRailActionItem(theme.SearchIcon(), "Search", func() {})
		m := NewNavRailMenuItem(theme.MenuIcon(), "Menu", fyne.NewMenu("", fyne.NewMenuItem("X", nil)))
		nr := NewNavRail([]*NavRailItem{a, b}, x, m)
		nr.SetPlaceholder("placeholder")
		p := nr.placeholder
		nr.DisableItem(a) // before the window is shown, like at app startup
		nr.DisableItem(b)
		w := test.NewWindow(nr)
		defer w.Close()

		assert.True(t, p.Visible())
		assert.False(t, a.content.Visible())
		assert.False(t, b.content.Visible())
		assert.Nil(t, nr.Selected())
		_, _, _, alpha := nr.indicator.FillColor.RGBA()
		assert.Zero(t, alpha)
		assert.True(t, nr.ItemEnabled(m), "menu item stays enabled")

		nr.EnableItem(b)
		assert.False(t, p.Visible())
		assert.True(t, b.content.Visible())
		assert.Equal(t, b, nr.Selected())
		assert.Equal(t, theme.Color(theme.ColorNamePrimary), nr.indicator.FillColor)
	})
	t.Run("selects first enabled item instead of previous selection", func(t *testing.T) {
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		nr := NewNavRail([]*NavRailItem{a, b})
		nr.SetPlaceholder("placeholder")
		p := nr.placeholder
		w := test.NewWindow(nr)
		defer w.Close()
		nr.Select(b)

		nr.DisableItem(a)
		nr.DisableItem(b)
		assert.False(t, b.dest.isActive)
		nr.EnableItem(a)
		nr.EnableItem(b)
		assert.Equal(t, a, nr.Selected())
		assert.True(t, a.content.Visible())
		assert.False(t, b.content.Visible())
		assert.False(t, p.Visible())
	})
	t.Run("set while empty", func(t *testing.T) {
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		nr := NewNavRail([]*NavRailItem{a})
		w := test.NewWindow(nr)
		defer w.Close()
		nr.DisableItem(a)
		assert.False(t, a.content.Visible())
		assert.True(t, nr.placeholder.Visible())

		nr.SetPlaceholder("placeholder")
		assert.Equal(t, "placeholder", nr.placeholder.text.String())
		seg := nr.placeholder.text.Segments[0].(*widget.TextSegment)
		assert.Equal(t, theme.ColorNamePlaceHolder, seg.Style.ColorName)
	})
	t.Run("empty text shows nothing", func(t *testing.T) {
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		nr := NewNavRail([]*NavRailItem{a})
		nr.SetPlaceholder("placeholder")

		nr.SetPlaceholder("")
		assert.Empty(t, nr.placeholder.text.String())
	})
}

func TestNavRail_OnSelectedFiresOnlyWhenItemIsSelected(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selected int
	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	a.OnSelected = func() { selected++ }
	nr := NewNavRail([]*NavRailItem{a, b})
	w := test.NewWindow(nr)
	defer w.Close()
	assert.Equal(t, 1, selected)

	nr.DisableItem(a)
	nr.DisableItem(b)
	assert.Equal(t, 1, selected, "not fired when becoming empty")

	nr.EnableItem(a)
	assert.Equal(t, 2, selected, "fired when re-enabled")

	nr.EnableItem(b)
	assert.Equal(t, 2, selected, "not fired when selection is unchanged")
}

func TestNavRail_IndicatorFollowsTrailingItemOnResize(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	c := NewNavRailItem(theme.SettingsIcon(), "C", widget.NewLabel("C"))
	nr := NewNavRail([]*NavRailItem{a}, c)
	w := test.NewWindow(nr)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 400))
	nr.Select(c)

	w.Resize(fyne.NewSize(300, 500))
	assertIndicatorAt(t, nr, c)
}

// assertIndicatorAt asserts that the indicator sits on the separator next to it.
func assertIndicatorAt(t *testing.T, nr *NavRail, it *NavRailItem) {
	t.Helper()
	d := fyne.CurrentApp().Driver()
	origin := d.AbsolutePositionForObject(nr)
	dest := d.AbsolutePositionForObject(it.dest).Subtract(origin)
	sep := d.AbsolutePositionForObject(nr.separator).Subtract(origin)
	assert.Equal(t, fyne.NewPos(sep.X, dest.Y), nr.indicator.Position())
	assert.Equal(t, it.dest.Size().Height, nr.indicator.Size().Height)
	assert.Equal(t, theme.Size(theme.SizeNameSeparatorThickness), nr.indicator.Size().Width)
	assert.Equal(t, theme.Color(theme.ColorNamePrimary), nr.indicator.FillColor)
}

func TestNavRail_LabelColorFollowsState(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	t.Run("shows the color for each state", func(t *testing.T) {
		// given
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		c := NewNavRailItem(theme.HomeIcon(), "C", widget.NewLabel("C"))
		nr := NewNavRail([]*NavRailItem{a, b, c})
		w := test.NewWindow(nr)
		defer w.Close()

		// when
		nr.DisableItem(c)

		// then
		assert.Equal(t, "A", a.dest.label.Text)
		assert.Equal(t, theme.Color(theme.ColorNamePrimary), a.dest.label.Color)
		assert.Equal(t, theme.Color(theme.ColorNameForeground), b.dest.label.Color)
		assert.Equal(t, theme.Color(theme.ColorNameDisabled), c.dest.label.Color)
	})

	t.Run("resets the color when deselected", func(t *testing.T) {
		// given
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		nr := NewNavRail([]*NavRailItem{a, b})
		w := test.NewWindow(nr)
		defer w.Close()

		// when
		nr.Select(b)

		// then
		assert.Equal(t, theme.Color(theme.ColorNameForeground), a.dest.label.Color)
		assert.Equal(t, theme.Color(theme.ColorNamePrimary), b.dest.label.Color)
	})
}

func TestNavRail_Tooltip(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	x := NewNavRailActionItem(theme.SearchIcon(), "Search", func() {})
	x.Tooltip = "Search everything"
	nr := NewNavRail([]*NavRailItem{a}, x)
	w := test.NewWindow(nr)
	defer w.Close()

	assert.Empty(t, a.dest.ToolTip())
	assert.Equal(t, "Search everything", x.dest.ToolTip())
}

func TestNavRail_Hover(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	t.Run("shows background and pointer while hovered", func(t *testing.T) {
		// given
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		nr := NewNavRail([]*NavRailItem{a, b})
		w := test.NewWindow(nr)
		defer w.Close()

		// when
		b.dest.MouseIn(&desktop.MouseEvent{})

		// then
		assert.Equal(t, theme.Color(theme.ColorNameHover), b.dest.hover.FillColor)
		assert.Equal(t, desktop.PointerCursor, b.dest.Cursor())
	})

	t.Run("clears background and pointer on mouse out", func(t *testing.T) {
		// given
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		nr := NewNavRail([]*NavRailItem{a, b})
		w := test.NewWindow(nr)
		defer w.Close()
		b.dest.MouseIn(&desktop.MouseEvent{})

		// when
		b.dest.MouseOut()

		// then
		assert.Equal(t, color.Transparent, b.dest.hover.FillColor)
		assert.Equal(t, desktop.DefaultCursor, b.dest.Cursor())
	})

	t.Run("shows no background and no pointer when disabled", func(t *testing.T) {
		// given
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		nr := NewNavRail([]*NavRailItem{a, b})
		w := test.NewWindow(nr)
		defer w.Close()
		nr.DisableItem(b)

		// when
		b.dest.MouseIn(&desktop.MouseEvent{})

		// then
		assert.Equal(t, color.Transparent, b.dest.hover.FillColor)
		assert.Equal(t, desktop.DefaultCursor, b.dest.Cursor())
	})

	t.Run("clears background and pointer when disabled while hovered", func(t *testing.T) {
		// given
		a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		nr := NewNavRail([]*NavRailItem{a, b})
		w := test.NewWindow(nr)
		defer w.Close()
		b.dest.MouseIn(&desktop.MouseEvent{})

		// when
		nr.DisableItem(b)

		// then
		assert.Equal(t, color.Transparent, b.dest.hover.FillColor)
		assert.Equal(t, desktop.DefaultCursor, b.dest.Cursor())
	})
}

func TestNavRail_ItemWithoutLabelShowsOnlyIcon(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	// given
	a := NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavRailItem(theme.HomeIcon(), "", widget.NewLabel("B"))

	// when
	nr := NewNavRail([]*NavRailItem{a, b})
	w := test.NewWindow(nr)
	defer w.Close()

	// then
	got := b.dest.MinSize()
	assert.Less(t, got.Height, a.dest.MinSize().Height)
	assert.Equal(t, got.Width, got.Height)
}
