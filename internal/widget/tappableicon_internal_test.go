package widget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestTappableIcon_CanHover(t *testing.T) {
	if fyne.CurrentDevice().IsMobile() {
		t.Skip("skipped on mobile")
	}
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	icon := NewTappableIcon(theme.HomeIcon(), nil)
	w := test.NewWindow(icon)
	defer w.Close()

	test.MoveMouse(w.Canvas(), fyne.NewPos(5, 5))
	icon.hovered = true

	test.MoveMouse(w.Canvas(), fyne.NewPos(0, 0))
	icon.hovered = false
}
