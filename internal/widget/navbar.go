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

type destinationDef struct {
	label   string
	icon    fyne.Resource
	content fyne.CanvasObject

	// OnSelected is an optional callback that fires when a new destination is selected.
	OnSelected func()

	// OnSelectedAgain is an optional callback that fires when the current destination is selected again.
	// This is often used for jumping back to the first page in a Navigator.
	OnSelectedAgain func()
}

// NewDestination returns a new destination for a [NavBar].
func NewDestination(label string, icon fyne.Resource, content fyne.CanvasObject) destinationDef {
	return destinationDef{label: label, icon: icon, content: content}
}

// A NavBar lets people switch between UI views on smaller devices.
type NavBar struct {
	widget.BaseWidget

	bar          *fyne.Container
	bg           *canvas.Rectangle
	body         *fyne.Container
	destinations *fyne.Container
	selectedIdx  int
}

// NewNavBar returns new navbar.
// It is recommended to have at most 5 destinations.
//
// It panics if no destinations are provided.
func NewNavBar(items ...destinationDef) *NavBar {
	if len(items) == 0 {
		panic("must define at least one item")
	}
	w := &NavBar{
		destinations: container.NewGridWithRows(1),
		body:         container.NewStack(),
		bg:           canvas.NewRectangle(theme.Color(colorBarBackground)),
		selectedIdx:  -1,
	}
	w.ExtendBaseWidget(w)

	for idx, it := range items {
		w.destinations.Add(newDestination(it.icon, it.label, w, idx, it.OnSelected, it.OnSelectedAgain))
		b := it.content
		b.Hide()
		w.body.Add(b)
	}
	p := theme.Padding()
	top := 3.5 * p
	w.bar = container.New(
		layout.NewCustomPaddedLayout(-top, 0, 0, 0),
		container.NewStack(
			w.bg,
			container.New(layout.NewCustomPaddedLayout(top, p*4, p*2, p*2), w.destinations),
		))

	w.selectDestination(0)
	return w
}

// Select switches to a new destination.
func (w *NavBar) Select(idx int) {
	if idx < 0 || idx >= len(w.body.Objects) {
		return // out of bounds
	}
	if idx == w.selectedIdx {
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
	if idx < 0 || idx >= len(w.destinations.Objects) {
		return nil // out of bounds
	}
	return w.destinations.Objects[idx].(*destination)
}

// selectDestination switches to a new destination.
func (w *NavBar) selectDestination(idx int) {
	currentIdx := w.selectedIdx
	currentDest := w.destination(currentIdx)
	newDest := w.destination(idx)
	newDest.enable()
	if currentDest != nil {
		currentDest.disable()
	}
	// start animation
	if currentIdx >= 0 {
		w.body.Objects[currentIdx].Hide()
	}
	w.body.Objects[idx].Show()
	// stop animation
	w.selectedIdx = idx
	if newDest.onSelected != nil {
		newDest.onSelected()
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
