package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// NewLabelWithTruncation returns a new label with default truncation.
func NewLabelWithTruncation(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

// NewLabelWithWrapping returns a new label with default wrapping.
func NewLabelWithWrapping(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}
