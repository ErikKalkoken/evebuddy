package widget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestTappableRichText_CanHover(t *testing.T) {
	if fyne.CurrentDevice().IsMobile() {
		t.Skip("skipped on mobile")
	}
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	icon := NewTappableRichText(RichTextSegmentsFromText("Test"), nil)
	w := test.NewWindow(icon)
	defer w.Close()

	test.MoveMouse(w.Canvas(), fyne.NewPos(5, 5))
	icon.hovered = true

	test.MoveMouse(w.Canvas(), fyne.NewPos(0, 0))
	icon.hovered = false
}
