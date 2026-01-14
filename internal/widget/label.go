package widget

import "fyne.io/fyne/v2/widget"

// NewLabelWithSelection returns a new label with selectable text.
func NewLabelWithSelection(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Selectable = true
	return l
}
