package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// TappableIcon is a [kxwidget.TappableIcon] with tooltip support.
type TappableIcon struct {
	kxwidget.TappableIcon
	ttwidget.ToolTipWidgetExtend
}

var _ desktop.Hoverable = (*TappableIcon)(nil)

// NewTappableIcon returns a new instance of a [TappableIcon] widget.
func NewTappableIcon(res fyne.Resource, tapped func()) *TappableIcon {
	w := &TappableIcon{}
	w.ExtendBaseWidget(w)
	w.SetResource(res)
	w.OnTapped = tapped
	return w
}

func (w *TappableIcon) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.TappableIcon.ExtendBaseWidget(wid)
}

func (w *TappableIcon) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseIn(e)
	w.TappableIcon.MouseIn(e)
}

func (w *TappableIcon) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseMoved(e)
	w.TappableIcon.MouseMoved(e)
}

func (w *TappableIcon) MouseOut() {
	w.ToolTipWidgetExtend.MouseOut()
	w.TappableIcon.MouseOut()
}
