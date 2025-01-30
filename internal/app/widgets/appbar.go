package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

const (
	colorAppBarBackground = theme.ColorNameMenuBackground
)

// An AppBar displays navigation, actions, and text at the top of a screen
type AppBar struct {
	widget.BaseWidget

	Navigator *Navigator

	body  fyne.CanvasObject
	bg    *canvas.Rectangle
	title *SubHeading
	items []widget.ToolbarItem
}

// NewAppBar returns a new AppBar. The toolbar items are optional.
func NewAppBar(title string, body fyne.CanvasObject, items ...widget.ToolbarItem) *AppBar {
	bg := canvas.NewRectangle(theme.Color(colorAppBarBackground))
	bg.SetMinSize(fyne.NewSize(10, 45))
	w := &AppBar{
		body:  body,
		items: items,
		bg:    bg,
	}
	w.ExtendBaseWidget(w)
	if title != "" {
		w.title = NewSubHeading(title)
	}
	return w
}

func (w *AppBar) Title() string {
	if w.title == nil {
		return ""
	}
	return w.title.Text
}

func (w *AppBar) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(colorAppBarBackground, v)
	w.bg.Refresh()
	w.BaseWidget.Refresh()
}

func (w *AppBar) CreateRenderer() fyne.WidgetRenderer {
	row := container.NewStack()
	if w.title != nil {
		row.Add(container.NewHBox(layout.NewSpacer(), w.title, layout.NewSpacer()))
	}
	if w.Navigator != nil {
		icon := kxwidget.NewTappableIcon(theme.NewThemedResource(iconChevronLeftSvg), func() {
			w.Navigator.Pop()
		})
		row.Add(container.NewHBox(icon))
	}
	if len(w.items) > 0 {
		row.Add(container.NewHBox(layout.NewSpacer(), widget.NewToolbar(w.items...)))
	}
	p := theme.Padding()
	top := container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), container.NewStack(w.bg, row))
	c := container.NewBorder(top, nil, nil, nil, w.body)
	return widget.NewSimpleRenderer(c)
}
