package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// TappableImage extends the TappableImage from fyne-kx to add tooltips
type TappableImage struct {
	kxwidget.TappableImage
	ttwidget.ToolTipWidgetExtend
}

func NewTappableImage(res fyne.Resource, tapped func()) *TappableImage {
	w := &TappableImage{}
	w.ExtendBaseWidget(w)
	w.SetResource(res)
	w.OnTapped = tapped
	return w
}

func NewTappableImageWithMenu(res fyne.Resource, menu *fyne.Menu) *TappableImage {
	w := &TappableImage{}
	w.ExtendBaseWidget(w)
	w.SetResource(res)
	w.SetMenuItems(menu.Items)
	return w
}

func (w *TappableImage) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.TappableImage.ExtendBaseWidget(wid)
}

func (w *TappableImage) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseIn(e)
	w.TappableImage.MouseIn(e)
}

func (w *TappableImage) MouseOut() {
	w.ToolTipWidgetExtend.MouseOut()
	w.TappableImage.MouseOut()
}

func (w *TappableImage) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseMoved(e)
	w.TappableImage.MouseMoved(e)
}
