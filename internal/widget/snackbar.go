package widget

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	shadowWidth          = 8 // from Fyne source
	snackbarDelayDefault = 3 * time.Second
)

// Snackbars show short updates about app processes at the bottom of the screen.
//
// Snackbars appear on the bottom of the screen and disapear on their own after a short while.
type Snackbar struct {
	widget.PopUp

	// Duration the snackbar is shown before it disappears on it's own
	Timeout time.Duration
}

// NewSnackbar returns a new snackbar. Call Show() to display it.
func NewSnackbar(text string, win fyne.Window) *Snackbar {
	w := &Snackbar{Timeout: snackbarDelayDefault}
	w.ExtendBaseWidget(w)
	w.Content = widget.NewLabel(text)
	c := win.Canvas()
	w.Canvas = c
	_, canvasSize := c.InteractiveArea()
	outerSize := w.Content.MinSize().Add(fyne.NewSquareSize(theme.Size(theme.SizeNameInnerPadding) + shadowWidth))
	w.Move(fyne.NewPos(canvasSize.Width/2-(outerSize.Width)/2, canvasSize.Height-outerSize.Height-0.2*outerSize.Height))
	return w
}

// Show displays the snackbar.
func (w *Snackbar) Show() {
	w.PopUp.Show()
	go func() {
		time.Sleep(w.Timeout)
		w.Hide()
	}()
}

// ShowSnackbar shows a snackbar immediately with default timeout.
func ShowSnackbar(text string, win fyne.Window) {
	sb := NewSnackbar(text, win)
	sb.Show()
}
