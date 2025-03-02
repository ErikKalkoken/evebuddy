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
- TODO: Make widgets thread safe
*/

const (
	colorBarBackground = theme.ColorNameMenuBackground
	colorForeground    = theme.ColorNameForeground
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

// NewDestinationDef returns a new destination definition for a [NavBar].
func NewDestinationDef(label string, icon fyne.Resource, content fyne.CanvasObject) destinationDef {
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

	for id, it := range items {
		w.destinations.Add(newDestination(it.icon, it.label, w, id, it.OnSelected, it.OnSelectedAgain))
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

// Enable enables a destination.
func (w *NavBar) Enable(id int) {
	if id < 0 || id >= len(w.destinations.Objects) {
		return // out of bounds
	}
	w.destinations.Objects[id].(*destination).Enable()
}

// Disable disables a destination. Disabled destination can not be interacted with.
func (w *NavBar) Disable(id int) {
	if id < 0 || id >= len(w.destinations.Objects) {
		return // out of bounds
	}
	w.destinations.Objects[id].(*destination).Disable()
}

// HideBar hides the nav bar, while still showing the rest of the page.
func (w *NavBar) HideBar() {
	w.destinations.Hide()
}

// Select switches to a new destination.
func (w *NavBar) Select(id int) {
	if id < 0 || id >= len(w.destinations.Objects) {
		return // out of bounds
	}
	if id == w.selectedIdx {
		if d := w.destination(id); d.onSelectedAgain != nil {
			d.onSelectedAgain()
		}
		return
	}
	w.selectDestination(id)
}

// ShowBar shows the nav bar again.
func (w *NavBar) ShowBar() {
	w.destinations.Show()
}

// SetBadge shows or hides the badge of a destination.
func (w *NavBar) SetBadge(id int, show bool) {
	d := w.destination(id)
	if d == nil {
		return
	}
	d.setBadge(show)
}

// destination returns the destination object for a given id.
// Returns nil for invalid IDs.
func (w *NavBar) destination(id int) *destination {
	if id < 0 || id >= len(w.destinations.Objects) {
		return nil // out of bounds
	}
	return w.destinations.Objects[id].(*destination)
}

// selectDestination switches to a new destination.
func (w *NavBar) selectDestination(id int) {
	currentIdx := w.selectedIdx
	currentDest := w.destination(currentIdx)
	newDest := w.destination(id)
	if newDest == nil {
		return
	}
	newDest.activate(currentDest != nil)
	if currentDest != nil {
		currentDest.deactivate()
	}
	if currentIdx >= 0 {
		w.body.Objects[currentIdx].Hide()
	}
	w.body.Objects[id].Show()
	w.selectedIdx = id
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
