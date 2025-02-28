package widget

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/syncqueue"
)

const (
	shadowWidth            = 8 // from Fyne source
	snackbarTimeoutDefault = 3 * time.Second
)

type snackbarMessage struct {
	text    string        // text of the message
	timeout time.Duration // Duration the snackbar is shown before it disappears on it's own
}

// Snackbars show short updates about app processes at the bottom of the screen
// and disapear on their own after a short while. Or after the user clicks on the window to dismiss them.
//
// Snackbars are designed to be created once for each window and then re-used. The can be used concurrently.
//
// When a snackbar receives several texts at the same time, it will queue them and display them one after the other.
type Snackbar struct {
	isRunning atomic.Bool
	q         *syncqueue.SyncQueue[snackbarMessage]
	popup     *widget.PopUp
	stopC     chan struct{}
}

// NewSnackbar returns a new snackbar. Call Start() to activate it.
func NewSnackbar(win fyne.Window) *Snackbar {
	sb := &Snackbar{
		popup: widget.NewPopUp(widget.NewLabel(""), win.Canvas()),
		q:     syncqueue.New[snackbarMessage](),
	}
	return sb
}

// Show displays a SnackBar with a messsage and the the default timeout.
func (w *Snackbar) Show(text string) {
	w.q.Put(snackbarMessage{text: text, timeout: snackbarTimeoutDefault})
}

// Show displays a SnackBar with a messsage and the a custom timeout.
func (w *Snackbar) ShowWithTimeout(text string, timeout time.Duration) {
	w.q.Put(snackbarMessage{text: text, timeout: timeout})
}

// Start starts the SnackBar so it can display messages.
// Start should be called after the Fyne app is started.
func (w *Snackbar) Start() {
	isRunning := !w.isRunning.CompareAndSwap(false, true)
	if isRunning {
		slog.Warn("Snackbar already running")
		return
	}
	go func() {
		for {
			m, _ := w.q.Get(context.Background())
			w.update(m.text)
			w.popup.Show()
			time.Sleep(m.timeout)
			w.popup.Hide()
		}
	}()
	slog.Debug("Snackbar started")
}

func (w *Snackbar) update(text string) {
	w.popup.Content.(*widget.Label).SetText(text)
	_, canvasSize := w.popup.Canvas.InteractiveArea()
	outerSize := w.popup.Content.MinSize().Add(fyne.NewSquareSize(
		theme.Size(theme.SizeNameInnerPadding) + shadowWidth,
	))
	w.popup.Move(fyne.NewPos(
		canvasSize.Width/2-(outerSize.Width)/2,
		canvasSize.Height-outerSize.Height-0.2*outerSize.Height,
	))
}
