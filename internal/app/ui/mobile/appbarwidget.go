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
	bg := canvas.NewRectangle(theme.Color(colorAppBarBackground))
	bg.SetMinSize(fyne.NewSize(10, 45))
	w := &AppBar{
		body:  body,
		items: items,
		bg:    bg,
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
	row := container.NewStack()
	if w.title != nil {
		row.Add(container.NewHBox(layout.NewSpacer(), w.title, layout.NewSpacer()))
	}
	if w.Navigator != nil {
		row.Add(container.NewHBox(kxwidget.NewTappableIcon(theme.NavigateBackIcon(), func() {
			w.Navigator.Pop()
		})))
	}
	if len(w.items) > 0 {
		row.Add(container.NewHBox(layout.NewSpacer(), widget.NewToolbar(w.items...)))
	}
	p := theme.Padding()
	top := container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), container.NewStack(w.bg, row))
	c := container.NewBorder(top, nil, nil, nil, w.body)
	return widget.NewSimpleRenderer(c)
}
