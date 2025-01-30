package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type NavListItem struct {
	Action     func()
	Icon       fyne.Resource
	Headline   string
	Supporting string
}

func NewNavListItemWithIcon(icon fyne.Resource, headline string, action func()) *NavListItem {
	return &NavListItem{
		Action:   action,
		Icon:     icon,
		Headline: headline,
	}
}

func NewNavListItem(headline string, action func()) *NavListItem {
	return &NavListItem{
		Action:   action,
		Headline: headline,
	}
}

func NewNavListItemWithNavigator(nav *Navigator, ab *AppBar) *NavListItem {
	return NewNavListItem(ab.Title(), func() {
		nav.Push(ab)
	})
}

type NavList struct {
	widget.BaseWidget

	title string
	items []*NavListItem
}

type navListItem struct {
	widget.BaseWidget

	headline   *canvas.Text
	supporting *canvas.Text
	icon       *canvas.Image
	indicator  *canvas.Image
}

func newNavListItem(icon fyne.Resource, headline, supporting string) *navListItem {
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
	w := &navListItem{
		headline:   h,
		supporting: t,
		icon:       i1,
		indicator:  i2,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *navListItem) Set(icon fyne.Resource, headline, supporting string) {
	w.icon.Resource = icon
	w.headline.Text = headline
	w.supporting.Text = supporting
	w.Refresh()
}

func (w *navListItem) Refresh() {
	w.icon.Refresh()
	w.headline.Refresh()
	if w.supporting.Text == "" {
		w.supporting.Hide()
	} else {
		w.supporting.Show()
	}
	w.supporting.Refresh()
	w.BaseWidget.Refresh()
}

func (w *navListItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.New(layout.NewCustomPaddedLayout(p, p, 2*p, 2*p), w.icon),
		container.New(layout.NewCustomPaddedLayout(p, p, 2*p, 2*p), w.indicator),
		container.NewPadded(
			container.New(
				layout.NewCustomPaddedVBoxLayout(0),
				w.headline,
				w.supporting,
			)),
	)
	return widget.NewSimpleRenderer(c)
}
