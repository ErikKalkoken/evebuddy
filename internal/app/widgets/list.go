package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func NewNavList(items ...*ListItem) *List {
	return NewNavListWithTitle("", items...)
}

func NewNavListWithTitle(title string, items ...*ListItem) *List {
	w := &List{
		items: items,
		title: title,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *List) CreateRenderer() fyne.WidgetRenderer {
	list := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			return newListItem(iconBlankSvg, "Headline", "Supporting")
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			item := w.items[id]
			w := co.(*listItem)
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
