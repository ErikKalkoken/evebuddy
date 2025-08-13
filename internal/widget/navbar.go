package widget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	colorIndicator = theme.ColorNameInputBorder
)

// A destination represents a fully configured item in a navigation bar.
type destination struct {
	widget.DisableableWidget

	badge           *canvas.Circle
	badgeImportance widget.Importance
	icon            *canvas.Image
	iconActive      fyne.Resource
	iconDisabled    fyne.Resource
	iconInactive    fyne.Resource
	id              int // id of a destination in a navbar
	indicator       *canvas.Rectangle
	isActive        bool
	label           *canvas.Text
	navbar          *NavBar
	onSelected      func()
	onSelectedAgain func()
	tapAnim         *fyne.Animation
}

var _ fyne.Tappable = (*destination)(nil)

func newDestination(icon fyne.Resource, label string, nb *NavBar, id int, onSelected func(), onSelectedAgain func()) *destination {
	l := canvas.NewText(label, theme.Color(colorForeground))
	l.TextSize = 10
	iconImage := NewImageFromResource(
		theme.NewThemedResource(icon),
		fyne.NewSquareSize(1.2*theme.Size(theme.SizeNameInlineIcon)),
	)
	pill := canvas.NewRectangle(theme.Color(colorIndicator))
	pill.CornerRadius = 12
	badge := canvas.NewCircle(color.Transparent)
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
	var c color.Color
	switch w.badgeImportance {
	case widget.DangerImportance:
		c = th.Color(theme.ColorNameError, v)
	case widget.HighImportance:
		c = th.Color(theme.ColorNamePrimary, v)
	case widget.LowImportance:
		c = th.Color(theme.ColorNameDisabled, v)
	case widget.WarningImportance:
		c = th.Color(theme.ColorNameWarning, v)
	case widget.MediumImportance:
		c = th.Color(theme.ColorNameForeground, v)
	case widget.SuccessImportance:
		c = th.Color(theme.ColorNameSuccess, v)
	}
	w.badge.FillColor = c
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

func (w *destination) showBadge(i widget.Importance) {
	w.badgeImportance = i
	w.badge.Show()
	w.Refresh()
}

func (w *destination) hideBadge() {
	w.badge.Hide()
}

func (w *destination) indicatorSize() fyne.Size {
	v := w.Theme().Size(theme.SizeNameInlineIcon)
	return fyne.NewSize(2.85*v, 1.3*v)
}

func (w *destination) CreateRenderer() fyne.WidgetRenderer {
	s := w.indicatorSize()
	w.tapAnim = canvas.NewSizeAnimation(
		s.SubtractWidthHeight(s.Width*0.95, 0),
		s,
		defaultAnimationDuration,
		func(s fyne.Size) {
			w.indicator.Resize(s)
			w.indicator.Move(fyne.NewPos(-s.Width/2, -s.Height/2+0.5))
		},
	)
	w.tapAnim.Curve = fyne.AnimationEaseOut

	w.badge.Resize(fyne.NewSquareSize(6))
	w.badge.Hide()

	p := theme.Padding()
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(p*1.2),
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

// Enabled reports whether a destination is enabled.
func (w *NavBar) Enabled(id int) bool {
	if id < 0 || id >= len(w.destinations.Objects) {
		return false // out of bounds
	}
	return !w.destinations.Objects[id].(*destination).Disabled()
}

// Selected returns the ID of the currently selected destination and reports whether a tab is selected.
func (w *NavBar) Selected() (int, bool) {
	return w.selectedIdx, w.selectedIdx != -1
}

// HideBar hides the nav bar, while still showing the rest of the page.
func (w *NavBar) HideBar() {
	w.bar.Hide()
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
	w.bar.Show()
}

// ShowBadge shows the badge of a destination.
func (w *NavBar) ShowBadge(id int, importance widget.Importance) {
	d := w.destination(id)
	if d == nil {
		return
	}
	d.showBadge(importance)
}

// HideBadge hides the badge of a destination.
func (w *NavBar) HideBadge(id int) {
	d := w.destination(id)
	if d == nil {
		return
	}
	d.hideBadge()
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
