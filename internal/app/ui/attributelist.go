package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// AttributeItem is a row in an [AttributeList].
type AttributeItem struct {
	Label      string
	Value      string
	Importance widget.Importance
	InfoAction func() // shows an info icon when set
	Action     func() // called when the row is tapped
}

// AttributeList is a widget that shows a list of labeled values.
type AttributeList struct {
	widget.BaseWidget

	items []AttributeItem
}

func NewAttributeList(items ...AttributeItem) *AttributeList {
	w := &AttributeList{items: items}
	w.ExtendBaseWidget(w)
	return w
}

// Set replaces all items.
func (w *AttributeList) Set(items []AttributeItem) {
	w.items = items
	w.Refresh()
}

func (w *AttributeList) CreateRenderer() fyne.WidgetRenderer {
	l := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			value := widget.NewLabel("Value")
			value.Truncation = fyne.TextTruncateEllipsis
			value.Alignment = fyne.TextAlignTrailing
			label := widget.NewLabel("Label")
			icon := xwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
			return container.NewBorder(
				nil,
				nil,
				label,
				container.NewVBox(layout.NewSpacer(), icon, layout.NewSpacer()),
				value,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			it := w.items[id]
			border := co.(*fyne.Container).Objects

			border[1].(*widget.Label).SetText(it.Label)

			value := border[0].(*widget.Label)
			value.Text = it.Value
			value.Importance = it.Importance
			value.Refresh()

			iconBox := border[2].(*fyne.Container)
			if it.InfoAction != nil {
				iconBox.Objects[1].(*xwidget.TappableIcon).OnTapped = it.InfoAction
				iconBox.Show()
			} else {
				iconBox.Hide()
			}
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(w.items) {
			return
		}
		if f := w.items[id].Action; f != nil {
			f()
		}
	}
	return widget.NewSimpleRenderer(l)
}
