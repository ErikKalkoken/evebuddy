package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// IconButton represents an icon button.
//
// This is a workaround until kxwidget.IconButton has been modified to be extensible with tooltips.
type IconButton struct {
	widget.BaseWidget
	Icon *TappableIcon
}

func NewIconButton(res fyne.Resource, tapped func()) *IconButton {
	w := &IconButton{
		Icon: NewTappableIcon(res, tapped),
	}
	w.ExtendBaseWidget(w)
	return w
}

func NewIconButtonWithMenu(res fyne.Resource, menu *fyne.Menu) *IconButton {
	w := &IconButton{
		Icon: NewTappableIconWithMenu(res, menu),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *IconButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewPadded(w.Icon))
}

func (w *IconButton) SetToolTip(toolTip string) {
	w.Icon.SetToolTip(toolTip)
}
