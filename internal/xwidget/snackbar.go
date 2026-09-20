package xwidget

import (
	"context"
	"image/color"
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

	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	"github.com/ErikKalkoken/evebuddy/internal/syncqueue"
)

const (
	snackbarColorNameBackground = theme.ColorNameForeground
	snackbarColorNameForeground = theme.ColorNameBackground
	snackbarMarginBottom        = 7  // multiples of standard padding
	snackbarMarginSides         = 10 // multiples of standard padding
	snackbarTimeoutDefault      = 3 * time.Second
	snackbarAnimDuration        = 150 * time.Millisecond // matches Material Design 3's snackbar enter transition
	snackbarAnimStartScale      = 0.8                    // matches Material Design 3's snackbar scale-in factor
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
	showAnim     *fyne.Animation
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
		container.NewClip(
			container.New(
				layout.NewCustomPaddedLayout(0, 0, p, p),
				sb.text,
			),
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

// Display shows a SnackBar with a message and the default timeout.
// Display can be used concurrently.
// When a snackbar receives several texts at the same time,
// it will queue them and display them one after the other.
func (sb *Snackbar) Display(text string) {
	sb.q.Put(snackbarMessage{text: text, timeout: snackbarTimeoutDefault})
}

// DisplayWithTimeout is similar to Display but uses a custom timeout.
func (sb *Snackbar) DisplayWithTimeout(text string, timeout time.Duration) {
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
			if sb.showAnim != nil {
				sb.showAnim.Stop()
			}
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

	wrapped := unwrappedSize.Width > maxW
	if wrapped {
		sb.text.Wrapping = fyne.TextWrapWord
	} else {
		sb.text.Wrapping = fyne.TextWrapOff
	}

	// 2. Set label size explicitly to fixed width so Fyne calculates the true wrapped height.
	// Measure at maxW-2p, the width CustomPaddedLayout will actually render it at below,
	// or the wrap could add a line at final layout that this height doesn't account for.
	if wrapped {
		sb.text.Resize(fyne.NewSize(maxW-2*p, sb.text.MinSize().Height))
	} else {
		sb.text.Resize(unwrappedSize)
	}

	sb.text.Refresh()

	// 3. Obtain the exact minimum size Fyne requires for this label.
	labelMin := sb.text.MinSize()

	// RichText.MinSize() has no meaningful width while wrapped, so use maxW then;
	// otherwise it's more accurate than the plain-text measurer above.
	var actualWidth float32
	if wrapped {
		actualWidth = maxW
	} else {
		actualWidth = labelMin.Width
		if actualWidth > maxW {
			actualWidth = maxW
		}
	}

	// 4. Set the content size explicitly before querying outer popup dimensions.
	// container.Clip (see NewSnackbar) hides the padding popup.MinSize() would
	// otherwise contribute below, so add it back for the single-line case; wrapped
	// text already reserves it within maxW.
	contentWidth := actualWidth
	if !wrapped {
		contentWidth += 2 * p
	}
	contentSize := fyne.NewSize(contentWidth, labelMin.Height)
	sb.popup.Content.Resize(contentSize)

	// 5. Query the outer popup size (includes theme paddings/borders)
	popupSize := sb.popup.MinSize()
	if popupSize.Width < contentSize.Width {
		popupSize.Width = contentSize.Width
	}
	if popupSize.Height < contentSize.Height {
		popupSize.Height = contentSize.Height
	}

	// 6. Calculate the final resting position, anchored to the bottom margin
	finalPos := fyne.NewPos(
		canvasSize.Width/2-popupSize.Width/2,
		canvasSize.Height-popupSize.Height-snackbarMarginBottom*p-sb.BottomMargin,
	)

	// 7. Reveal with a Material Design 3 style scale + fade entrance
	sb.animateShow(popupSize, finalPos)
}

// animateShow plays a Material Design 3 style scale + fade entrance. Fyne has
// no content-scale transform, so scale is approximated by resizing/moving the
// popup around its final rect's center; container.Clip (see NewSnackbar) lets
// it shrink below its natural size and hides the overflow while it does.
func (sb *Snackbar) animateShow(finalSize fyne.Size, finalPos fyne.Position) {
	if sb.showAnim != nil {
		sb.showAnim.Stop()
	}

	center := finalPos.AddXY(finalSize.Width/2, finalSize.Height/2)
	r, g, b, _ := fynetools.ToNRGBA(theme.Color(snackbarColorNameBackground))

	setFrame := func(scale, alpha float32) {
		size := fyne.NewSize(finalSize.Width*scale, finalSize.Height*scale)
		sb.popup.Resize(size)
		sb.popup.Move(center.SubtractXY(size.Width/2, size.Height/2))
		sb.bg.FillColor = &color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(255 * alpha)}
		sb.bg.Refresh()
	}

	setFrame(snackbarAnimStartScale, 0)
	sb.popup.Show()

	sb.showAnim = fyne.NewAnimation(snackbarAnimDuration, func(done float32) {
		scale := snackbarAnimStartScale + (1-snackbarAnimStartScale)*fyne.AnimationEaseOut(done)
		setFrame(scale, done)
	})
	sb.showAnim.Curve = fyne.AnimationLinear
	sb.showAnim.Start()
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
