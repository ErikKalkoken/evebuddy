package widget

import "fyne.io/fyne/v2/widget"

// NewCustomHyperlink returns a new hyperlink with a custom action.
func NewCustomHyperlink(text string, onTapped func()) *widget.Hyperlink {
	x := widget.NewHyperlink(text, nil)
	x.OnTapped = onTapped
	return x
}
