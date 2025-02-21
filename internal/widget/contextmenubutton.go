// Package widgets contains custom widgets for this app.
package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type ContextMenuButton struct {
	widget.Button
	menu *fyne.Menu
}

// NewContextMenuButtonWithIcon is an icon button that shows a context menu. The label is optional.
func NewContextMenuButtonWithIcon(icon fyne.Resource, label string, menu *fyne.Menu) *ContextMenuButton {
	b := &ContextMenuButton{menu: menu}
	b.Text = label
	b.Icon = icon

	b.ExtendBaseWidget(b)
	return b
}

func (b *ContextMenuButton) Tapped(e *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
}

// SetMenuItems replaces the menu items.
func (b *ContextMenuButton) SetMenuItems(menuItems []*fyne.MenuItem) {
	b.menu.Items = menuItems
	b.menu.Refresh()
}
