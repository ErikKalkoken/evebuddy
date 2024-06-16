package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// TappableIcon is an icon widget, which runs a function when tapped.
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
