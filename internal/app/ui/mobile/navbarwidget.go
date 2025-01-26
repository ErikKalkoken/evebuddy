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
	colorPill       = theme.ColorNameInputBorder
	colorLabel      = theme.ColorNameForeground
	colorBackground = theme.ColorNameMenuBackground
	iconMinSize     = 24
)

type navBarItem struct {
	label   string
	icon    fyne.Resource
	content fyne.CanvasObject
}

func NewNavBarItem(label string, icon fyne.Resource, content fyne.CanvasObject) navBarItem {
	return navBarItem{label: label, icon: icon, content: content}
}

type destination struct {
	widget.BaseWidget
	icon   *canvas.Image
	pill   *canvas.Rectangle
	label  *canvas.Text
	navbar *NavBar
	id     int
}

var _ fyne.Tappable = (*destination)(nil)

func newDestination(icon fyne.Resource, label string, nb *NavBar, id int) *destination {
	l := canvas.NewText(label, theme.Color(colorLabel))
	l.TextSize = theme.Size(theme.SizeNameCaptionText)
	p := canvas.NewRectangle(theme.Color(colorPill))
	p.Hide()
	p.CornerRadius = 10
	i := canvas.NewImageFromResource(icon)
	i.FillMode = canvas.ImageFillContain
	i.SetMinSize(fyne.NewSquareSize(iconMinSize))
	w := &destination{
		icon:   i,
		pill:   p,
		label:  l,
		navbar: nb,
		id:     id,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *destination) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.label.Color = th.Color(colorLabel, v)
	w.label.Refresh()
	w.pill.FillColor = th.Color(colorPill, v)
	w.pill.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *destination) Tapped(_ *fyne.PointEvent) {
	w.navbar.Select(w.id)
}

func (w *destination) TappedSecondary(_ *fyne.PointEvent) {
}

func (w *destination) enable() {
	w.label.TextStyle.Bold = true
	w.label.Refresh()
	w.pill.Show()
	w.pill.Refresh()
}

func (w *destination) disable() {
	w.label.TextStyle.Bold = false
	w.label.Refresh()
	w.pill.Hide()
	w.pill.Refresh()
}

func (w *destination) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		container.New(layout.NewCustomPaddedLayout(-p/2, -p/2, 0, 0), container.NewCenter(w.label)),
		nil,
		nil,
		container.NewStack(
			w.pill,
			container.New(layout.NewCustomPaddedLayout(p/2, p/2, 0, 0), w.icon),
		),
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
		w.bar.Add(newDestination(it.icon, it.label, w, idx))
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
