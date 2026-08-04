package mailer

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type entityItem struct {
	widget.BaseWidget

	category *widget.Label
	icon     *canvas.Image
	name     *widget.Label
	loadIcon ui.EveEntityIconLoader
}

func newEntityItem(loadIcon ui.EveEntityIconLoader) *entityItem {
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateClip
	category := widget.NewLabel("")
	category.SizeName = theme.SizeNameCaptionText
	icon := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(ui.IconUnitSize))
	w := &entityItem{
		category: category,
		loadIcon: loadIcon,
		icon:     icon,
		name:     name,
	}
	w.ExtendBaseWidget(w)

	return w
}

func (w *entityItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(layout.NewCustomPaddedLayout(p, p, p, -p), w.icon),
			layout.NewSpacer(),
		),
		w.category,
		container.NewVBox(
			layout.NewSpacer(),
			w.name,
			layout.NewSpacer(),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *entityItem) set(o *app.EveEntity) {
	w.name.SetText(o.Name)
	w.category.SetText(o.CategoryDisplay())
	w.loadIcon(o, ui.IconPixelSize, func(r fyne.Resource) {
		w.icon.Resource = r
		w.icon.Refresh()
	})
}
