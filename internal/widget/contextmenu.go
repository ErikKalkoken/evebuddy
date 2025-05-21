package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type ContextMenuButton struct {
	widget.Button
	menu *fyne.Menu
}

// NewContextMenuButton is a button that shows a context menu.
func NewContextMenuButton(label string, menu *fyne.Menu) *ContextMenuButton {
	return NewContextMenuButtonWithIcon(nil, label, menu)
}

// NewContextMenuButtonWithIcon is an icon button that shows a context menu. The label is optional.
func NewContextMenuButtonWithIcon(icon fyne.Resource, label string, menu *fyne.Menu) *ContextMenuButton {
	b := &ContextMenuButton{menu: menu}
	b.ExtendBaseWidget(b)
	b.Text = label
	if icon != nil {
		b.Icon = icon
	}
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

func ShowContextMenu(o fyne.CanvasObject, menu *fyne.Menu) {
	m := widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(o))
	m.ShowAtRelativePosition(fyne.NewPos(
		-m.Size().Width+o.Size().Width,
		o.Size().Height,
	), o)
}
