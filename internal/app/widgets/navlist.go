package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

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
			return newNavListItem(iconBlankSvg, "", "")
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			item := w.items[id]
			w := co.(*navListItem)
			w.Set(item.Icon, item.Headline, item.Supporting)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if a := w.items[id].Action; a != nil {
			a()
		}
	}
	l := widget.NewLabel(w.title)
	l.TextStyle.Bold = true
	if w.title == "" {
		l.Hide()
	}
	c := container.NewBorder(l, nil, nil, nil, list)
	return widget.NewSimpleRenderer(c)
}
