package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
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

	test.Tap(b.Icon)
	assert.True(t, tapped)
}

func TestIconButton_CanCreateWithMenu(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	b := xwidget.NewIconButtonWithMenu(theme.HomeIcon(), menu)
	w := test.NewWindow(b)
	defer w.Close()

	assert.NotNil(t, b.Icon)
	assert.NotPanics(t, func() {
		test.Tap(b.Icon)
	})
}

func TestIconButton_SetToolTip(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	b := xwidget.NewIconButton(theme.HomeIcon(), nil)
	w := test.NewWindow(b)
	defer w.Close()

	b.SetToolTip("Hello")
	assert.Equal(t, "Hello", b.Icon.ToolTip())
}
