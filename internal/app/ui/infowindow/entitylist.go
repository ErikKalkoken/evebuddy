package infowindow

import (
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
	ID       int64
	Category string
	Text     string
	Variant  InfoVariant
}

func NewEntityItemFromEveEntity(ee *app.EveEntity, text string) EntityItem {
	if text == "" {
		text = ee.Name
	}
	return EntityItem{
		ID:       int64(ee.ID),
		Category: ee.CategoryDisplay(),
		Text:     text,
		Variant:  eveEntity2InfoVariant(ee),
	}
}

// EntitiyList is a list widget for showing entities.
type EntitiyList struct {
	widget.BaseWidget

	items    []EntityItem
	showInfo func(InfoVariant, int64)
}

func NewEntityListFromEntities(show func(InfoVariant, int64), s ...*app.EveEntity) *EntitiyList {
	items := slices.Collect(xiter.MapSlice(s, func(ee *app.EveEntity) EntityItem {
		return NewEntityItemFromEveEntity(ee, "")
	}))
	return NewEntityListFromItems(show, items...)
}

func NewEntityList(show func(InfoVariant, int64)) *EntitiyList {
	items := make([]EntityItem, 0)
	return NewEntityListFromItems(show, items...)
}

func NewEntityListFromItems(show func(InfoVariant, int64), items ...EntityItem) *EntitiyList {
	w := &EntitiyList{
		items:    items,
		showInfo: show,
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
			category.SetText(it.Category)
			if it.Variant == Unknown {
				icon.Hide()
			} else {
				icon.Show()
			}
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
		if it.Variant == Unknown {
			return
		}
		w.showInfo(it.Variant, it.ID)
	}
	return widget.NewSimpleRenderer(l)
}
