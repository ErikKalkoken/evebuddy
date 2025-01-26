package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/*
- TODO: Remove padding between navbar and body
- TODO: Double-check we have sufficient padding on the utmost border of the app
*/

const (
	colorPrimary    = theme.ColorNamePrimary
	colorPill       = theme.ColorNameInputBorder
	colorForeground = theme.ColorNameForeground
	colorBackground = theme.ColorNameMenuBackground
	iconMinSize     = 24
)

type navBarItem struct {
	label        string
	iconActive   fyne.Resource
	iconInactive fyne.Resource
	content      fyne.CanvasObject
}

func NewNavBarItem(label string, iconActive fyne.Resource, iconInactive fyne.Resource, content fyne.CanvasObject) navBarItem {
	return navBarItem{label: label, iconActive: iconActive, iconInactive: iconInactive, content: content}
}

type destination struct {
	widget.BaseWidget
	iconActive   fyne.Resource
	iconInactive fyne.Resource
	icon         *canvas.Image
	label        *canvas.Text
	navbar       *NavBar
	id           int
	isEnabled    bool
}

var _ fyne.Tappable = (*destination)(nil)

func newDestination(iconActive fyne.Resource, iconInactive fyne.Resource, label string, nb *NavBar, id int) *destination {
	l := canvas.NewText(label, theme.Color(colorForeground))
	l.TextSize = theme.Size(theme.SizeNameCaptionText)
	i := canvas.NewImageFromResource(theme.NewThemedResource(iconInactive))
	i.FillMode = canvas.ImageFillContain
	i.SetMinSize(fyne.NewSquareSize(iconMinSize))
	w := &destination{
		iconActive:   theme.NewPrimaryThemedResource(iconActive),
		iconInactive: theme.NewThemedResource(iconInactive),
		icon:         i,
		label:        l,
		navbar:       nb,
		id:           id,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *destination) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	if w.isEnabled {
		w.label.Color = th.Color(colorPrimary, v)
		w.icon.Resource = w.iconActive
	} else {
		w.label.Color = th.Color(colorForeground, v)
		w.icon.Resource = w.iconInactive
	}
	w.label.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *destination) Tapped(_ *fyne.PointEvent) {
	w.navbar.Select(w.id)
}

func (w *destination) TappedSecondary(_ *fyne.PointEvent) {
}

func (w *destination) enable() {
	w.isEnabled = true
	w.Refresh()
}

func (w *destination) disable() {
	w.isEnabled = false
	w.Refresh()
}

func (w *destination) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		container.New(layout.NewCustomPaddedLayout(-p/2, -p/2, 0, 0), container.NewCenter(w.label)),
		nil,
		nil,
		w.icon,
	)
	return widget.NewSimpleRenderer(c)
}

type NavBar struct {
	widget.BaseWidget

	bar      *fyne.Container
	body     *fyne.Container
	bg       *canvas.Rectangle
	selected int
}

func NewNavBar(items ...navBarItem) *NavBar {
	if len(items) == 0 {
		panic("must define at least one item")
	}
	w := &NavBar{
		bar:  container.NewGridWithRows(1),
		body: container.NewStack(),
		bg:   canvas.NewRectangle(theme.Color(colorBackground)),
	}
	w.ExtendBaseWidget(w)
	for idx, it := range items {
		w.bar.Add(newDestination(it.iconActive, it.iconInactive, it.label, w, idx))
		b := it.content
		b.Hide()
		w.body.Add(b)
	}
	w.Select(0)
	return w
}

func (w *NavBar) Select(idx int) {
	if idx > len(w.body.Objects)-1 {
		return
	}
	current := w.selected
	w.body.Objects[current].Hide()
	w.body.Objects[idx].Show()
	w.bar.Objects[current].(*destination).disable()
	w.bar.Objects[idx].(*destination).enable()
	w.selected = idx
}

func (w *NavBar) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(colorBackground, v)
	w.bg.Refresh()
	w.BaseWidget.Refresh()
}

func (w *NavBar) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), container.NewBorder(
		nil,
		container.NewStack(w.bg, container.New(layout.NewCustomPaddedLayout(p*3, p*3, p, p), w.bar)),
		nil,
		nil,
		container.NewPadded(w.body),
	))
	return widget.NewSimpleRenderer(c)
}
