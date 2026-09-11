package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestContextMenuButton_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	b := xwidget.NewContextMenuButton("Test", menu)
	w := test.NewWindow(b)
	defer w.Close()

	assert.Equal(t, "Test", b.Text)
}

func TestContextMenuButton_CanCreateWithIcon(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	b := xwidget.NewContextMenuButtonWithIcon("Test", theme.HomeIcon(), menu)
	w := test.NewWindow(b)
	defer w.Close()

	assert.Equal(t, "Test", b.Text)
	assert.Equal(t, theme.HomeIcon(), b.Icon)
}

func TestContextMenuButton_TapShowsMenuWithoutPanic(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	b := xwidget.NewContextMenuButton("Test", menu)
	w := test.NewWindow(b)
	defer w.Close()

	assert.NotPanics(t, func() {
		test.Tap(b)
	})
}
