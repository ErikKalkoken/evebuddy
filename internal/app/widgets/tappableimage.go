package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// TappableImage is an icon widget, which runs a custom function when tapped.
type TappableImage struct {
	widget.BaseWidget
	image *canvas.Image

	OnTapped func()
}

func NewTappableImage(res fyne.Resource, fillMode canvas.ImageFill, tapped func()) *TappableImage {
	w := &TappableImage{OnTapped: tapped, image: canvas.NewImageFromResource(res)}
	w.image.FillMode = fillMode
	w.ExtendBaseWidget(w)
	return w
}

func (w *TappableImage) SetMinSize(size fyne.Size) {
	w.image.SetMinSize(size)
}

func (w *TappableImage) Tapped(_ *fyne.PointEvent) {
	w.OnTapped()
}

func (w *TappableImage) TappedSecondary(_ *fyne.PointEvent) {
}

func (w *TappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.image)
}
