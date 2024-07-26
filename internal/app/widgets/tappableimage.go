package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TappableImage is an icon widget, which runs a custom function when tapped.
type TappableImage struct {
	widget.BaseWidget
	image *canvas.Image

	// The function that is called when the label is tapped.
	OnTapped func()

	hovered bool
}

var _ fyne.Tappable = (*TappableImage)(nil)
var _ desktop.Hoverable = (*TappableImage)(nil)

func NewTappableImage(res fyne.Resource, fillMode canvas.ImageFill, tapped func()) *TappableImage {
	ti := &TappableImage{OnTapped: tapped, image: canvas.NewImageFromResource(res)}
	ti.image.FillMode = fillMode
	ti.ExtendBaseWidget(ti)
	return ti
}

func (ti *TappableImage) SetMinSize(size fyne.Size) {
	ti.image.SetMinSize(size)
}

func (ti *TappableImage) Tapped(_ *fyne.PointEvent) {
	ti.OnTapped()
}

func (ti *TappableImage) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (ti *TappableImage) Cursor() desktop.Cursor {
	if ti.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (ti *TappableImage) MouseIn(e *desktop.MouseEvent) {
	ti.hovered = true
}

func (ti *TappableImage) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (ti *TappableImage) MouseOut() {
	ti.hovered = false
}

func (ti *TappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ti.image)
}
