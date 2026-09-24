package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNavRail_PanicsWithoutLeadingItems(t *testing.T) {
	assert.Panics(t, func() {
		xwidget.NewNavRail(nil, xwidget.NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A")))
	})
}

func TestNavRail_PanicsWhenItemIsReused(t *testing.T) {
	test.NewTempApp(t)
	a := xwidget.NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	xwidget.NewNavRail([]*xwidget.NavRailItem{a})
	assert.Panics(t, func() {
		xwidget.NewNavRail([]*xwidget.NavRailItem{a})
	})
}

func TestNavRail_SelectsFirstLeadingItemOnCreation(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selectedA int
	a := xwidget.NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	a.OnSelected = func() { selectedA++ }
	b := xwidget.NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))

	nr := xwidget.NewNavRail([]*xwidget.NavRailItem{a, b})
	w := test.NewWindow(nr)
	defer w.Close()

	assert.Equal(t, a, nr.Selected())
	assert.Equal(t, 1, selectedA)
}

func TestNavRail_SelectSwitchesItemAndContent(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selectedB int
	contentA := widget.NewLabel("A")
	contentB := widget.NewLabel("B")
	a := xwidget.NewNavRailItem(theme.HomeIcon(), "A", contentA)
	b := xwidget.NewNavRailItem(theme.HomeIcon(), "B", contentB)
	b.OnSelected = func() { selectedB++ }

	nr := xwidget.NewNavRail([]*xwidget.NavRailItem{a, b})
	w := test.NewWindow(nr)
	defer w.Close()

	nr.Select(b)
	assert.Equal(t, b, nr.Selected())
	assert.Equal(t, 1, selectedB)
	assert.False(t, contentA.Visible())
	assert.True(t, contentB.Visible())
}

func TestNavRail_SelectingCurrentItemAgainFiresOnSelectedAgain(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selected, selectedAgain int
	a := xwidget.NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	a.OnSelected = func() { selected++ }
	a.OnSelectedAgain = func() { selectedAgain++ }

	nr := xwidget.NewNavRail([]*xwidget.NavRailItem{a})
	w := test.NewWindow(nr)
	defer w.Close()

	nr.Select(a)
	assert.Equal(t, 1, selected)
	assert.Equal(t, 1, selectedAgain)
}

func TestNavRail_DisabledItemCanNotBeSelected(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := xwidget.NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := xwidget.NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	nr := xwidget.NewNavRail([]*xwidget.NavRailItem{a, b})
	w := test.NewWindow(nr)
	defer w.Close()

	assert.True(t, nr.ItemEnabled(b))

	nr.DisableItem(b)
	assert.False(t, nr.ItemEnabled(b))
	nr.Select(b)
	assert.Equal(t, a, nr.Selected())

	nr.EnableItem(b)
	assert.True(t, nr.ItemEnabled(b))
	nr.Select(b)
	assert.Equal(t, b, nr.Selected())
}

func TestNavRail_DisablingSelectedItemSelectsFirstEnabledItem(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := xwidget.NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := xwidget.NewNavRailItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	c := xwidget.NewNavRailItem(theme.HomeIcon(), "C", widget.NewLabel("C"))
	nr := xwidget.NewNavRail([]*xwidget.NavRailItem{a, b, c})
	w := test.NewWindow(nr)
	defer w.Close()

	nr.DisableItem(a)
	nr.Select(c)
	nr.DisableItem(c)
	assert.Equal(t, b, nr.Selected())
}

func TestNavRail_TrailingItemsCanBeSelected(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := xwidget.NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	s := xwidget.NewNavRailItem(theme.SettingsIcon(), "Settings", widget.NewLabel("S"))
	nr := xwidget.NewNavRail([]*xwidget.NavRailItem{a}, s)
	w := test.NewWindow(nr)
	defer w.Close()

	assert.Equal(t, a, nr.Selected())
	nr.Select(s)
	assert.Equal(t, s, nr.Selected())
}

func TestNavRail_ForeignItemsAreIgnored(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := xwidget.NewNavRailItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	nr := xwidget.NewNavRail([]*xwidget.NavRailItem{a})
	w := test.NewWindow(nr)
	defer w.Close()

	foreign := xwidget.NewNavRailItem(theme.HomeIcon(), "X", widget.NewLabel("X"))
	assert.NotPanics(t, func() { nr.Select(foreign) })
	assert.NotPanics(t, func() { nr.Select(nil) })
	assert.NotPanics(t, func() { nr.DisableItem(foreign) })
	assert.NotPanics(t, func() { nr.EnableItem(nil) })
	assert.False(t, nr.ItemEnabled(foreign))
	assert.Equal(t, a, nr.Selected())
}
