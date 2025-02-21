package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func ShowContextMenu(o fyne.CanvasObject, menu *fyne.Menu) {
	m := widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(o))
	m.ShowAtRelativePosition(fyne.NewPos(
		-m.Size().Width+o.Size().Width,
		o.Size().Height,
	), o)
}
