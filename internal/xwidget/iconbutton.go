package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// IconButton is a [kxwidget.IconButton] with tooltip support.
type IconButton struct {
	kxwidget.IconButton
	ttwidget.ToolTipWidgetExtend
}

var _ desktop.Hoverable = (*IconButton)(nil)

func NewIconButton(res fyne.Resource, tapped func()) *IconButton {
	w := &IconButton{}
	w.ExtendBaseWidget(w)
	w.SetIcon(res)
	w.OnTapped = tapped
	return w
}

func NewIconButtonWithMenu(res fyne.Resource, menu *fyne.Menu) *IconButton {
	w := NewIconButton(res, nil)
	if menu == nil {
		fyne.LogError("IconButton misconfigured: missing menu", nil)
		return w
	}
	w.SetMenuItems(menu.Items)
	return w
}

func (w *IconButton) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.IconButton.ExtendBaseWidget(wid)
}

func (w *IconButton) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseIn(e)
	w.IconButton.MouseIn(e)
}

func (w *IconButton) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseMoved(e)
	w.IconButton.MouseMoved(e)
}

func (w *IconButton) MouseOut() {
	w.ToolTipWidgetExtend.MouseOut()
	w.IconButton.MouseOut()
}
