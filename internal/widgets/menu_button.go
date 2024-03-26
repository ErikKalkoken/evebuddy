package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type ContextMenuButton struct {
	widget.Button
	menu *fyne.Menu
}

func (b *ContextMenuButton) Tapped(e *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
}

func NewContextMenuButton(label string, menu *fyne.Menu) *ContextMenuButton {
	b := &ContextMenuButton{menu: menu}
	b.Text = label

	b.ExtendBaseWidget(b)
	return b
}
