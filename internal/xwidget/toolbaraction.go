package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// NewToolbarActionMenu returns a ToolBarAction with a context menu.
func NewToolbarActionMenu(icon fyne.Resource, menu *fyne.Menu) *widget.ToolbarAction {
	a := widget.NewToolbarAction(icon, nil)
	SetToolbarActionMenu(a, menu)
	return a
}

// SetToolbarActionMenu replaces the menu of a toolbar action.
func SetToolbarActionMenu(a *widget.ToolbarAction, menu *fyne.Menu) {
	o := a.ToolbarObject()
	a.OnActivated = func() {
		c := fyne.CurrentApp().Driver().CanvasForObject(o)
		m := widget.NewPopUpMenu(menu, c)
		m.ShowAtRelativePosition(fyne.Position{}, o)
	}
}
