package infowindow

import (
	"net/url"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type EntityItem struct {
	Entity   *app.EveEntity
	Text     string
	Category string // when set will be used instead of category from EveEntity
}

func NewEntityItem(ee *app.EveEntity, text string) EntityItem {
	if text == "" {
		text = ee.Name
	}
	return EntityItem{Entity: ee, Text: text}
}

type EntitiyList struct {
	widget.BaseWidget

	ShowEveEntity func(*app.EveEntity)

	items   []EntityItem
	openURL func(*url.URL) error
}

func NewEntityListFromEntities(s ...*app.EveEntity) *EntitiyList {
	items := slices.Collect(xiter.MapSlice(s, func(ee *app.EveEntity) EntityItem {
		return NewEntityItem(ee, "")
	}))
	return NewEntityListFromItems(items...)
}

func NewEntityList() *EntitiyList {
	items := make([]EntityItem, 0)
	return NewEntityListFromItems(items...)
}

func NewEntityListFromItems(items ...EntityItem) *EntitiyList {
	w := &EntitiyList{
		items:   items,
		openURL: fyne.CurrentApp().OpenURL,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *EntitiyList) Set(items ...EntityItem) {
	w.items = items
	w.Refresh()
}

func (w *EntitiyList) CreateRenderer() fyne.WidgetRenderer {
	l := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			category := iwidget.NewLabelWithSize("Category", theme.SizeNameCaptionText)
			text := widget.NewRichText()
			text.Truncation = fyne.TextTruncateEllipsis
			icon := widget.NewIcon(theme.InfoIcon())
			p := theme.Padding()
			return container.NewBorder(
				nil,
				nil,
				nil,
				container.NewVBox(layout.NewSpacer(), icon, layout.NewSpacer()),
				container.New(
					layout.NewCustomPaddedVBoxLayout(0),
					container.New(layout.NewCustomPaddedLayout(0, -1.5*p, 0, 0), category),
					container.New(layout.NewCustomPaddedLayout(-1.5*p, 0, 0, 0), text),
				))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			it := w.items[id]
			border1 := co.(*fyne.Container).Objects
			border2 := border1[0].(*fyne.Container).Objects
			icon := border1[1]
			category := border2[0].(*fyne.Container).Objects[0].(*iwidget.Label)
			var s string
			if it.Category != "" {
				s = it.Category
			} else if it.Entity != nil {
				s = it.Entity.CategoryDisplay()
			}
			if it.Entity != nil && slices.Contains(SupportedCategories(), it.Entity.Category) {
				icon.Show()
			} else {
				icon.Hide()
			}
			category.SetText(s)
			text := border2[1].(*fyne.Container).Objects[0].(*widget.RichText)
			text.ParseMarkdown(it.Text)
			text.Refresh()
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(w.items) {
			return
		}
		it := w.items[id]
		if it.Entity != nil && slices.Contains(SupportedCategories(), it.Entity.Category) && w.ShowEveEntity != nil {
			w.ShowEveEntity(it.Entity)
		}
	}
	return widget.NewSimpleRenderer(l)
}
