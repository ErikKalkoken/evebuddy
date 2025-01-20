package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type navListItem struct {
	action func()
	icon   fyne.Resource
	title  string
}

func NewNavListItem(icon fyne.Resource, title string, action func()) navListItem {
	return navListItem{
		action: action,
		icon:   icon,
		title:  title,
	}
}

type NavList struct {
	widget.BaseWidget

	items []navListItem
}

func NewNavList(items ...navListItem) *NavList {
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
			return container.NewHBox(
				widget.NewIcon(theme.BrokenImageIcon()),
				widget.NewLabel("Template"),
				layout.NewSpacer(),
				widget.NewIcon(theme.NewThemedResource(ui.IconChevronRightSvg)),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			item := w.items[id]
			hbox := co.(*fyne.Container).Objects
			hbox[0].(*widget.Icon).SetResource(item.icon)
			hbox[1].(*widget.Label).SetText(item.title)
		},
	)
	c.OnSelected = func(id widget.ListItemID) {
		defer c.UnselectAll()
		if a := w.items[id].action; a != nil {
			a()
		}
	}
	return widget.NewSimpleRenderer(c)
}
