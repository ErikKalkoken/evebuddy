package xwidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

// NewSpacer returns an invisible object of fixed size.
func NewSpacer(s fyne.Size) fyne.CanvasObject {
	w := canvas.NewRectangle(color.Transparent)
	w.SetMinSize(s)
	return w
}

// NewStandardSpacer returns an invisible object of fixed size
// which its width and height equal to the theme's padding.
func NewStandardSpacer() fyne.CanvasObject {
	return NewSpacer(fyne.NewSquareSize(theme.Padding()))
}
