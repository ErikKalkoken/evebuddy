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

type Snackbar struct {
	widget.PopUp

	Delay time.Duration
}

func NewSnackbar(text string, win fyne.Window) *Snackbar {
	w := &Snackbar{Delay: snackbarDelayDefault}
	w.ExtendBaseWidget(w)
	w.Content = widget.NewLabel(text)
	c := win.Canvas()
	w.Canvas = c
	_, canvasSize := c.InteractiveArea()
	outerSize := w.Content.MinSize().Add(fyne.NewSquareSize(theme.Size(theme.SizeNameInnerPadding) + shadowWidth))
	w.Move(fyne.NewPos(canvasSize.Width/2-(outerSize.Width)/2, canvasSize.Height-outerSize.Height-0.2*outerSize.Height))
	return w
}

func (w *Snackbar) Show() {
	w.PopUp.Show()
	go func() {
		time.Sleep(w.Delay)
		w.Hide()
	}()

}
