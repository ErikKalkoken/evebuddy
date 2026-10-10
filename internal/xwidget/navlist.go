package xwidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

var _ fyne.Disableable = (*NavListItem)(nil)
var _ fyne.Tappable = (*NavListItem)(nil)
var _ fyne.Widget = (*NavListItem)(nil)

type NavListItem struct {
	widget.DisableableWidget

	Headline             string
	Leading              fyne.Resource
	OnTapped             func()
	Supporting           string
	SupportingImportance widget.Importance // Medium and Low render muted (default)
	Trailing             fyne.Resource

	backgroundRectangle *canvas.Rectangle
	headlineText        *canvas.Text
	isAnimating         bool
	leadingImage        *canvas.Image
	leadingWrapped      *fyne.Container
	supportingText      *canvas.Text
	tapAnim             *fyne.Animation
	tapBGRectangle      *canvas.Rectangle
	trailingImage       *canvas.Image
	trailingWrapped     *fyne.Container
}

func NewNavListItem(headline string, leading fyne.Resource, action func()) *NavListItem {
	return newNavListItem(leading, nil, headline, "", action)
}

const (
	navListItemBackgroundColor = theme.ColorNameInputBackground
	navListItemDisabledColor   = theme.ColorNameDisabled
	navListItemHeadlineColor   = theme.ColorNameForeground
	navListItemSupportingColor = theme.ColorNamePlaceHolder
)

func newNavListItem(leading, trailing fyne.Resource, headline, supporting string, action func()) *NavListItem {
	w := &NavListItem{
		backgroundRectangle: canvas.NewRectangle(theme.Color(navListItemBackgroundColor)),
		Headline:            headline,
		headlineText:        canvas.NewText(headline, theme.Color(navListItemHeadlineColor)),
		Leading:             leading,
		leadingImage:        NewImageFromResource(leading, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon))),
		OnTapped:            action,
		Supporting:          supporting,
		supportingText:      canvas.NewText(supporting, theme.Color(navListItemSupportingColor)),
		tapBGRectangle:      canvas.NewRectangle(color.Transparent),
		Trailing:            trailing,
		trailingImage:       NewImageFromResource(trailing, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon))),
	}
	w.ExtendBaseWidget(w)

	p := theme.Padding()
	w.backgroundRectangle.SetMinSize(fyne.NewSize(1, 14*p))
	w.headlineText.TextStyle.Bold = true

	w.tapAnim = newButtonTapAnimation(w.tapBGRectangle, w, w.Theme())
	w.tapAnim.Curve = fyne.AnimationEaseOut

	w.leadingWrapped = container.NewCenter(container.New(
		layout.NewCustomPaddedLayout(0, 0, p, 2*p),
		w.leadingImage,
	))
	w.trailingWrapped = container.NewCenter(container.New(
		layout.NewCustomPaddedLayout(0, 0, p, p),
		w.trailingImage,
	))
	return w
}

func (w *NavListItem) Refresh() {
	w.updateValues()
	w.updateVisibility()
	w.updateStyle()

	w.backgroundRectangle.Refresh()
	w.leadingImage.Refresh()
	w.trailingImage.Refresh()
	w.headlineText.Refresh()
	w.supportingText.Refresh()
	w.BaseWidget.Refresh()
}

func (w *NavListItem) updateValues() {
	w.leadingImage.Resource = w.Leading
	w.trailingImage.Resource = w.Trailing
	w.headlineText.Text = w.Headline
	w.supportingText.Text = w.Supporting
}

func (w *NavListItem) updateVisibility() {
	if w.Supporting == "" {
		w.supportingText.Hide()
	} else {
		w.supportingText.Show()
	}
	if w.Leading == nil {
		w.leadingWrapped.Hide()
	} else {
		w.leadingWrapped.Show()
	}
	if w.Trailing == nil {
		w.trailingWrapped.Hide()
	} else {
		w.trailingWrapped.Show()
	}
}

func (w *NavListItem) updateStyle() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	w.backgroundRectangle.CornerRadius = th.Size(theme.SizeNameButtonRadius)
	w.backgroundRectangle.FillColor = th.Color(navListItemBackgroundColor, v)
	w.headlineText.TextSize = th.Size(theme.SizeNameText)
	w.supportingText.TextSize = th.Size(theme.SizeNameText)
	w.tapBGRectangle.CornerRadius = th.Size(theme.SizeNameButtonRadius)

	if w.Disabled() {
		c := th.Color(navListItemDisabledColor, v)
		w.headlineText.Color = c
		w.supportingText.Color = c
		if w.Leading != nil {
			w.leadingImage.Resource = theme.NewDisabledResource(w.Leading)
		}
	} else {
		w.headlineText.Color = th.Color(navListItemHeadlineColor, v)
		var supportingColor fyne.ThemeColorName
		switch w.SupportingImportance {
		case widget.DangerImportance:
			supportingColor = theme.ColorNameError
		case widget.HighImportance:
			supportingColor = theme.ColorNamePrimary
		case widget.SuccessImportance:
			supportingColor = theme.ColorNameSuccess
		case widget.WarningImportance:
			supportingColor = theme.ColorNameWarning
		default:
			supportingColor = navListItemSupportingColor
		}
		w.supportingText.Color = th.Color(supportingColor, v)
		if w.Leading != nil {
			w.leadingImage.Resource = w.Leading
		}
	}
}

func (w *NavListItem) CreateRenderer() fyne.WidgetRenderer {
	w.updateValues()
	w.updateVisibility()
	w.updateStyle()
	c := container.NewBorder(
		nil,
		nil,
		w.leadingWrapped,
		w.trailingWrapped,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(
				layout.NewCustomPaddedVBoxLayout(0),
				w.headlineText,
				w.supportingText,
			),
			layout.NewSpacer(),
		),
	)
	p := theme.Padding()
	c2 := container.NewStack(
		w.backgroundRectangle,
		w.tapBGRectangle,
		container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 2*p, 2*p), c),
	)
	return widget.NewSimpleRenderer(c2)
}

func (w *NavListItem) Tapped(_ *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
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

		c := color.NRGBAModel.Convert(th.Color(theme.ColorNamePressed, v)).(color.NRGBA)
		fade := c.A - uint8(float32(c.A)*done)
		if fade > 0 {
			c.A = fade
			bg.FillColor = &c
		} else {
			bg.FillColor = color.Transparent
		}
		canvas.Refresh(bg)
		if done == 1.0 {
			w.isAnimating = false
		}
	})
}
