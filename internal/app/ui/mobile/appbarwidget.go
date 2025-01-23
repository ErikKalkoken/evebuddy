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
	top := container.NewVBox()
	title := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{
			ColorName: theme.ColorNameForeground,
			Inline:    false,
			SizeName:  theme.SizeNameSubHeadingText,
		},
		Text: w.title,
	})
	if w.Navigator == nil {
		row := container.NewHBox(title, layout.NewSpacer())
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
		top.Add(title)
	}
	top.Add(widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, w.body)
	return widget.NewSimpleRenderer(c)
}
