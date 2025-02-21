package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ListItem struct {
	Action     func()
	Headline   string
	Leading    fyne.Resource
	Supporting string
	Trailing   fyne.Resource
}

func NewListItemWithIcon(headline string, leading fyne.Resource, action func()) *ListItem {
	return &ListItem{
		Action:   action,
		Leading:  leading,
		Headline: headline,
	}
}

func NewListItem(headline string, action func()) *ListItem {
	return &ListItem{
		Action:   action,
		Headline: headline,
	}
}

func NewListItemWithNavigator(nav *Navigator, ab *AppBar) *ListItem {
	w := NewListItem(ab.Title(), func() {
		nav.Push(ab)
	})
	return w
}

type List struct {
	widget.BaseWidget

	title string
	items []*ListItem
}

type listItem struct {
	widget.BaseWidget

	headline        *canvas.Text
	supporting      *canvas.Text
	leading         *canvas.Image
	leadingWrapped  *fyne.Container
	trailingWrapped *fyne.Container
	trailing        *canvas.Image
}

func newListItem(leading, trailing fyne.Resource, headline, supporting string) *listItem {
	if leading == nil {
		leading = iconBlankSvg
	}
	i1 := NewImageFromResource(leading, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon)))
	if trailing == nil {
		trailing = iconBlankSvg
	}
	i2 := NewImageFromResource(trailing, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon)))
	h := canvas.NewText(headline, theme.Color(theme.ColorNameForeground))
	h.TextSize = theme.Size(theme.SizeNameSubHeadingText)
	t := canvas.NewText(supporting, theme.Color(theme.ColorNameForeground))
	t.TextSize = theme.Size(theme.SizeNameText)
	w := &listItem{
		headline:   h,
		supporting: t,
		leading:    i1,
		trailing:   i2,
	}
	p := theme.Padding()
	w.leadingWrapped = container.NewCenter(container.New(layout.NewCustomPaddedLayout(0, 0, p, p), w.leading))
	w.trailingWrapped = container.NewCenter(container.New(layout.NewCustomPaddedLayout(0, 0, p, p), w.trailing))
	w.ExtendBaseWidget(w)
	w.updateVisibility(leading, trailing, supporting)
	return w
}

func (w *listItem) Set(leading, trailing fyne.Resource, headline, supporting string) {
	w.leading.Resource = leading
	w.trailing.Resource = trailing
	w.headline.Text = headline
	w.supporting.Text = supporting
	w.updateVisibility(leading, trailing, supporting)
	w.Refresh()
}

func (w *listItem) updateVisibility(leading fyne.Resource, trailing fyne.Resource, supporting string) {
	if supporting == "" {
		w.supporting.Hide()
	} else {
		w.supporting.Show()
	}
	if leading == nil {
		w.leadingWrapped.Hide()
	} else {
		w.leadingWrapped.Show()
	}
	if trailing == nil {
		w.trailingWrapped.Hide()
	} else {
		w.trailingWrapped.Show()
	}
}

func (w *listItem) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.headline.Color = th.Color(theme.ColorNameForeground, v)
	w.supporting.Color = th.Color(theme.ColorNameForeground, v)
	w.leading.Refresh()
	w.trailing.Refresh()
	w.headline.Refresh()
	w.supporting.Refresh()
	w.BaseWidget.Refresh()
}

func (w *listItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewPadded(container.NewBorder(
		nil,
		nil,
		w.leadingWrapped,
		w.trailingWrapped,
		container.NewPadded(
			container.NewVBox(
				layout.NewSpacer(),
				container.New(
					layout.NewCustomPaddedVBoxLayout(0),
					w.headline,
					w.supporting,
				),
				layout.NewSpacer(),
			)),
	))
	return widget.NewSimpleRenderer(c)
}
