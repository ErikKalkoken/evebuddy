package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// NewToolbarActionMenu returns a ToolBarAction with a context menu.
func NewToolbarActionMenu(icon fyne.Resource, menu *fyne.Menu) *widget.ToolbarAction {
	a := widget.NewToolbarAction(icon, nil)
	o := a.ToolbarObject()
	a.OnActivated = func() {
		ShowContextMenu(o, menu)
	}
	return a
}

func ShowContextMenu(o fyne.CanvasObject, menu *fyne.Menu) {
	m := widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(o))
	m.ShowAtRelativePosition(fyne.NewPos(
		-m.Size().Width+o.Size().Width,
		o.Size().Height,
	), o)
}
