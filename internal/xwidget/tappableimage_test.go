package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestTappableImage_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	image := xwidget.NewTappableImage(theme.HomeIcon(), nil)
	image.SetFillMode(canvas.ImageFillContain)
	image.SetMinSize(fyne.NewSquareSize(50))
	w := test.NewWindow(image)
	defer w.Close()

	test.AssertImageMatches(t, "tappableimage/default.png", w.Canvas().Capture())
}

func TestTappableImage_CanSetResource(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	image := xwidget.NewTappableImage(theme.HomeIcon(), nil)
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
	image := xwidget.NewTappableImage(theme.HomeIcon(), func() {
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
	image := xwidget.NewTappableImage(theme.HomeIcon(), nil)
	image.SetFillMode(canvas.ImageFillContain)
	image.SetMinSize(fyne.NewSquareSize(50))
	w := test.NewWindow(image)
	defer w.Close()

	test.Tap(image)
}

func TestTappableImage_CanCreateWithMenu(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	image := xwidget.NewTappableImageWithMenu(theme.HomeIcon(), menu)
	w := test.NewWindow(container.NewCenter(image))
	defer w.Close()

	test.Tap(image)
	assert.NotNil(t, w.Canvas().Overlays().Top(), "should show menu")
}

func TestTappableImage_NilMenuDoesNotPanic(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var image *xwidget.TappableImage
	assert.NotPanics(t, func() {
		image = xwidget.NewTappableImageWithMenu(theme.HomeIcon(), nil)
	})
	w := test.NewWindow(container.NewCenter(image))
	defer w.Close()

	test.Tap(image)
	assert.Nil(t, w.Canvas().Overlays().Top())
}

func TestTappableImage_SetToolTip(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	image := xwidget.NewTappableImage(theme.HomeIcon(), nil)
	w := test.NewWindow(image)
	defer w.Close()

	image.SetToolTip("Hello")
	assert.Equal(t, "Hello", image.ToolTip())
}
