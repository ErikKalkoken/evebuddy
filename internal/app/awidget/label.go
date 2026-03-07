package awidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func NewLabelWithTruncation(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

func NewLabelWithWrapping(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}
