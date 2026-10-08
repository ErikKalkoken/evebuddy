package core

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

func TestPageHeader(t *testing.T) {
	test.NewTempApp(t)
	newHeader := func(t *testing.T) (*PageHeader, fyne.Window) {
		h := NewPageHeader("Title", nil)
		w := test.NewWindow(container.NewHBox(h))
		t.Cleanup(w.Close)
		return h, w
	}
	items := []*fyne.MenuItem{fyne.NewMenuItem("Alpha", nil)}
	t.Run("should show background when hovered and has menu", func(t *testing.T) {
		h, _ := newHeader(t)
		h.SetMenu(items)

		h.MouseIn(&desktop.MouseEvent{})

		assert.True(t, h.background.Visible())
	})
	t.Run("should hide background after mouse left", func(t *testing.T) {
		h, _ := newHeader(t)
		h.SetMenu(items)
		h.MouseIn(&desktop.MouseEvent{})

		h.MouseOut()

		assert.False(t, h.background.Visible())
	})
	t.Run("should not show background when hovered without menu", func(t *testing.T) {
		h, _ := newHeader(t)

		h.MouseIn(&desktop.MouseEvent{})

		assert.False(t, h.background.Visible())
	})
	t.Run("should hide background when menu removed while hovered", func(t *testing.T) {
		h, _ := newHeader(t)
		h.SetMenu(items)
		h.MouseIn(&desktop.MouseEvent{})

		h.SetMenu(nil)

		assert.False(t, h.background.Visible())
		assert.False(t, h.trailingIcon.Visible())
	})
	t.Run("should show background when menu added while hovered", func(t *testing.T) {
		h, _ := newHeader(t)
		h.MouseIn(&desktop.MouseEvent{})

		h.SetMenu(items)

		assert.True(t, h.background.Visible())
		assert.True(t, h.trailingIcon.Visible())
	})
	t.Run("should open menu when tapped on content", func(t *testing.T) {
		h, w := newHeader(t)
		h.SetMenu(items)

		h.Tapped(&fyne.PointEvent{})

		assert.NotNil(t, w.Canvas().Overlays().Top())
	})
	t.Run("should not open menu when tapped without menu", func(t *testing.T) {
		h, w := newHeader(t)

		h.Tapped(&fyne.PointEvent{})

		assert.Nil(t, w.Canvas().Overlays().Top())
	})
}
