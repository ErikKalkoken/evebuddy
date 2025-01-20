package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NewMenuToolbarAction returns a ToolBarAction with a context menu.
func NewMenuToolbarAction(items ...*fyne.MenuItem) *widget.ToolbarAction {
	if len(items) == 0 {
		panic("Need to define at least one item")
	}
	a := widget.NewToolbarAction(theme.MoreVerticalIcon(), nil)
	o := a.ToolbarObject()
	a.OnActivated = func() {
		widget.ShowPopUpMenuAtRelativePosition(
			fyne.NewMenu("", items...),
			fyne.CurrentApp().Driver().CanvasForObject(o),
			fyne.Position{},
			o,
		)
	}
	return a
}
