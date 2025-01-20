package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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
				widget.NewIcon(ui.IconBlankSvg),
				widget.NewLabel("Template"),
				layout.NewSpacer(),
				widget.NewIcon(ui.IconChevronRightSvg),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			item := w.items[id]
			hbox := co.(*fyne.Container).Objects
			title := hbox[1]
			title.(*widget.Label).SetText(item.title)
			icon := hbox[0]
			if item.icon != nil {
				icon.(*widget.Icon).SetResource(item.icon)
				icon.Show()
			} else {
				icon.Hide()
			}
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
