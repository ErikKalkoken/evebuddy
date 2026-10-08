package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestTappableLabel_CanTap(t *testing.T) {
	test.NewTempApp(t)
	var tapped bool
	label := xwidget.NewTappableLabel("Test", func() {
		tapped = true
	})
	w := test.NewWindow(label)
	defer w.Close()

	test.Tap(label)

	assert.True(t, tapped)
	assert.Equal(t, "Test", label.Text)
}

func TestTappableLabel_Cursor(t *testing.T) {
	t.Run("pointer when hovered with callback", func(t *testing.T) {
		test.NewTempApp(t)
		label := xwidget.NewTappableLabel("Test", func() {})
		label.MouseIn(&desktop.MouseEvent{})
		assert.Equal(t, desktop.PointerCursor, label.Cursor())
		label.MouseOut()
		assert.Equal(t, desktop.DefaultCursor, label.Cursor())
	})
	t.Run("no pointer without callback", func(t *testing.T) {
		test.NewTempApp(t)
		label := xwidget.NewTappableLabel("Test", nil)
		label.MouseIn(&desktop.MouseEvent{})
		assert.Equal(t, desktop.DefaultCursor, label.Cursor())
	})
}

func TestTappableLabel_SetToolTip(t *testing.T) {
	test.NewTempApp(t)
	label := xwidget.NewTappableLabel("Test", nil)
	label.SetToolTip("Hello")
	assert.Equal(t, "Hello", label.ToolTip())
}

func TestTappableLabelWithClipboardCopy(t *testing.T) {
	a := test.NewTempApp(t)
	label := xwidget.NewTappableLabelWithClipboardCopy("42")
	w := test.NewWindow(label)
	defer w.Close()

	test.Tap(label)

	assert.Equal(t, "42", a.Clipboard().Content())
	assert.Equal(t, "Click to copy to clipboard", label.ToolTip())
}

var _ fyne.Disableable = (*xwidget.TappableLabel)(nil)
