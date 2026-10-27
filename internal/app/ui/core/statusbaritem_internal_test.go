package core

import (
	"testing"

	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestStatusBarItem_Tapped(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should call OnTapped when tapped", func(t *testing.T) {
		// given
		var called bool
		w := NewStatusBarItem(theme.InfoIcon(), "text", func() {
			called = true
		})
		// when
		test.Tap(w)
		// then
		assert.True(t, called)
	})
	t.Run("should do nothing when tapped without OnTapped", func(t *testing.T) {
		// given
		w := NewStatusBarItem(theme.InfoIcon(), "text", nil)
		// when
		test.Tap(w)
	})
}

func TestStatusBarItem_Hover(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should show hover background when tappable", func(t *testing.T) {
		// given
		w := NewStatusBarItem(theme.InfoIcon(), "text", func() {})
		// when
		w.MouseIn(&desktop.MouseEvent{})
		// then
		assert.True(t, w.hoover.Visible())
	})
	t.Run("should not show hover background when not tappable", func(t *testing.T) {
		// given
		w := NewStatusBarItem(theme.InfoIcon(), "text", nil)
		// when
		w.MouseIn(&desktop.MouseEvent{})
		// then
		assert.False(t, w.hoover.Visible())
	})
	t.Run("should hide hover background when mouse leaves", func(t *testing.T) {
		// given
		w := NewStatusBarItem(theme.InfoIcon(), "text", func() {})
		w.MouseIn(&desktop.MouseEvent{})
		// when
		w.MouseOut()
		// then
		assert.False(t, w.hoover.Visible())
	})
}

func TestStatusBarItem_SetLeading(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should show new icon", func(t *testing.T) {
		// given
		w := NewStatusBarItem(nil, "text", nil)
		// when
		w.SetLeading(theme.InfoIcon())
		// then
		assert.True(t, w.leadingIcon.Visible())
		xassert.Equal(t, theme.InfoIcon(), w.leadingIcon.Resource)
	})
	t.Run("should hide icon when nil", func(t *testing.T) {
		// given
		w := NewStatusBarItem(theme.InfoIcon(), "text", nil)
		// when
		w.SetLeading(nil)
		// then
		assert.False(t, w.leadingIcon.Visible())
	})
	t.Run("should hide icon when created without one", func(t *testing.T) {
		// when
		w := NewStatusBarItem(nil, "text", nil)
		// then
		assert.False(t, w.leadingIcon.Visible())
	})
}

func TestStatusBarItem_SetText(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should set text and importance", func(t *testing.T) {
		// given
		w := NewStatusBarItem(nil, "old", nil)
		// when
		w.SetTextAndImportance("new", widget.DangerImportance)
		// then
		xassert.Equal(t, "new", w.label.Text)
		xassert.Equal(t, widget.DangerImportance, w.label.Importance)
	})
	t.Run("should reset importance to medium when setting text", func(t *testing.T) {
		// given
		w := NewStatusBarItem(nil, "old", nil)
		w.SetTextAndImportance("old", widget.DangerImportance)
		// when
		w.SetText("new")
		// then
		xassert.Equal(t, "new", w.label.Text)
		xassert.Equal(t, widget.MediumImportance, w.label.Importance)
	})
}

func TestStatusBarItem_Trailing(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should show trailing object", func(t *testing.T) {
		// given
		trailing := widget.NewLabel("trailing")
		// when
		w := NewStatusBarItemWithTrailing(nil, trailing, "text", nil)
		// then
		assert.Same(t, trailing, w.trailing)
		assert.True(t, w.trailing.Visible())
	})
	t.Run("should hide placeholder when there is no trailing object", func(t *testing.T) {
		// when
		w := NewStatusBarItem(nil, "text", nil)
		// then
		assert.False(t, w.trailing.Visible())
	})
}
