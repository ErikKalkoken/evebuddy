package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// An AppBar displays navigation, actions, and text at the top of a screen
type AppBar struct {
	widget.BaseWidget

	Navigator *Navigator

	body  fyne.CanvasObject
	bg    *canvas.Rectangle
	title *Label
	items []*IconButton
}

// NewAppBar returns a new AppBar. The toolbar items are optional.
func NewAppBar(title string, body fyne.CanvasObject, items ...*IconButton) *AppBar {
	bg := canvas.NewRectangle(theme.Color(colorBarBackground))
	bg.SetMinSize(fyne.NewSize(10, 45))
	t := NewLabelWithSize(title, theme.SizeNameSubHeadingText)
	t.TextStyle.Bold = true
	t.Truncation = fyne.TextTruncateEllipsis
	w := &AppBar{
		body:  body,
		items: items,
		bg:    bg,
		title: t,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *AppBar) Title() string {
	return w.title.Text
}

func (w *AppBar) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(colorBarBackground, v)
	w.bg.Refresh()
	w.BaseWidget.Refresh()
}

func (w *AppBar) CreateRenderer() fyne.WidgetRenderer {
	var left, right fyne.CanvasObject
	if w.Navigator != nil {
		left = NewIconButton(theme.NavigateBackIcon(), func() {
			w.Navigator.Pop()
		})
	}
	p := theme.Padding()
	is := theme.IconInlineSize()
	if len(w.items) > 0 {
		icons := container.New(layout.NewCustomPaddedHBoxLayout(is))
		for _, ib := range w.items {
			icons.Add(ib)
		}
		right = container.New(layout.NewCustomPaddedLayout(0, 0, 0, p), icons)
	}
	row := container.NewBorder(nil, nil, left, right, w.title)
	top := container.New(
		layout.NewCustomPaddedLayout(-p, -2*p, -p, -p),
		container.NewStack(w.bg, container.NewPadded(row)),
	)
	c := container.NewBorder(top, nil, nil, nil, container.New(layout.NewCustomPaddedLayout(p, p, 0, 0), w.body))
	return widget.NewSimpleRenderer(c)
}
