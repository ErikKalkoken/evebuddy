package widget

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
	colorBarBackground = theme.ColorNameMenuBackground
	colorForeground    = theme.ColorNameForeground
	colorIndicator     = theme.ColorNameInputBorder
	colorPrimary       = theme.ColorNamePrimary
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

// A NavBar lets people switch between UI views on smaller devices.
type NavBar struct {
	widget.BaseWidget

	bar          *fyne.Container
	bg           *canvas.Rectangle
	body         *fyne.Container
	destinations *fyne.Container
	selected     int
}

// NewNavBar returns new navbar.
// It is recommended to have at most 5 destinations.
//
// It panics if not at least one destination is provided.
func NewNavBar(items ...navBarItem) *NavBar {
	if len(items) == 0 {
		panic("must define at least one item")
	}
	w := &NavBar{
		destinations: container.NewGridWithRows(1),
		body:         container.NewStack(),
		bg:           canvas.NewRectangle(theme.Color(colorBarBackground)),
	}
	w.ExtendBaseWidget(w)

	for idx, it := range items {
		w.destinations.Add(newDestination(it.icon, it.label, w, idx, it.OnSelected, it.OnSelectedAgain))
		b := it.content
		b.Hide()
		w.body.Add(b)
	}
	p := theme.Padding()
	w.bar = container.New(
		layout.NewCustomPaddedLayout(-3*p, 0, 0, 0),
		container.NewStack(
			w.bg,
			container.New(layout.NewCustomPaddedLayout(p*3, p*3, p*2, p*2), w.destinations),
		))

	w.selectDestination(0)
	return w
}

// Select switches to a new destination.
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

// HideBar hides the nav bar, while still showing the rest of the page.
func (w *NavBar) HideBar() {
	w.destinations.Hide()
}

// ShowBar shows the nav bar again.
func (w *NavBar) ShowBar() {
	w.destinations.Show()
}

func (w *NavBar) destination(idx int) *destination {
	return w.destinations.Objects[idx].(*destination)
}

func (w *NavBar) selectDestination(idx int) {
	current := w.selected
	w.destination(current).disable()
	d := w.destination(idx)
	d.enable()
	w.body.Objects[current].Hide()
	w.body.Objects[idx].Show()
	w.selected = idx
	if d.onSelected != nil {
		d.onSelected()
	}
}

func (w *NavBar) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(colorBarBackground, v)
	w.bg.Refresh()
	w.destinations.Refresh()
	w.BaseWidget.Refresh()
}

func (w *NavBar) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.New(
		layout.NewCustomPaddedLayout(-p, -p, -p, -p),
		container.NewBorder(
			nil,
			w.bar,
			nil,
			nil,
			container.New(layout.NewCustomPaddedLayout(p, p, p, p), w.body),
		))
	return widget.NewSimpleRenderer(c)
}
