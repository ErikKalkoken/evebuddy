package widgets

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
	Icon       fyne.Resource
	Headline   string
	Supporting string
}

func NewListItemWithIcon(icon fyne.Resource, headline string, action func()) *ListItem {
	return &ListItem{
		Action:   action,
		Icon:     icon,
		Headline: headline,
	}
}

func NewListItemWithIconAndText(icon fyne.Resource, headline, supporting string, action func()) *ListItem {
	return &ListItem{
		Action:     action,
		Icon:       icon,
		Headline:   headline,
		Supporting: supporting,
	}
}

func NewListItem(headline string, action func()) *ListItem {
	return &ListItem{
		Action:   action,
		Headline: headline,
	}
}

func NewListItemWithNavigator(nav *Navigator, ab *AppBar) *ListItem {
	return NewListItem(ab.Title(), func() {
		nav.Push(ab)
	})
}

type List struct {
	widget.BaseWidget

	title string
	items []*ListItem
}

type listItem struct {
	widget.BaseWidget

	headline    *canvas.Text
	supporting  *canvas.Text
	icon        *canvas.Image
	iconWrapped *fyne.Container
	indicator   *canvas.Image
}

func newListItem(icon fyne.Resource, headline, supporting string) *listItem {
	i1 := canvas.NewImageFromResource(icon)
	i1.FillMode = canvas.ImageFillContain
	i1.SetMinSize(fyne.NewSquareSize(sizeIcon))
	i2 := canvas.NewImageFromResource(theme.NewThemedResource(iconChevronRightSvg))
	i2.FillMode = canvas.ImageFillContain
	i2.SetMinSize(fyne.NewSquareSize(sizeIcon))
	h := canvas.NewText(headline, theme.Color(theme.ColorNameForeground))
	h.TextSize = 16
	t := canvas.NewText(supporting, theme.Color(theme.ColorNameForeground))
	t.TextSize = 14
	w := &listItem{
		headline:   h,
		supporting: t,
		icon:       i1,
		indicator:  i2,
	}
	p := theme.Padding()
	w.iconWrapped = container.NewCenter(container.New(layout.NewCustomPaddedLayout(0, 0, p, p), w.icon))
	w.ExtendBaseWidget(w)
	return w
}

func (w *listItem) Set(icon fyne.Resource, headline, supporting string) {
	w.icon.Resource = icon
	w.headline.Text = headline
	w.supporting.Text = supporting
	if supporting == "" {
		w.supporting.Hide()
	} else {
		w.supporting.Show()
	}
	if icon == nil {
		w.iconWrapped.Hide()
	} else {
		w.iconWrapped.Show()
	}
	w.Refresh()
}

func (w *listItem) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.headline.Color = th.Color(theme.ColorNameForeground, v)
	w.supporting.Color = th.Color(theme.ColorNameForeground, v)
	w.icon.Refresh()
	w.headline.Refresh()
	w.supporting.Refresh()
	w.BaseWidget.Refresh()
}

func (w *listItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewPadded(container.NewBorder(
		nil,
		nil,
		w.iconWrapped,
		container.NewCenter(w.indicator),
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
