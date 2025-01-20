package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// An AppBar displays navigation, actions, and text at the top of a screen
type AppBar struct {
	widget.BaseWidget

	Navigator *Navigator

	body  fyne.CanvasObject
	title string
	items []widget.ToolbarItem
}

// NewAppBar returns a new AppBar. The toolbar items are optional.
func NewAppBar(title string, body fyne.CanvasObject, items ...widget.ToolbarItem) *AppBar {
	w := &AppBar{
		body:  body,
		title: title,
		items: items,
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
			theme.NavigateBackIcon(), func() {
				w.Navigator.Pop()
			})))
	}
	if len(w.items) > 0 {
		row.Add(container.NewHBox(layout.NewSpacer(), widget.NewToolbar(w.items...)))
	}
	top := container.NewVBox(row, widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, w.body)
	return widget.NewSimpleRenderer(c)
}
