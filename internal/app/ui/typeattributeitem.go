package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The typeAttributeItem widget is used to render items on the type info window.
type typeAttributeItem struct {
	widget.BaseWidget
	icon  *widget.Icon
	label *widget.Label
	value *widget.Label
}

func newTypeAttributeItem() *typeAttributeItem {
	w := &typeAttributeItem{
		icon:  widget.NewIcon(theme.QuestionIcon()),
		label: widget.NewLabel(""),
		value: widget.NewLabel(""),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *typeAttributeItem) SetRegular(icon fyne.Resource, label, value string) {
	w.label.TextStyle.Bold = false
	w.label.Importance = widget.MediumImportance
	w.label.Text = label
	w.label.Refresh()
	w.icon.SetResource(icon)
	w.icon.Show()
	w.value.SetText(value)
	w.value.Show()
}

func (w *typeAttributeItem) SetTitle(label string) {
	w.label.TextStyle.Bold = true
	w.label.Text = label
	w.label.Refresh()
	w.icon.Hide()
	w.value.Hide()
}

func (w *typeAttributeItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHBox(w.icon, w.label, layout.NewSpacer(), w.value)
	return widget.NewSimpleRenderer(c)
}
