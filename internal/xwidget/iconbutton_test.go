package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestIconButton_CanCreateAndTap(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var tapped bool
	b := xwidget.NewIconButton(theme.HomeIcon(), func() {
		tapped = true
	})
	w := test.NewWindow(b)
	defer w.Close()

	test.Tap(b)
	assert.True(t, tapped)
}

func TestIconButton_CanCreateWithMenu(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	b := xwidget.NewIconButtonWithMenu(theme.HomeIcon(), menu)
	w := test.NewWindow(container.NewCenter(b))
	defer w.Close()

	test.Tap(b)
	assert.NotNil(t, w.Canvas().Overlays().Top(), "should show menu")
}

func TestIconButton_SetToolTip(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	b := xwidget.NewIconButton(theme.HomeIcon(), nil)
	w := test.NewWindow(b)
	defer w.Close()

	b.SetToolTip("Hello")
	assert.Equal(t, "Hello", b.ToolTip())
}
