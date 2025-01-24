package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type NavListItem struct {
	Action func()
	Icon   fyne.Resource
	Title  string
	Suffix string
}

func NewNavListItemWithIcon(icon fyne.Resource, title string, action func()) *NavListItem {
	return &NavListItem{
		Action: action,
		Icon:   icon,
		Title:  title,
	}
}

func NewNavListItem(title string, action func()) *NavListItem {
	return &NavListItem{
		Action: action,
		Title:  title,
	}
}

func NewNavListItemWithNavigator(nav *Navigator, ab *AppBar) *NavListItem {
	return NewNavListItem(ab.title, func() {
		nav.Push(ab)
	})
}

type NavList struct {
	widget.BaseWidget

	title string
	items []*NavListItem
}

func NewNavList(items ...*NavListItem) *NavList {
	return NewNavListWithTitle("", items...)
}

func NewNavListWithTitle(title string, items ...*NavListItem) *NavList {
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
				widget.NewLabel("Title"),
				layout.NewSpacer(),
				widget.NewLabel("Suffix"),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			item := w.items[id]
			hbox := co.(*fyne.Container).Objects

			title := hbox[1].(*widget.Label)
			title.SetText(item.Title)

			icon := hbox[0].(*widget.Icon)
			if item.Icon != nil {
				icon.SetResource(item.Icon)
				icon.Show()
			} else {
				icon.Hide()
			}

			suffix := hbox[3].(*widget.Label)
			if item.Suffix == "" {
				suffix.Hide()
			} else {
				suffix.SetText(item.Suffix)
				suffix.Show()
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if a := w.items[id].Action; a != nil {
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
