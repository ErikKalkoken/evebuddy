package awidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// EntityListItem is a list item for an entity. It has an icon and a name.
type EntityListItem struct {
	widget.BaseWidget

	name     *widget.Label
	icon     *canvas.Image
	loadIcon func(id int64, size int, setter func(r fyne.Resource))
}

func NewEntityListItem(isAvatar bool, loadIcon func(id int64, size int, setter func(r fyne.Resource))) *EntityListItem {
	portrait := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
	if isAvatar {
		portrait.CornerRadius = app.IconUnitSize / 2
	}
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateEllipsis
	w := &EntityListItem{
		name:     name,
		icon:     portrait,
		loadIcon: loadIcon,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *EntityListItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(layout.NewCustomPaddedLayout(p, p, 2*p, -p), w.icon),
			layout.NewSpacer(),
		),
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			w.name,
			layout.NewSpacer(),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *EntityListItem) Set(id int64, name string) {
	w.loadIcon(id, app.IconPixelSize, func(r fyne.Resource) {
		w.icon.Resource = r
		w.icon.Refresh()
	})
	w.name.SetText(name)
}
