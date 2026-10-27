package core

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestNavSelector_SetTitle(t *testing.T) {
	test.NewTempApp(t)
	// given
	w := newNavSelector()
	// when
	w.SetTitle("Bruce Wayne")
	// then
	xassert.Equal(t, "Bruce Wayne", w.label.Text)
	xassert.Equal(t, "Bruce Wayne", w.ToolTip())
}

func TestNavSelector_SetMenu(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should show dropdown icon when there is a menu", func(t *testing.T) {
		// given
		w := newNavSelector()
		// when
		w.SetMenu([]*fyne.MenuItem{fyne.NewMenuItem("Alpha", nil)})
		// then
		assert.True(t, w.trailingIcon.Visible())
	})
	t.Run("should hide dropdown icon when menu is removed", func(t *testing.T) {
		// given
		w := newNavSelector()
		w.SetMenu([]*fyne.MenuItem{fyne.NewMenuItem("Alpha", nil)})
		// when
		w.SetMenu(nil)
		// then
		assert.False(t, w.trailingIcon.Visible())
	})
	t.Run("should hide dropdown icon initially", func(t *testing.T) {
		// when
		w := newNavSelector()
		// then
		assert.False(t, w.trailingIcon.Visible())
	})
}

func TestNavSelector_Tapped(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should show menu when tapped", func(t *testing.T) {
		// given
		w := newNavSelector()
		w.SetMenu([]*fyne.MenuItem{fyne.NewMenuItem("Alpha", nil)})
		win := test.NewWindow(w)
		defer win.Close()
		// when
		test.Tap(w)
		// then
		assert.NotNil(t, win.Canvas().Overlays().Top())
	})
	t.Run("should not show menu when tapped without items", func(t *testing.T) {
		// given
		w := newNavSelector()
		win := test.NewWindow(w)
		defer win.Close()
		// when
		test.Tap(w)
		// then
		assert.Nil(t, win.Canvas().Overlays().Top())
	})
}

func TestNavSelector_Hover(t *testing.T) {
	test.NewTempApp(t)
	th := test.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	t.Run("should show hover state when there is a menu", func(t *testing.T) {
		// given
		w := newNavSelector()
		w.SetMenu([]*fyne.MenuItem{fyne.NewMenuItem("Alpha", nil)})
		// when
		w.MouseIn(&desktop.MouseEvent{})
		// then
		xassert.Equal[desktop.Cursor](t, desktop.PointerCursor, w.Cursor())
		xassert.Equal(t, th.Color(theme.ColorNameHover, v), w.background.FillColor)
	})
	t.Run("should not show hover state when there is no menu", func(t *testing.T) {
		// given
		w := newNavSelector()
		// when
		w.MouseIn(&desktop.MouseEvent{})
		// then
		xassert.Equal[desktop.Cursor](t, desktop.DefaultCursor, w.Cursor())
		xassert.Equal(t, th.Color(theme.ColorNameInputBackground, v), w.background.FillColor)
	})
	t.Run("should end hover state when mouse leaves", func(t *testing.T) {
		// given
		w := newNavSelector()
		w.SetMenu([]*fyne.MenuItem{fyne.NewMenuItem("Alpha", nil)})
		w.MouseIn(&desktop.MouseEvent{})
		// when
		w.MouseOut()
		// then
		xassert.Equal[desktop.Cursor](t, desktop.DefaultCursor, w.Cursor())
		xassert.Equal(t, th.Color(theme.ColorNameInputBackground, v), w.background.FillColor)
	})
	t.Run("should show hover state when menu is added while hovered", func(t *testing.T) {
		// given
		w := newNavSelector()
		w.MouseIn(&desktop.MouseEvent{})
		// when
		w.SetMenu([]*fyne.MenuItem{fyne.NewMenuItem("Alpha", nil)})
		// then
		xassert.Equal[desktop.Cursor](t, desktop.PointerCursor, w.Cursor())
		xassert.Equal(t, th.Color(theme.ColorNameHover, v), w.background.FillColor)
	})
}
