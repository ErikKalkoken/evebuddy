package mobile

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
	title *widget.RichText
	items []widget.ToolbarItem
}

// NewAppBar returns a new AppBar. The toolbar items are optional.
func NewAppBar(title string, body fyne.CanvasObject, items ...widget.ToolbarItem) *AppBar {
	w := &AppBar{
		body:  body,
		items: items,
		bg:    canvas.NewRectangle(theme.Color(colorAppBarBackground)),
	}
	w.ExtendBaseWidget(w)
	if title != "" {
		w.title = widget.NewRichText(&widget.TextSegment{
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameForeground,
				Inline:    false,
				SizeName:  theme.SizeNameSubHeadingText,
			},
			Text: title,
		})
	}
	return w
}

func (w *AppBar) Title() string {
	if w.title == nil || len(w.title.Segments) == 0 {
		return ""
	}
	return w.title.Segments[0].Textual()
}

func (w *AppBar) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(colorAppBarBackground, v)
	w.bg.Refresh()
	w.BaseWidget.Refresh()
}

func (w *AppBar) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewVBox()
	if w.Navigator == nil {
		row := container.NewHBox()
		if w.title != nil {
			row.Add(w.title)
			row.Add(layout.NewSpacer())
		}
		if len(w.items) > 0 {
			row.Add(container.NewVBox(widget.NewToolbar(w.items...)))
		}
		top.Add(row)
	} else {
		row := container.NewHBox(
			kxwidget.NewTappableIcon(theme.NavigateBackIcon(), func() {
				w.Navigator.Pop()
			}),
			layout.NewSpacer(),
		)
		if len(w.items) > 0 {
			row.Add(container.NewVBox(widget.NewToolbar(w.items...)))
		}
		top.Add(row)
		if w.title != nil {
			top.Add(w.title)
		}
	}
	p := theme.Padding()
	top2 := container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), container.NewStack(w.bg, top))
	c := container.NewBorder(top2, nil, nil, nil, w.body)
	return widget.NewSimpleRenderer(c)
}
