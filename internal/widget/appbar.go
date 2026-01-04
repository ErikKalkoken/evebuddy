package widget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// An AppBar displays navigation, actions, and text at the top of a screen.
//
// AppBars can be used for both mobile and desktop UIs.
type AppBar struct {
	widget.BaseWidget

	Navigator      *Navigator
	HideBackground bool

	bg       *canvas.Rectangle
	body     fyne.CanvasObject
	trailing *fyne.Container
	title    *widget.Label
}

// NewAppBar returns a new AppBar with a title and a body.
// It can also have one or several trailing widgets.
func NewAppBar(title string, body fyne.CanvasObject, trailing ...fyne.CanvasObject) *AppBar {
	t2 := container.New(layout.NewCustomPaddedHBoxLayout(theme.IconInlineSize()))
	if len(trailing) > 0 {
		for _, x := range trailing {
			t2.Add(x)
		}
	} else {
		t2.Hide()
	}
	w := &AppBar{
		body:     body,
		trailing: t2,
	}
	w.ExtendBaseWidget(w)
	w.bg = canvas.NewRectangle(theme.Color(colorBarBackground))
	w.bg.SetMinSize(fyne.NewSize(10, 45))
	if w.HideBackground {
		w.bg.Hide()
	}
	w.title = widget.NewLabel(title)
	w.title.SizeName = theme.SizeNameSubHeadingText
	w.title.Truncation = fyne.TextTruncateEllipsis
	return w
}

func (w *AppBar) SetTitle(text string) {
	w.title.SetText(text)
}

func (w *AppBar) Title() string {
	return w.title.Text
}

func (w *AppBar) Refresh() {
	if !w.HideBackground {
		th := w.Theme()
		v := fyne.CurrentApp().Settings().ThemeVariant()
		w.bg.FillColor = th.Color(colorBarBackground, v)
		w.bg.Refresh()
	}
	w.title.Refresh()
	w.body.Refresh()
	w.trailing.Refresh()
	w.BaseWidget.Refresh()
}

func (w *AppBar) CreateRenderer() fyne.WidgetRenderer {
	var left, right fyne.CanvasObject
	if w.Navigator != nil {
		left = kxwidget.NewIconButton(theme.NavigateBackIcon(), func() {
			w.Navigator.Pop()
		})
	}
	p := theme.Padding()
	if w.trailing != nil {
		right = container.New(layout.NewCustomPaddedLayout(0, 0, 0, p), w.trailing)
	}
	row := container.NewBorder(nil, nil, left, right, w.title)
	var top, main fyne.CanvasObject
	if !w.HideBackground {
		top = container.New(
			layout.NewCustomPaddedLayout(-p, -2*p, -p, -p),
			container.NewStack(w.bg, container.NewPadded(row)),
		)
		main = container.New(layout.NewCustomPaddedLayout(2*p, p, 0, 0), w.body)
	} else {
		top = container.NewVBox(
			row,
			canvas.NewRectangle(color.Transparent),
		)
		main = w.body
	}
	c := container.NewBorder(top, nil, nil, nil, main)
	return widget.NewSimpleRenderer(c)
}
