package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// Button is Button widget with tooltips.
type Button = ttwidget.Button

// NewIconButton creates a icon button.
// An icon button is an icon that behaves like a button.
func NewIconButton(res fyne.Resource, tapped func()) *Button {
	w := ttwidget.NewButtonWithIcon("", res, tapped)
	w.Importance = widget.LowImportance
	return w
}
