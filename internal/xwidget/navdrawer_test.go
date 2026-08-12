package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNavDrawer_CanCreateBasic(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	drawer := xwidget.NewNavDrawer(
		xwidget.NewNavPage("First", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 1")),
	)
	drawer.MinWidth = 200
	w := test.NewWindow(drawer)
	defer w.Close()
	w.Resize(fyne.NewSize(500, 500))

	test.AssertImageMatches(t, "navdrawer/minimal.png", w.Canvas().Capture())
}

func TestNavDrawer_CanCreateFull(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	drawer := xwidget.NewNavDrawer(
		xwidget.NewNavPage("First", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 1")),
		xwidget.NewNavPage("Second", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 2")),
		xwidget.NewNavPage("Third", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 3")),
		xwidget.NewNavSectionLabel("Section"),
		xwidget.NewNavPage("Forth", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 4")),
	)
	drawer.MinWidth = 200
	w := test.NewWindow(drawer)
	defer w.Close()
	w.Resize(fyne.NewSize(500, 500))

	test.AssertImageMatches(t, "navdrawer/full.png", w.Canvas().Capture())
}
