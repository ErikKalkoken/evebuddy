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

	body      fyne.CanvasObject
	navigator *Navigator
	title     string
}

func NewAppBar(title string, body fyne.CanvasObject, n *Navigator) *AppBar {
	w := &AppBar{
		body:      body,
		navigator: n,
		title:     title,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *AppBar) CreateRenderer() fyne.WidgetRenderer {
	title := widget.NewLabel(w.title)
	title.TextStyle.Bold = true
	row := container.NewStack(container.NewHBox(layout.NewSpacer(), title, layout.NewSpacer()))
	if w.navigator != nil {
		row.Add(container.NewHBox(kxwidget.NewTappableIcon(
			theme.NewThemedResource(ui.IconArrowLeftSvg), func() {
				w.navigator.Pop()
			})))
	}
	top := container.NewVBox(row, widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, w.body)
	return widget.NewSimpleRenderer(c)
}
