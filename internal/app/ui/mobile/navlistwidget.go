package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type navListItem struct {
	action func()
	icon   fyne.Resource
	title  string
}

func NewNavListItemWithIcon(icon fyne.Resource, title string, action func()) navListItem {
	return navListItem{
		action: action,
		icon:   icon,
		title:  title,
	}
}

func NewNavListItem(title string, action func()) navListItem {
	return navListItem{
		action: action,
		title:  title,
	}
}

func NewNavListItemWithNavigator(nav *Navigator, ab *AppBar) navListItem {
	return NewNavListItem(ab.title, func() {
		nav.Push(ab)
	})
}

type NavList struct {
	widget.BaseWidget

	title string
	items []navListItem
}

func NewNavList(items ...navListItem) *NavList {
	return NewNavListWithTitle("", items...)
}

func NewNavListWithTitle(title string, items ...navListItem) *NavList {
	w := &NavList{
		items: items,
		title: title,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *NavList) CreateRenderer() fyne.WidgetRenderer {
	list := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(ui.IconBlankSvg),
				widget.NewLabel("Template"),
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
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if a := w.items[id].action; a != nil {
			a()
		}
	}
	if w.title == "" {
		return widget.NewSimpleRenderer(list)
	}
	l := widget.NewLabel(w.title)
	l.TextStyle.Bold = true
	return widget.NewSimpleRenderer(container.NewBorder(l, nil, nil, nil, list))
}
