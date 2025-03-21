package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	colorIndicator = theme.ColorNameInputBorder
	colorBadge     = theme.ColorNamePrimary
)

// A destination represents a fully configured item in a navigation bar.
type destination struct {
	widget.DisableableWidget

	badge           *canvas.Circle
	icon            *canvas.Image
	iconActive      fyne.Resource
	iconDisabled    fyne.Resource
	iconInactive    fyne.Resource
	id              int // id of a destination in a navbar
	indicator       *canvas.Rectangle
	isActive        bool
	hasBadge        bool
	label           *canvas.Text
	navbar          *NavBar
	onSelected      func()
	onSelectedAgain func()
	tapAnim         *fyne.Animation
}

var _ fyne.Tappable = (*destination)(nil)

func newDestination(icon fyne.Resource, label string, nb *NavBar, id int, onSelected func(), onSelectedAgain func()) *destination {
	l := canvas.NewText(label, theme.Color(colorForeground))
	l.TextSize = theme.Size(theme.SizeNameCaptionText)
	iconImage := NewImageFromResource(theme.NewThemedResource(icon), fyne.NewSquareSize(1.5*theme.Size(theme.SizeNameInlineIcon)))
	pill := canvas.NewRectangle(theme.Color(colorIndicator))
	pill.CornerRadius = 12
	badge := canvas.NewCircle(theme.Color(colorBadge))
	w := &destination{
		badge:           badge,
		icon:            iconImage,
		iconActive:      theme.NewPrimaryThemedResource(icon),
		iconInactive:    theme.NewThemedResource(icon),
		iconDisabled:    theme.NewDisabledResource(icon),
		id:              id,
		label:           l,
		navbar:          nb,
		onSelected:      onSelected,
		onSelectedAgain: onSelectedAgain,
		indicator:       pill,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *destination) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.badge.FillColor = th.Color(colorBadge, v)
	if w.isActive {
		w.label.Color = th.Color(colorPrimary, v)
		w.label.TextStyle.Bold = true
		w.icon.Resource = w.iconActive
		w.indicator.FillColor = th.Color(colorIndicator, v)
		w.indicator.Show()
		w.indicator.Refresh()
	} else if w.Disabled() {
		w.label.Color = th.Color(theme.ColorNameDisabled, v)
		w.label.TextStyle.Bold = false
		w.icon.Resource = w.iconDisabled
		w.indicator.Hide()
	} else {
		w.label.Color = th.Color(colorForeground, v)
		w.label.TextStyle.Bold = false
		w.icon.Resource = w.iconInactive
		w.indicator.Hide()
	}
	w.label.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *destination) Tapped(_ *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	w.navbar.Select(w.id)
}

func (w *destination) TappedSecondary(_ *fyne.PointEvent) {
}

func (w *destination) activate(showAnimation bool) {
	w.isActive = true
	w.tapAnim.Stop()
	if showAnimation && fyne.CurrentApp().Settings().ShowAnimations() {
		w.tapAnim.Start()
	} else {
		// set to animation end state
		s := w.indicatorSize()
		w.indicator.Resize(s)
		w.indicator.Move(fyne.NewPos(-s.Width/2, -s.Height/2))
	}
	w.Refresh()
}

func (w *destination) deactivate() {
	w.isActive = false
	w.tapAnim.Stop()
	w.Refresh()
}

func (w *destination) setBadge(show bool) {
	if show {
		w.badge.Show()
	} else {
		w.badge.Hide()
	}
}

func (w *destination) indicatorSize() fyne.Size {
	v := w.Theme().Size(theme.SizeNameInlineIcon)
	return fyne.NewSize(2.85*v, 1.3*v)
}

func (w *destination) CreateRenderer() fyne.WidgetRenderer {
	s := w.indicatorSize()
	w.tapAnim = canvas.NewSizeAnimation(
		s.SubtractWidthHeight(s.Width*0.95, 0), s, defaultAnimationDuration, func(s fyne.Size) {
			w.indicator.Resize(s)
			w.indicator.Move(fyne.NewPos(-s.Width/2, -s.Height/2))
		})
	w.tapAnim.Curve = fyne.AnimationEaseOut
	w.badge.Resize(fyne.NewSquareSize(6))
	w.badge.Hide()
	c := container.NewVBox(
		container.NewStack(
			container.NewCenter(container.NewWithoutLayout(w.indicator)),
			container.NewCenter(
				container.NewStack(
					container.NewCenter(w.icon),
					container.NewHBox(layout.NewSpacer(), container.NewWithoutLayout(w.badge)),
				),
			),
		),
		container.NewHBox(layout.NewSpacer(), w.label, layout.NewSpacer()),
	)
	return widget.NewSimpleRenderer(c)
}
