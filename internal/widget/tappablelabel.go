package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// TappableLabel is a variant of the Fyne Label which runs a function when tapped.
// It also supports tooltips.
type TappableLabel struct {
	widget.Label
	ttwidget.ToolTipWidgetExtend

	// The function that is called when the label is tapped.
	OnTapped func()

	hovered bool
}

var _ fyne.Tappable = (*TappableLabel)(nil)
var _ desktop.Hoverable = (*TappableLabel)(nil)

// NewTappableLabel returns a new TappableLabel instance.
func NewTappableLabel(text string, tapped func()) *TappableLabel {
	w := &TappableLabel{OnTapped: tapped}
	w.ExtendBaseWidget(w)
	w.SetText(text)
	return w
}

func (w *TappableLabel) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.Label.ExtendBaseWidget(wid)
}

func (w *TappableLabel) Tapped(_ *fyne.PointEvent) {
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *TappableLabel) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *TappableLabel) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseIn(e)
	if w.OnTapped != nil {
		w.hovered = true
	}
}

func (w *TappableLabel) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseMoved(e)
}

func (w *TappableLabel) MouseOut() {
	w.ToolTipWidgetExtend.MouseOut()
	w.hovered = false
}
