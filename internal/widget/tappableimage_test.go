package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/fyne-kx/widget"
)

func TestTappableImage_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	image := widget.NewTappableImage(theme.HomeIcon(), nil)
	image.SetFillMode(canvas.ImageFillContain)
	image.SetMinSize(fyne.NewSquareSize(50))
	w := test.NewWindow(image)
	defer w.Close()

	test.AssertImageMatches(t, "tappableimage/default.png", w.Canvas().Capture())
}

func TestTappableImage_CanSetResource(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	image := widget.NewTappableImage(theme.HomeIcon(), nil)
	image.SetFillMode(canvas.ImageFillContain)
	image.SetMinSize(fyne.NewSquareSize(50))
	w := test.NewWindow(image)
	defer w.Close()

	image.SetResource(theme.ComputerIcon())

	test.AssertImageMatches(t, "tappableimage/set_resource.png", w.Canvas().Capture())
}

func TestTappableImage_CanTap(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	var tapped bool
	image := widget.NewTappableImage(theme.HomeIcon(), func() {
		tapped = true
	})
	image.SetFillMode(canvas.ImageFillContain)
	image.SetMinSize(fyne.NewSquareSize(50))
	w := test.NewWindow(image)
	defer w.Close()

	test.Tap(image)
	assert.True(t, tapped)
}

func TestTappableImage_IgnoreTapWhenNoCallback(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	image := widget.NewTappableImage(theme.HomeIcon(), nil)
	image.SetFillMode(canvas.ImageFillContain)
	image.SetMinSize(fyne.NewSquareSize(50))
	w := test.NewWindow(image)
	defer w.Close()

	test.Tap(image)
}
