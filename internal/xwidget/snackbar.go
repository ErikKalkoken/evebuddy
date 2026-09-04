package xwidget

import (
	"context"
	"log/slog"
	"sync"
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
	snackbarColorNameBackground = theme.ColorNameForeground
	snackbarColorNameForeground = theme.ColorNameBackground
	snackbarMarginBottom        = 7  // multiples of standard padding
	snackbarMarginSides         = 10 // multiples of standard padding
	snackbarTimeoutDefault      = 3 * time.Second
)

type snackbarMessage struct {
	text    string        // text of the message
	timeout time.Duration // Duration the snackbar is shown before it disappears on its own
}

// A Snackbar shows short updates about app processes at the bottom of the window.
// and disappear on their own after a short while.
// Snackbars can also be dismissed by clicking anywhere on the screen.
//
// Snackbars are designed to be created once for each window and then re-used.
//
// A snackbar can be stopped and started.
// Texts received while a snackbar is not started will be queued.
type Snackbar struct {
	BottomMargin float32 // additional bottom padding makes a snackbar appear higher

	bg           *canvas.Rectangle
	isRunning    atomic.Bool
	itemCancel   func()
	mu           sync.Mutex
	parentCancel func()
	popup        *popUp2
	q            *syncqueue.SyncQueue[snackbarMessage]
	text         *RichText
}

// NewSnackbar returns a new snackbar. Call Start() to activate it.
func NewSnackbar(c fyne.Canvas) *Snackbar {
	sb := &Snackbar{
		bg:   canvas.NewRectangle(theme.Color(snackbarColorNameBackground)),
		q:    syncqueue.New[snackbarMessage](),
		text: NewRichText(),
	}
	p := theme.Padding()
	content := container.NewStack(
		sb.bg,
		container.New(
			layout.NewCustomPaddedLayout(0, 0, p, p),
			sb.text,
		),
	)
	sb.popup = newPopUp2(content, c, func() {
		sb.hide()
	})
	sb.popup.Hide()
	// TODO: Once on Fyne >= 2.9, set sb.popup.OnDismiss = sb.hide instead of/in
	// addition to the popUp2 tapped callback above. Currently, tapping outside the
	// snackbar dismisses it via widget.PopUp's own Hide() without notifying sb.hide(),
	// so itemCtx isn't canceled and showMessage keeps blocking until its timeout,
	// stalling any queued messages. OnDismiss fixes this since it's invoked from
	// Hide() itself. See https://github.com/fyne-io/fyne/issues/6468.
	return sb
}

// Show displays a SnackBar with a message and the default timeout.
// Show can be used concurrently.
// When a snackbar receives several texts at the same time,
// it will queue them and display them one after the other.
func (sb *Snackbar) Show(text string) {
	sb.q.Put(snackbarMessage{text: text, timeout: snackbarTimeoutDefault})
}

// ShowWithTimeout is similar to Show but uses a custom timeout.
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
	parentCtx, parentCancel := context.WithCancel(context.Background())
	sb.mu.Lock()
	sb.parentCancel = parentCancel
	sb.mu.Unlock()
	go func() {
		defer func() {
			sb.isRunning.Store(false)
			parentCancel()
			slog.Debug("Snackbar stopped")
		}()
		for {
			m, err := sb.q.Get(parentCtx)
			if err != nil {
				break
			}
			abort := sb.showMessage(parentCtx, m)
			if abort {
				return
			}
		}
	}()
	slog.Debug("Snackbar started")
}

func (sb *Snackbar) showMessage(parentCtx context.Context, m snackbarMessage) bool {
	itemCtx, itemCancel := context.WithCancel(parentCtx)
	sb.mu.Lock()
	sb.itemCancel = itemCancel
	sb.mu.Unlock()
	fyne.Do(func() {
		sb.show(m.text)
	})
	timer := time.NewTimer(m.timeout)
	defer func() {
		timer.Stop()
		itemCancel()
		fyne.Do(func() {
			sb.popup.Hide()
		})
	}()
	select {
	case <-parentCtx.Done():
		return true
	case <-timer.C:
	case <-itemCtx.Done():
	}
	return false
}

func (sb *Snackbar) hide() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.itemCancel != nil {
		sb.itemCancel()
	}
}

// Stop stops a running snackbar and allows the gc to clean up its resources.
// A stopped snackbar can be restarted.
func (sb *Snackbar) Stop() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.parentCancel != nil {
		sb.parentCancel()
	}
}

func (sb *Snackbar) show(text string) {
	_, canvasSize := sb.popup.Canvas.InteractiveArea()
	p := theme.Padding()
	padding := snackbarMarginSides * p
	maxW := canvasSize.Width - padding

	// 1. Assign text and determine if wrapping is needed
	sb.text.SetWithText(text, widget.RichTextStyle{ColorName: snackbarColorNameForeground})

	measurer := canvas.NewText(text, theme.Color(snackbarColorNameForeground))
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

type popUp2 struct {
	widget.PopUp
	tapped func()
}

func newPopUp2(content fyne.CanvasObject, canvas fyne.Canvas, tapped func()) *popUp2 {
	w := &popUp2{
		PopUp: widget.PopUp{
			Content: content,
			Canvas:  canvas,
		},
		tapped: tapped,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *popUp2) Tapped(pe *fyne.PointEvent) {
	if w.tapped != nil {
		w.tapped()
	}
	w.PopUp.Tapped(pe)
}
