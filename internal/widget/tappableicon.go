package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// TappableIcon is an icon widget, which runs a function when tapped and supports tooltips.
type TappableIcon struct {
	widget.Icon
	ttwidget.ToolTipWidgetExtend

	// The function that is called when the icon is tapped.
	OnTapped func()

	hovered bool
}

var _ fyne.Tappable = (*TappableIcon)(nil)
var _ desktop.Hoverable = (*TappableIcon)(nil)

// NewTappableIcon returns a new instance of a [TappableIcon] widget.
func NewTappableIcon(res fyne.Resource, tapped func()) *TappableIcon {
	w := &TappableIcon{OnTapped: tapped}
	w.ExtendBaseWidget(w)
	w.SetResource(res)
	return w
}

func (w *TappableIcon) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.Icon.ExtendBaseWidget(wid)
}
func (w *TappableIcon) Tapped(_ *fyne.PointEvent) {
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *TappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (w *TappableIcon) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *TappableIcon) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseIn(e)
	w.hovered = true
}

func (w *TappableIcon) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseMoved(e)
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *TappableIcon) MouseOut() {
	w.ToolTipWidgetExtend.MouseOut()
	w.hovered = false
}
