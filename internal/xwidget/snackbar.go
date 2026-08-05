package xwidget

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/syncqueue"
)

const (
	snackbarTimeoutDefault      = 3 * time.Second
	snackbarColorNameBackground = theme.ColorNameForeground
	snackbarColorNameForeGround = theme.ColorNameBackground
	snackbarMarginSides         = 10 // multiples of standard padding
	snackbarMarginBottom        = 7  // multiples of standard padding
)

type snackbarMessage struct {
	text    string        // text of the message
	timeout time.Duration // Duration the snackbar is shown before it disappears on it's own
}

// FIXME: Horizontal positioning is off for multi-line snackbars

// A Snackbar shows short updates about app processes at the bottom of the window.
// and disappear on their own after a short while.
// Or after the user clicks on the window to dismiss them.
//
// Snackbars are designed to be created once for each window and then re-used.
//
// A snackbar can be used concurrently.
// When a snackbar receives several texts at the same time,
// it will queue them and display them one after the other.
type Snackbar struct {
	BottomMargin float32 // additional bottom padding makes a snackbar appear higher

	bg        *canvas.Rectangle
	hideC     chan struct{}
	isRunning atomic.Bool
	popup     *widget.PopUp
	q         *syncqueue.SyncQueue[snackbarMessage]
	stopC     chan struct{}
	text      *RichText
}

// NewSnackbar returns a new snackbar. Call Start() to activate it.
func NewSnackbar(c fyne.Canvas) *Snackbar {
	sb := &Snackbar{
		bg:    canvas.NewRectangle(theme.Color(snackbarColorNameBackground)),
		hideC: make(chan struct{}),
		q:     syncqueue.New[snackbarMessage](),
		stopC: make(chan struct{}),
		text:  NewRichText(),
	}
	p := theme.Padding()
	content := container.NewStack(
		sb.bg,
		container.New(
			layout.NewCustomPaddedLayout(0, 0, p, p),
			sb.text,
		),
	)
	sb.popup = widget.NewPopUp(content, c)
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
				fyne.Do(func() {
					sb.popup.Hide()
				})
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
	_, canvasSize := sb.popup.Canvas.InteractiveArea()
	p := theme.Padding()
	padding := snackbarMarginSides * p
	maxW := canvasSize.Width - padding

	// 1. Assign text and determine if wrapping is needed
	sb.text.SetWithText(text, widget.RichTextStyle{ColorName: snackbarColorNameForeGround})

	measurer := canvas.NewText(text, theme.Color(snackbarColorNameForeGround))
	measurer.TextSize = theme.TextSize()
	unwrappedSize := measurer.MinSize()

	if unwrappedSize.Width > maxW {
		sb.text.Wrapping = fyne.TextWrapWord
	} else {
		sb.text.Wrapping = fyne.TextWrapOff
	}

	// 2. Set label size explicitly to fixed width so Fyne calculates the true wrapped height
	if unwrappedSize.Width > maxW {
		// Force the label to maxW; Fyne's internal layout recalculates height for wrapped text
		sb.text.Resize(fyne.NewSize(maxW, sb.text.MinSize().Height))
	} else {
		sb.text.Resize(unwrappedSize)
	}

	sb.text.Refresh()

	// 3. Obtain the exact minimum height Fyne requires for this wrapped label
	labelMin := sb.text.MinSize()
	actualWidth := unwrappedSize.Width
	if actualWidth > maxW {
		actualWidth = maxW
	}

	// 4. Set the content size explicitly before querying outer popup dimensions
	contentSize := fyne.NewSize(actualWidth, labelMin.Height)
	sb.popup.Content.Resize(contentSize)

	// 5. Query the outer popup size (includes theme paddings/borders)
	popupSize := sb.popup.MinSize()
	if popupSize.Width < contentSize.Width {
		popupSize.Width = contentSize.Width
	}
	if popupSize.Height < contentSize.Height {
		popupSize.Height = contentSize.Height
	}

	sb.popup.Resize(popupSize)

	// 6. Calculate position anchored to bottom margin
	sb.popup.Move(fyne.NewPos(
		canvasSize.Width/2-popupSize.Width/2,
		canvasSize.Height-popupSize.Height-snackbarMarginBottom*p-sb.BottomMargin,
	))

	// 7. Update background style
	sb.bg.FillColor = theme.Color(snackbarColorNameBackground)
	sb.bg.Refresh()

	sb.popup.Show()
}
