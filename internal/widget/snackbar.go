package widget

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
	button    *IconButton
	isRunning atomic.Bool
	label     *widget.Label
	popup     *widget.PopUp
	q         *syncqueue.SyncQueue[snackbarMessage]
	hideC     chan struct{}
}

// NewSnackbar returns a new snackbar. Call Start() to activate it.
func NewSnackbar(win fyne.Window) *Snackbar {
	sb := &Snackbar{
		label: widget.NewLabel(""),
		q:     syncqueue.New[snackbarMessage](),
		hideC: make(chan struct{}),
	}
	sb.button = NewIconButton(theme.WindowCloseIcon(), func() {
		sb.hideC <- struct{}{}
	})
	sb.popup = widget.NewPopUp(container.NewBorder(nil, nil, nil, sb.button, sb.label), win.Canvas())
	return sb
}

// Show displays a SnackBar with a messsage and the the default timeout.
func (sb *Snackbar) Show(text string) {
	sb.q.Put(snackbarMessage{text: text, timeout: snackbarTimeoutDefault})
}

// Show displays a SnackBar with a messsage and the a custom timeout.
func (sb *Snackbar) ShowWithTimeout(text string, timeout time.Duration) {
	sb.q.Put(snackbarMessage{text: text, timeout: timeout})
}

// Start starts the SnackBar so it can display messages.
// Start should be called after the Fyne app is started.
func (sb *Snackbar) Start() {
	isRunning := !sb.isRunning.CompareAndSwap(false, true)
	if isRunning {
		slog.Warn("Snackbar already running")
		return
	}
	go func() {
		for {
			m, _ := sb.q.Get(context.Background())
			sb.show(m.text)
			select {
			case <-sb.hideC:
			case <-time.After(m.timeout):
			}
			sb.popup.Hide()
		}
	}()
	slog.Debug("Snackbar started")
}

func (sb *Snackbar) show(text string) {
	sb.label.SetText(text)
	_, canvasSize := sb.popup.Canvas.InteractiveArea()
	padding := theme.Padding()
	bWidth := sb.button.MinSize().Width + 2*padding + shadowWidth
	maxw := canvasSize.Width - bWidth
	lSize := widget.NewLabel(text).MinSize()
	var cSize fyne.Size
	if lSize.Width > maxw {
		h := lSize.Height * lSize.Width / maxw
		cSize = fyne.NewSize(maxw, h)
		sb.label.Wrapping = fyne.TextWrapWord
	} else {
		cSize = lSize
		sb.label.Wrapping = fyne.TextWrapOff
	}
	sb.popup.Resize(cSize)
	sb.popup.Move(fyne.NewPos(
		canvasSize.Width/2-(cSize.Width+shadowWidth)/2,
		canvasSize.Height-1.4*(cSize.Height+shadowWidth),
	))
	sb.popup.Show()
}
