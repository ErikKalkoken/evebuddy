package widget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// StrippedList is List with stripped rows.
type StrippedList struct {
	widget.List
	bgColor color.Color

	BackgroundColorName fyne.ThemeColorName
}

func NewStrippedList(length func() int, createItem func() fyne.CanvasObject, updateItem func(widget.ListItemID, fyne.CanvasObject)) *StrippedList {
	w := &StrippedList{
		BackgroundColorName: theme.ColorNameInputBackground,
	}
	w.ExtendBaseWidget(w)
	w.HideSeparators = true
	w.Length = length
	w.CreateItem = func() fyne.CanvasObject {
		return container.NewStack(
			canvas.NewRectangle(color.Transparent),
			createItem(),
		)
	}
	w.UpdateItem = func(id widget.ListItemID, co fyne.CanvasObject) {
		x := co.(*fyne.Container).Objects
		bg := x[0].(*canvas.Rectangle)
		if id%2 == 0 {
			bg.FillColor = w.bgColor
		} else {
			bg.FillColor = color.Transparent
		}
		bg.Refresh()
		updateItem(id, x[1])
	}
	w.applyTheme()
	return w
}

func (w *StrippedList) Refresh() {
	w.applyTheme()
	w.List.Refresh()
}

func (w *StrippedList) applyTheme() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bgColor = th.Color(w.BackgroundColorName, v)
}
