package widget

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// List is a widget that renders a list of selectable items.
type List struct {
	widget.BaseWidget

	SelectDelay time.Duration

	title string
	items []*ListItem
}

func NewNavList(items ...*ListItem) *List {
	return NewNavListWithTitle("", items...)
}

func NewNavListWithTitle(title string, items ...*ListItem) *List {
	w := &List{
		items:       items,
		SelectDelay: 500 * time.Millisecond,
		title:       title,
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
			return newListItem(iconBlankSvg, iconBlankSvg, "Headline", "Supporting")
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			co.(*listItem).Set(w.items[id])
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(w.items) {
			list.UnselectAll()
			return
		}
		it := w.items[id]
		a := it.Action
		if a == nil || it.IsDisabled {
			list.UnselectAll()
			return
		}
		a()
		go func() {
			time.Sleep(w.SelectDelay)
			list.UnselectAll()
		}()
	}
	list.HideSeparators = true
	l := widget.NewLabel(w.title)
	l.TextStyle.Bold = true
	if w.title == "" {
		l.Hide()
	}
	c := container.NewBorder(l, nil, nil, nil, list)
	return widget.NewSimpleRenderer(c)
}
