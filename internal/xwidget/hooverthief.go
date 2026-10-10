package xwidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// HooverThief is a widget that can disable hoovering of lower widgets.
// For example when put on top a list item, the list item no longer hoovers.
type HooverThief struct {
	widget.BaseWidget

	hovered bool
}

var _ desktop.Hoverable = (*HooverThief)(nil)

func NewHooverThief() *HooverThief {
	w := &HooverThief{}
	w.ExtendBaseWidget(w)
	return w
}

func (w *HooverThief) CreateRenderer() fyne.WidgetRenderer {
	r := canvas.NewRectangle(color.Transparent)
	return widget.NewSimpleRenderer(r)
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *HooverThief) MouseIn(_ *desktop.MouseEvent) {
	w.hovered = true
}

func (w *HooverThief) MouseMoved(_ *desktop.MouseEvent) {}

func (w *HooverThief) MouseOut() {
	w.hovered = false
}
