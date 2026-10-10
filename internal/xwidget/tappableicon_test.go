package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestTappableIcon_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	icon := xwidget.NewTappableIcon(theme.HomeIcon(), nil)
	w := test.NewWindow(icon)
	defer w.Close()

	test.AssertImageMatches(t, "tappableicon/default.png", w.Canvas().Capture())
}

func TestTappableIcon_CanTap(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	var tapped bool
	icon := xwidget.NewTappableIcon(theme.HomeIcon(), func() {
		tapped = true
	})
	w := test.NewWindow(icon)
	defer w.Close()

	test.Tap(icon)
	assert.True(t, tapped)
}

func TestTappableIcon_Disable(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	var tapped bool
	icon := xwidget.NewTappableIcon(theme.HomeIcon(), func() {
		tapped = true
	})
	w := test.NewWindow(icon)
	defer w.Close()

	t.Run("should ignore tap when disabled", func(t *testing.T) {
		tapped = false
		icon.Disable()
		test.Tap(icon)
		assert.True(t, icon.Disabled())
		assert.False(t, tapped)
	})
	t.Run("should accept tap when enabled again", func(t *testing.T) {
		tapped = false
		icon.Disable()
		icon.Enable()
		test.Tap(icon)
		assert.False(t, icon.Disabled())
		assert.True(t, tapped)
	})
}

func TestTappableIcon_IgnoreTapWhenNoCallback(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	icon := xwidget.NewTappableIcon(theme.HomeIcon(), nil)
	w := test.NewWindow(icon)
	defer w.Close()

	test.Tap(icon)
}

func TestTappableIcon_Cursor(t *testing.T) {
	t.Run("pointer when hovered with callback", func(t *testing.T) {
		test.NewTempApp(t)
		icon := xwidget.NewTappableIcon(theme.HomeIcon(), func() {})
		icon.MouseIn(&desktop.MouseEvent{})
		assert.Equal(t, desktop.PointerCursor, icon.Cursor())
		icon.MouseOut()
		assert.Equal(t, desktop.DefaultCursor, icon.Cursor())
	})
	t.Run("no pointer without callback", func(t *testing.T) {
		test.NewTempApp(t)
		icon := xwidget.NewTappableIcon(theme.HomeIcon(), nil)
		icon.MouseIn(&desktop.MouseEvent{})
		assert.Equal(t, desktop.DefaultCursor, icon.Cursor())
	})
}

func TestTappableIcon_SetToolTip(t *testing.T) {
	test.NewTempApp(t)
	icon := xwidget.NewTappableIcon(theme.HomeIcon(), nil)
	icon.SetToolTip("Hello")
	assert.Equal(t, "Hello", icon.ToolTip())
}
