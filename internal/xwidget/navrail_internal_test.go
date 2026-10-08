package xwidget

import (
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
