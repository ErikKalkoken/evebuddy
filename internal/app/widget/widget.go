// Package widget contains generic widgets with app dependencies.
package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// Default ScaleMode for images
var DefaultImageScaleMode canvas.ImageScale

// MakeTopLabel returns a new empty label meant for the top bar on a screen.
func MakeTopLabel() *widget.Label {
	l := widget.NewLabel("")
	l.Wrapping = fyne.TextWrapWord
	return l
}
