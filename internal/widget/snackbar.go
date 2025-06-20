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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

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

// A Snackbar shows short updates about app processes at the bottom of the screen
// and disappear on their own after a short while. Or after the user clicks on the window to dismiss them.
//
// Snackbars are designed to be created once for each window and then re-used. The can be used concurrently.
//
// When a snackbar receives several texts at the same time, it will queue them and display them one after the other.
type Snackbar struct {
	button    *kxwidget.IconButton
	hideC     chan struct{}
	isRunning atomic.Bool
	label     *widget.Label
	popup     *widget.PopUp
	q         *syncqueue.SyncQueue[snackbarMessage]
	stopC     chan struct{}
}

// NewSnackbar returns a new snackbar. Call Start() to activate it.
func NewSnackbar(win fyne.Window) *Snackbar {
	sb := &Snackbar{
		hideC: make(chan struct{}),
		label: widget.NewLabel(""),
		q:     syncqueue.New[snackbarMessage](),
		stopC: make(chan struct{}),
	}
	sb.button = kxwidget.NewIconButton(theme.WindowCloseIcon(), func() {
		sb.hideC <- struct{}{}
	})
	sb.popup = widget.NewPopUp(container.NewBorder(nil, nil, nil, sb.button, sb.label), win.Canvas())
	return sb
}

// Show displays a SnackBar with a message and the the default timeout.
func (sb *Snackbar) Show(text string) {
	sb.q.Put(snackbarMessage{text: text, timeout: snackbarTimeoutDefault})
}

// ShowWithTimeout displays a SnackBar with a message and the a custom timeout.
func (sb *Snackbar) ShowWithTimeout(text string, timeout time.Duration) {
	sb.q.Put(snackbarMessage{text: text, timeout: timeout})
}

// Start starts the SnackBar so it can display messages.
// Start should be called after the Fyne app is started.
func (sb *Snackbar) Start() {
	isRunning := !sb.isRunning.CompareAndSwap(false, true)
	if isRunning {
		slog.Warn("Snackbar has already been started")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sb.stopC
		cancel()
	}()
	go func() {
	L:
		for {
			m, err := sb.q.Get(ctx)
			if err != nil {
				break
			}
			fyne.Do(func() {
				sb.show(m.text)
			})
			select {
			case <-sb.hideC:
			case <-sb.stopC:
				sb.popup.Hide()
				cancel()
				break L
			case <-time.After(m.timeout):
			}
			fyne.Do(func() {
				sb.popup.Hide()
			})
		}
		sb.isRunning.Store(false)
		slog.Debug("Snackbar stopped")
	}()
	slog.Debug("Snackbar started")
}

// Stop stops a running snackbar and allows the gc to clean up it's resources.
func (sb *Snackbar) Stop() {
	if !sb.isRunning.Load() {
		return
	}
	sb.stopC <- struct{}{}
}

func (sb *Snackbar) IsRunning() bool {
	return sb.isRunning.Load()
}

func (sb *Snackbar) show(text string) {
	sb.label.SetText(text)
	_, canvasSize := sb.popup.Canvas.InteractiveArea()
	padding := theme.Padding()
	bWidth := sb.button.MinSize().Width + 2*padding + shadowWidth
	maxW := canvasSize.Width - bWidth
	lSize := widget.NewLabel(text).MinSize()
	var cSize fyne.Size
	if lSize.Width > maxW {
		h := lSize.Height * lSize.Width / maxW
		cSize = fyne.NewSize(maxW, h)
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
