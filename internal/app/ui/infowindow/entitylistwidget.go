package infowindow

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type entityItem struct {
	id           int64
	category     string
	text         string               // text in markdown
	textSegments []widget.TextSegment // takes precendence over text when not empty
	infoVariant  InfoVariant
}

func NewEntityItem(id int64, category, text string, v InfoVariant) entityItem {
	return entityItem{
		id:          id,
		category:    category,
		text:        text,
		infoVariant: v,
	}
}

func NewEntityItemFromEvePlanet(o *app.EvePlanet) entityItem {
	return entityItem{
		id:          int64(o.ID),
		category:    "Planet",
		text:        o.Name,
		infoVariant: NotSupported,
	}
}

func NewEntityItemFromEveSolarSystem(o *app.EveSolarSystem) entityItem {
	ee := o.ToEveEntity()
	return entityItem{
		id:           int64(ee.ID),
		category:     ee.CategoryDisplay(),
		textSegments: o.DisplayRichText(),
		infoVariant:  eveEntity2InfoVariant(ee),
	}
}

func NewEntityItemFromEveEntity(ee *app.EveEntity, text string) entityItem {
	if text == "" {
		text = ee.Name
	}
	return NewEntityItem(int64(ee.ID), ee.CategoryDisplay(), text, eveEntity2InfoVariant(ee))
}

// EntitiyList is a list widget for showing entities.
type EntitiyList struct {
	widget.BaseWidget

	items    []entityItem
	showInfo func(InfoVariant, int64)
}

func NewEntityListFromEntities(show func(InfoVariant, int64), s ...*app.EveEntity) *EntitiyList {
	items := slices.Collect(xiter.MapSlice(s, func(ee *app.EveEntity) entityItem {
		return NewEntityItemFromEveEntity(ee, "")
	}))
	return NewEntityListFromItems(show, items...)
}

func NewEntityList(show func(InfoVariant, int64)) *EntitiyList {
	items := make([]entityItem, 0)
	return NewEntityListFromItems(show, items...)
}

func NewEntityListFromItems(show func(InfoVariant, int64), items ...entityItem) *EntitiyList {
	w := &EntitiyList{
		items:    items,
		showInfo: show,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *EntitiyList) Set(items ...entityItem) {
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
			icon := kxwidget.NewTappableIcon(theme.InfoIcon(), nil)
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
			icon := border1[1].(*fyne.Container).Objects[1].(*kxwidget.TappableIcon)
			category := border2[0].(*fyne.Container).Objects[0].(*iwidget.Label)
			category.SetText(it.category)
			if it.infoVariant == NotSupported {
				icon.OnTapped = nil
				icon.Hide()
			} else {
				icon.OnTapped = func() {
					w.showInfo(it.infoVariant, it.id)
				}
				icon.Show()
			}
			text := border2[1].(*fyne.Container).Objects[0].(*widget.RichText)
			if len(it.textSegments) != 0 {
				iwidget.SetRichText(text, it.textSegments...)
			} else {
				text.ParseMarkdown(it.text)
			}
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()

	}
	return widget.NewSimpleRenderer(l)
}
