package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type NavListItem struct {
	Action func()
	Icon   fyne.Resource
	Title  string
}

func NewNavListItem(icon fyne.Resource, title string, action func()) NavListItem {
	return NavListItem{
		Action: action,
		Icon:   icon,
		Title:  title,
	}
}

type NavList struct {
	widget.BaseWidget

	items []NavListItem
}

func NewNavList(items ...NavListItem) *NavList {
	w := &NavList{
		items: items,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *NavList) CreateRenderer() fyne.WidgetRenderer {
	c := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.BrokenImageIcon()), widget.NewLabel("Template"), layout.NewSpacer(), widget.NewLabel(">"))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			item := w.items[id]
			hbox := co.(*fyne.Container).Objects
			hbox[0].(*widget.Icon).SetResource(item.Icon)
			hbox[1].(*widget.Label).SetText(item.Title)
		},
	)
	c.OnSelected = func(id widget.ListItemID) {
		defer c.UnselectAll()
		if a := w.items[id].Action; a != nil {
			a()
		}
	}
	return widget.NewSimpleRenderer(c)
}
