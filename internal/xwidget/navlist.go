package xwidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
)

// NavList is a widget that renders a list of selectable items.
type NavList struct {
	widget.BaseWidget

	items []*NavListItem
}

func NewNavList(items ...*NavListItem) *NavList {
	w := &NavList{
		items: items,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *NavList) CreateRenderer() fyne.WidgetRenderer {
	var items []fyne.CanvasObject
	for _, it := range w.items {
		items = append(items, it)
	}
	c := container.NewVScroll(container.NewVBox(items...))
	return widget.NewSimpleRenderer(c)
}

type NavListItem struct {
	widget.BaseWidget

	Headline   string
	IsDisabled bool
	Leading    fyne.Resource
	OnTapped   func()
	Supporting string
	Trailing   fyne.Resource

	background      *canvas.Rectangle
	tapBG           *canvas.Rectangle
	headline        *canvas.Text
	leading         *canvas.Image
	leadingWrapped  *fyne.Container
	supporting      *canvas.Text
	trailing        *canvas.Image
	trailingWrapped *fyne.Container
	tapAnim         *fyne.Animation
	isAnimating     bool
}

func NewNavListItem(headline string, leading fyne.Resource, action func()) *NavListItem {
	return newNavListItem(leading, nil, headline, "", action)
}

const (
	navListItemBackgroundColor = theme.ColorNameInputBackground
	navListItemDisabledColor   = theme.ColorNameDisabled
	navListItemTextColor       = theme.ColorNameForeground
)

func newNavListItem(leading, trailing fyne.Resource, headline, supporting string, action func()) *NavListItem {
	if leading == nil {
		leading = iconBlankSvg
	}
	if trailing == nil {
		trailing = iconBlankSvg
	}

	h := canvas.NewText(headline, theme.Color(navListItemTextColor))
	h.TextSize = theme.Size(theme.SizeNameText)
	h.TextStyle.Bold = true

	t := canvas.NewText(supporting, theme.Color(navListItemTextColor))
	t.TextSize = theme.Size(theme.SizeNameText)

	p := theme.Padding()
	background := canvas.NewRectangle(theme.Color(navListItemBackgroundColor))
	background.CornerRadius = 10
	background.SetMinSize(fyne.NewSize(1, 14*p))

	w := &NavListItem{
		background: background,
		headline:   h,
		Headline:   headline,
		leading:    NewImageFromResource(leading, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon))),
		Leading:    leading,
		OnTapped:   action,
		Supporting: supporting,
		supporting: t,
		trailing:   NewImageFromResource(trailing, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon))),
		Trailing:   trailing,
		tapBG:      canvas.NewRectangle(color.Transparent),
	}
	w.ExtendBaseWidget(w)

	w.tapAnim = newButtonTapAnimation(w.tapBG, w, w.Theme())
	w.tapAnim.Curve = fyne.AnimationEaseOut

	w.leadingWrapped = container.NewCenter(container.New(
		layout.NewCustomPaddedLayout(0, 0, p, 2*p),
		w.leading,
	))
	w.trailingWrapped = container.NewCenter(container.New(
		layout.NewCustomPaddedLayout(0, 0, p, p),
		w.trailing,
	))
	w.updateVisibility(leading, trailing, supporting)
	return w
}

func (w *NavListItem) updateVisibility(leading fyne.Resource, trailing fyne.Resource, supporting string) {
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

func (w *NavListItem) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	w.leading.Resource = w.Leading
	w.trailing.Resource = w.Trailing
	w.headline.Text = w.Headline
	w.supporting.Text = w.Supporting
	w.updateVisibility(w.Leading, w.Trailing, w.Supporting)

	if w.IsDisabled {
		c := th.Color(navListItemDisabledColor, v)
		w.headline.Color = c
		w.supporting.Color = c
		if w.leading != nil {
			w.leading.Resource = theme.NewDisabledResource(w.leading.Resource)
		}
	} else {
		c := th.Color(navListItemTextColor, v)
		w.headline.Color = c
		w.supporting.Color = c
		if w.leading != nil {
			w.leading.Resource = theme.NewThemedResource(w.leading.Resource)
		}
	}

	w.background.FillColor = th.Color(navListItemBackgroundColor, v)

	w.background.Refresh()
	w.leading.Refresh()
	w.trailing.Refresh()
	w.headline.Refresh()
	w.supporting.Refresh()
	w.BaseWidget.Refresh()
}

func (w *NavListItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		w.leadingWrapped,
		w.trailingWrapped,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(
				layout.NewCustomPaddedVBoxLayout(0),
				w.headline,
				w.supporting,
			),
			layout.NewSpacer(),
		),
	)
	p := theme.Padding()
	c2 := container.NewStack(
		w.background,
		w.tapBG,
		container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 2*p, 2*p), c),
	)
	return widget.NewSimpleRenderer(c2)
}

func (w *NavListItem) Tapped(_ *fyne.PointEvent) {
	w.tapAnim.Stop()
	w.isAnimating = true
	w.tapAnim.Start()

	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func newButtonTapAnimation(bg *canvas.Rectangle, w *NavListItem, th fyne.Theme) *fyne.Animation {
	v := fyne.CurrentApp().Settings().ThemeVariant()
	return fyne.NewAnimation(canvas.DurationStandard, func(done float32) {
		mid := w.Size().Width / 2
		size := mid * done
		bg.Resize(fyne.NewSize(size*2, w.Size().Height))
		bg.Move(fyne.NewPos(mid-size, 0))

		r, g, bb, a := fynetools.ToNRGBA(th.Color(theme.ColorNamePressed, v))
		aa := uint8(a)
		fade := aa - uint8(float32(aa)*done)
		if fade > 0 {
			bg.FillColor = &color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(bb), A: fade}
		} else {
			bg.FillColor = color.Transparent
		}
		canvas.Refresh(bg)
		if done == 1.0 {
			w.isAnimating = false
		}
	})
}
