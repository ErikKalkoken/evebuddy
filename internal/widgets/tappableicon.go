package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// ToDo: Add background highlight for mouse hover to indicate to user that this icon can be tapped

// TappableIcon is an icon widget, which executed a function when tapped.
type TappableIcon struct {
	widget.Icon

	OnTapped func()
}

func NewTappableIcon(res fyne.Resource, tapped func()) *TappableIcon {
	icon := &TappableIcon{OnTapped: tapped}
	icon.ExtendBaseWidget(icon)
	icon.SetResource(res)
	return icon
}

func (t *TappableIcon) Tapped(_ *fyne.PointEvent) {
	t.OnTapped()
}

func (t *TappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}
