package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// NewImageFromResource creates an canvas.Image with defaults.
func NewImageFromResource(res fyne.Resource, minSize fyne.Size) *canvas.Image {
	x := canvas.NewImageFromResource(res)
	x.FillMode = canvas.ImageFillContain
	x.ScaleMode = canvas.ImageScaleFastest
	x.SetMinSize(minSize)
	return x
}
