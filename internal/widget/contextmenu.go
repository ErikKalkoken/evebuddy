package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// ContextMenuButton is a button that shows a context menu.
// The button can have a label and a leading icon.
// And it supports tooltips.
type ContextMenuButton struct {
	widget.Button
	ttwidget.ToolTipWidgetExtend

	menu *fyne.Menu
}

func NewContextMenuButton(label string, menu *fyne.Menu) *ContextMenuButton {
	return NewContextMenuButtonWithIcon(label, nil, menu)
}

func NewContextMenuButtonWithIcon(label string, icon fyne.Resource, menu *fyne.Menu) *ContextMenuButton {
	w := &ContextMenuButton{menu: menu}
	w.ExtendBaseWidget(w)
	w.SetText(label)
	if icon != nil {
		w.SetIcon(icon)
	}
	return w
}

func (w *ContextMenuButton) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.Button.ExtendBaseWidget(wid)
}

func (w *ContextMenuButton) Tapped(e *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(w.menu, fyne.CurrentApp().Driver().CanvasForObject(w), e.AbsolutePosition)
}

// SetMenuItems replaces the menu items.
func (w *ContextMenuButton) SetMenuItems(menuItems []*fyne.MenuItem) {
	w.menu.Items = menuItems
	w.menu.Refresh()
}

func (w *ContextMenuButton) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseIn(e)
	w.Button.MouseIn(e)
}

func (w *ContextMenuButton) MouseOut() {
	w.ToolTipWidgetExtend.MouseOut()
	w.Button.MouseOut()
}

func (w *ContextMenuButton) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseMoved(e)
	w.Button.MouseMoved(e)
}

func ShowContextMenu(o fyne.CanvasObject, menu *fyne.Menu) {
	m := widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(o))
	m.ShowAtRelativePosition(fyne.NewPos(
		-m.Size().Width+o.Size().Width,
		o.Size().Height,
	), o)
}
