package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TappableIcon is an icon widget, which runs a function when tapped.
type TappableIcon struct {
	widget.Icon

	// The function that is called when the label is tapped.
	OnTapped func()

	hovered bool
}

var _ fyne.Tappable = (*TappableIcon)(nil)
var _ desktop.Hoverable = (*TappableIcon)(nil)

func NewTappableIcon(res fyne.Resource, tapped func()) *TappableIcon {
	ti := &TappableIcon{OnTapped: tapped}
	ti.ExtendBaseWidget(ti)
	ti.SetResource(res)
	return ti
}

func (ti *TappableIcon) Tapped(_ *fyne.PointEvent) {
	ti.OnTapped()
}

func (ti *TappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (ti *TappableIcon) Cursor() desktop.Cursor {
	if ti.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (ti *TappableIcon) MouseIn(e *desktop.MouseEvent) {
	ti.hovered = true
}

func (ti *TappableIcon) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (ti *TappableIcon) MouseOut() {
	ti.hovered = false
}
