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
- TODO: Make widgets thread safe
*/

const (
	colorPrimary    = theme.ColorNamePrimary
	colorPill       = theme.ColorNameInputBorder
	colorForeground = theme.ColorNameForeground
	colorBackground = theme.ColorNameMenuBackground
	iconMinSize     = 24
)

type navBarItem struct {
	label   string
	icon    fyne.Resource
	content fyne.CanvasObject

	// OnSelected is an optional callback that fires when a new destination is selected.
	OnSelected func()

	// OnSelectedAgain is an optional callback that fires when the current destination is selected again.
	// This is often used for jumping back to the first page in a Navigator.
	OnSelectedAgain func()
}

// NewNavBarItem returns a new NavBarItem.
//
// A NavBarItem sets up a destination in a navigation bar.
func NewNavBarItem(label string, icon fyne.Resource, content fyne.CanvasObject) navBarItem {
	return navBarItem{label: label, icon: icon, content: content}
}

// A destination represents a fully configured item in a navigation bar.
type destination struct {
	widget.BaseWidget

	icon            *canvas.Image
	iconActive      fyne.Resource
	iconInactive    fyne.Resource
	id              int
	isEnabled       bool
	label           *canvas.Text
	navbar          *NavBar
	onSelected      func()
	onSelectedAgain func()
}

var _ fyne.Tappable = (*destination)(nil)

func newDestination(icon fyne.Resource, label string, nb *NavBar, id int, onSelected func(), onSelectedAgain func()) *destination {
	l := canvas.NewText(label, theme.Color(colorForeground))
	l.TextSize = theme.Size(theme.SizeNameCaptionText)
	i := canvas.NewImageFromResource(theme.NewThemedResource(icon))
	i.FillMode = canvas.ImageFillContain
	i.SetMinSize(fyne.NewSquareSize(iconMinSize))
	w := &destination{
		icon:            i,
		iconActive:      theme.NewPrimaryThemedResource(icon),
		iconInactive:    theme.NewThemedResource(icon),
		id:              id,
		label:           l,
		navbar:          nb,
		onSelected:      onSelected,
		onSelectedAgain: onSelectedAgain,
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

// A NavBar lets people switch between UI views on smaller devices.
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
		w.bar.Add(newDestination(it.icon, it.label, w, idx, it.OnSelected, it.OnSelectedAgain))
		b := it.content
		b.Hide()
		w.body.Add(b)
	}
	w.selectDestination(0)
	return w
}

func (w *NavBar) Select(idx int) {
	if idx > len(w.body.Objects)-1 {
		return
	}
	if idx == w.selected {
		if d := w.destination(idx); d.onSelectedAgain != nil {
			d.onSelectedAgain()
		}
		return
	}
	w.selectDestination(idx)
}

func (w *NavBar) destination(idx int) *destination {
	return w.bar.Objects[idx].(*destination)
}

func (w *NavBar) selectDestination(idx int) {
	current := w.selected
	w.body.Objects[current].Hide()
	w.body.Objects[idx].Show()
	w.destination(current).disable()
	d := w.destination(idx)
	d.enable()
	w.selected = idx
	if d.onSelected != nil {
		d.onSelected()
	}
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
		container.NewStack(w.bg, container.New(layout.NewCustomPaddedLayout(p*2, p*2, p, p), w.bar)),
		nil,
		nil,
		container.NewPadded(w.body),
	))
	return widget.NewSimpleRenderer(c)
}
