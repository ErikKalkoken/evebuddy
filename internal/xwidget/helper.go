package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// ShowPopUpMenuBelowLeading shows a popup menu below a widget and aligned leading.
func ShowPopUpMenuBelowLeading(w fyne.CanvasObject, m *fyne.Menu) {
	if m == nil {
		return
	}
	pos := fyne.NewPos(0, w.Size().Height)
	widget.ShowPopUpMenuAtRelativePosition(
		m,
		fyne.CurrentApp().Driver().CanvasForObject(w),
		pos,
		w,
	)
}

// ShowPopUpMenuBelowTrailing shows a popup menu below a widget and aligned trailing.
func ShowPopUpMenuBelowTrailing(w fyne.CanvasObject, m *fyne.Menu) {
	if m == nil {
		return
	}
	pum := widget.NewPopUpMenu(m, fyne.CurrentApp().Driver().CanvasForObject(w))
	pum.ShowAtRelativePosition(
		fyne.NewPos(
			-pum.Size().Width+w.Size().Width,
			w.Size().Height,
		),
		w,
	)
}
