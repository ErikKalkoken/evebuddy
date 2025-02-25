package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TODO: Add hover shadow

// IconButton is an icon widget, which runs a function when tapped.
type IconButton struct {
	widget.DisableableWidget

	// This callback runs when the icon is tapped.
	OnTapped func()

	icon    *canvas.Image
	menu    *fyne.Menu
	hovered bool
}

var _ fyne.Tappable = (*IconButton)(nil)
var _ desktop.Hoverable = (*IconButton)(nil)

// NewIconButton returns a new instance of an [IconButton].
func NewIconButton(icon fyne.Resource, tapped func()) *IconButton {
	i := NewImageFromResource(icon, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon)))
	w := &IconButton{
		OnTapped: tapped,
		icon:     i,
	}
	w.ExtendBaseWidget(w)
	return w
}

// NewIconButtonWithMenu returns an [IconButton] with a context menu.
func NewIconButtonWithMenu(icon fyne.Resource, menu *fyne.Menu) *IconButton {
	w := NewIconButton(icon, nil)
	w.menu = menu
	w.OnTapped = func() {
		if len(w.menu.Items) == 0 {
			return
		}
		ShowContextMenu(w, w.menu)
	}
	return w
}

// SetIcon replaces the current icon.
func (w *IconButton) SetIcon(icon fyne.Resource) {
	w.icon.Resource = icon
	w.icon.Refresh()
}

// SetMenuItems replaces the menu items.
func (w *IconButton) SetMenuItems(menuItems []*fyne.MenuItem) {
	if w.menu == nil {
		return
	}
	w.menu.Items = menuItems
	w.menu.Refresh()
}

func (w *IconButton) Tapped(_ *fyne.PointEvent) {
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *IconButton) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (w *IconButton) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *IconButton) MouseIn(e *desktop.MouseEvent) {
	w.hovered = true
}

func (w *IconButton) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *IconButton) MouseOut() {
	w.hovered = false
}

func (w *IconButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewPadded(w.icon))
}
