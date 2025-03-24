package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TappableRichText is a variant of the RichText widget which runs a function when tapped.
type TappableRichText struct {
	widget.RichText

	// The function that is called when the label is tapped.
	OnTapped func()

	hovered bool
}

var _ fyne.Tappable = (*TappableRichText)(nil)
var _ desktop.Hoverable = (*TappableRichText)(nil)

func NewTappableRichTextWithText(text string, tapped func()) *TappableRichText {
	w := NewTappableRichText(tapped, NewRichTextSegmentFromText(text)...)
	return w
}

// NewTappableRichText returns a new TappableRichText instance.
func NewTappableRichText(tapped func(), segments ...widget.RichTextSegment) *TappableRichText {
	w := &TappableRichText{OnTapped: tapped}
	w.ExtendBaseWidget(w)
	w.Segments = segments
	w.Scroll = container.ScrollNone
	return w
}

func (w *TappableRichText) Tapped(_ *fyne.PointEvent) {
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

// Cursor returns the cursor type of this widget
func (w *TappableRichText) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *TappableRichText) MouseIn(e *desktop.MouseEvent) {
	w.hovered = true
}

func (w *TappableRichText) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *TappableRichText) MouseOut() {
	w.hovered = false
}
