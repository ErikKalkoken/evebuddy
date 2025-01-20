package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type AppBar struct {
	widget.BaseWidget

	Navigator *Navigator

	body  fyne.CanvasObject
	title string
}

func NewAppBar(title string, body fyne.CanvasObject) *AppBar {
	w := &AppBar{
		body:  body,
		title: title,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *AppBar) CreateRenderer() fyne.WidgetRenderer {
	title := widget.NewLabel(w.title)
	title.TextStyle.Bold = true
	row := container.NewStack(container.NewHBox(layout.NewSpacer(), title, layout.NewSpacer()))
	if w.Navigator != nil {
		row.Add(container.NewHBox(kxwidget.NewTappableIcon(
			theme.NewThemedResource(ui.IconChevronLeftSvg), func() {
				w.Navigator.Pop()
			})))
	}
	top := container.NewVBox(row, widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, w.body)
	return widget.NewSimpleRenderer(c)
}
