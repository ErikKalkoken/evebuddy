package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// An AppBar displays navigation, actions, and text at the top of a screen.
//
// AppBars can be used for both mobile and desktop UIs.
type AppBar struct {
	widget.BaseWidget

	Navigator *Navigator

	bg       *canvas.Rectangle
	body     fyne.CanvasObject
	isMobile bool
	items    []*IconButton
	title    *Label
}

// NewAppBar returns a new AppBar. The toolbar items are optional.
func NewAppBar(title string, body fyne.CanvasObject, items ...*IconButton) *AppBar {
	w := &AppBar{
		body:     body,
		isMobile: fyne.CurrentDevice().IsMobile(),
		items:    items,
	}
	w.ExtendBaseWidget(w)
	w.bg = canvas.NewRectangle(theme.Color(colorBarBackground))
	w.bg.SetMinSize(fyne.NewSize(10, 45))
	if !w.isMobile {
		w.bg.Hide()
	}
	var size fyne.ThemeSizeName
	if w.isMobile {
		size = theme.SizeNameSubHeadingText
	} else {
		size = theme.SizeNameText
	}
	w.title = NewLabelWithSize(title, size)
	w.title.TextStyle.Bold = true
	w.title.Truncation = fyne.TextTruncateEllipsis
	return w
}

func (w *AppBar) Title() string {
	return w.title.Text
}

func (w *AppBar) Refresh() {
	if w.isMobile {
		th := w.Theme()
		v := fyne.CurrentApp().Settings().ThemeVariant()
		w.bg.FillColor = th.Color(colorBarBackground, v)
		w.bg.Refresh()
	}
	w.title.Refresh()
	w.body.Refresh()
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
	var top fyne.CanvasObject
	if w.isMobile {
		top = container.New(
			layout.NewCustomPaddedLayout(-p, -2*p, -p, -p),
			container.NewStack(w.bg, container.NewPadded(row)),
		)
	} else {
		top = container.NewVBox(
			row,
			widget.NewSeparator(),
		)
	}
	c := container.NewBorder(top, nil, nil, nil, container.New(layout.NewCustomPaddedLayout(p, p, 0, 0), w.body))
	return widget.NewSimpleRenderer(c)
}
