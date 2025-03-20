package tools

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// MakeTopLabel returns a new empty label meant for the top bar on a screen.
func MakeTopLabel() *widget.Label {
	l := widget.NewLabel("")
	l.TextStyle.Bold = true
	l.Wrapping = fyne.TextWrapWord
	return l
}
