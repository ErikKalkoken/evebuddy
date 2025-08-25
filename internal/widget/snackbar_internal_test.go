package widget

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

const longText = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt. "

func TestSnackbar_Show(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	t.Run("create single line", func(t *testing.T) {
		w := test.NewWindow(widget.NewLabel(""))
		defer w.Close()
		w.Resize(fyne.NewSize(500, 300))
		sb := NewSnackbar(w)
		sb.Start()
		sb.Show("Dummy")
		for {
			time.Sleep(50 * time.Millisecond)
			if sb.q.IsEmpty() {
				break
			}
		}
		test.AssertImageMatches(t, "snackbar/single.png", w.Canvas().Capture())
	})
	t.Run("create with double line text", func(t *testing.T) {
		w := test.NewWindow(widget.NewLabel(""))
		defer w.Close()
		w.Resize(fyne.NewSize(500, 300))
		sb := NewSnackbar(w)
		sb.Start()
		sb.Show(longText)
		for {
			time.Sleep(50 * time.Millisecond)
			if sb.q.IsEmpty() {
				break
			}
		}
		test.AssertImageMatches(t, "snackbar/long.png", w.Canvas().Capture())
	})
}
