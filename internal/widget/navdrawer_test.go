package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

func TestNavDrawer_CanCreateBasic(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	drawer := iwidget.NewNavDrawer(
		nil,
		iwidget.NewNavPage("First", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 1")),
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

	drawer := iwidget.NewNavDrawer(
		nil,
		iwidget.NewNavPage("First", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 1")),
		iwidget.NewNavPage("Second", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 2")),
		iwidget.NewNavSeparator(),
		iwidget.NewNavPage("Third", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 3")),
		iwidget.NewNavSectionLabel("Section"),
		iwidget.NewNavPage("Forth", theme.HomeIcon(), widget.NewLabel("PLACEHOLDER 4")),
	)
	drawer.MinWidth = 200
	w := test.NewWindow(drawer)
	defer w.Close()
	w.Resize(fyne.NewSize(500, 500))

	test.AssertImageMatches(t, "navdrawer/full.png", w.Canvas().Capture())
}
