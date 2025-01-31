package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TODO: Add hover shadow

// IconButton is an icon widget, which runs a function when tapped.
type IconButton struct {
	widget.DisableableWidget

	// The function that is called when the icon is tapped.
	OnTapped func()

	icon    *canvas.Image
	hovered bool
}

var _ fyne.Tappable = (*IconButton)(nil)
var _ desktop.Hoverable = (*IconButton)(nil)

// NewIconButton returns a new instance of a [IconButton] widget.
func NewIconButton(icon fyne.Resource, tapped func()) *IconButton {
	i := canvas.NewImageFromResource(icon)
	i.FillMode = canvas.ImageFillContain
	i.SetMinSize(fyne.NewSquareSize(sizeIcon))
	w := &IconButton{
		OnTapped: tapped,
		icon:     i,
	}
	w.ExtendBaseWidget(w)
	return w
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
	return widget.NewSimpleRenderer(w.icon)
}
