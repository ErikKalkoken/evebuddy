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
