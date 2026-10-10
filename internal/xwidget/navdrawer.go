package xwidget

import (
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NavDrawerItem is a destination in a [NavDrawer].
type NavDrawerItem struct {
	// OnSelected is an optional callback that fires when this item is selected.
	OnSelected func()

	content  fyne.CanvasObject
	dest     *drawerDestination
	disabled bool
	drawer   *NavDrawer
}

// NewNavDrawerItem returns a new item for a [NavDrawer].
func NewNavDrawerItem(icon fyne.Resource, text string, content fyne.CanvasObject) *NavDrawerItem {
	it := &NavDrawerItem{content: content}
	it.dest = newDrawerDestination(icon, text, func() {
		if it.drawer != nil {
			it.drawer.Select(it)
		}
	})
	return it
}

// SetBadge sets the badge text of an item. An empty text hides the badge.
func (it *NavDrawerItem) SetBadge(text string) {
	it.dest.setBadge(text)
}

// SetText sets the text of an item.
func (it *NavDrawerItem) SetText(text string) {
	it.dest.setText(text)
}

// NavDrawer lets people switch between UI views on larger devices.
// It shows a scrollable list of items next to the content of the selected item.
// While no item is enabled, it shows an optional placeholder instead.
type NavDrawer struct {
	widget.DisableableWidget

	MinWidth float32 // minimum width of the navigation area

	body        *fyne.Container
	column      *fyne.Container
	indicator   *canvas.Rectangle
	items       []*NavDrawerItem
	placeholder fyne.CanvasObject
	scroll      *container.Scroll
	selected    *NavDrawerItem // nil while no item is enabled
	separator   *widget.Separator
}

// NewNavDrawer returns a new navigation drawer. The first item is selected initially.
//
// It panics if there are no items or an item already belongs to another drawer.
func NewNavDrawer(items ...*NavDrawerItem) *NavDrawer {
	if len(items) == 0 {
		panic("must define at least one item")
	}
	w := &NavDrawer{
		body:      container.NewStack(),
		column:    container.New(layout.NewVBoxLayout()),
		indicator: canvas.NewRectangle(color.Transparent),
		separator: widget.NewSeparator(),
	}
	w.ExtendBaseWidget(w)
	// the separator and indicator live inside the scroll, so the indicator scrolls with its item
	w.scroll = container.NewVScroll(
		container.New(&navDrawerStripLayout{w: w}, w.column, w.separator, w.indicator),
	)
	for _, it := range items {
		if it.drawer != nil {
			panic("item already belongs to a drawer")
		}
		it.drawer = w
		w.column.Add(it.dest)
		w.items = append(w.items, it)
		it.content.Hide()
		w.body.Add(it.content)
	}
	w.selectItem(items[0])
	return w
}

// Select switches to an item. Does nothing when the item is disabled.
func (w *NavDrawer) Select(it *NavDrawerItem) {
	if !w.owns(it) || it.dest.Disabled() {
		return
	}
	w.selectItem(it)
}

// Selected returns the currently selected item or nil when no item is enabled.
func (w *NavDrawer) Selected() *NavDrawerItem {
	return w.selected
}

// SetPlaceholder sets an object to show instead of any content while no item is enabled.
// A nil object removes the placeholder.
func (w *NavDrawer) SetPlaceholder(obj fyne.CanvasObject) {
	if w.placeholder != nil {
		w.body.Remove(w.placeholder)
	}
	w.placeholder = obj
	if obj != nil {
		if w.selected == nil {
			obj.Show()
		} else {
			obj.Hide()
		}
		w.body.Add(obj)
	}
	w.body.Refresh()
}

// EnableItem enables an item.
// When no item was enabled before, the drawer switches to the first enabled item.
func (w *NavDrawer) EnableItem(it *NavDrawerItem) {
	if !w.owns(it) {
		return
	}
	it.disabled = false
	w.updateItemState(it)
	w.updateSelection()
}

// DisableItem disables an item. Disabled items can not be selected.
// When the selected item is disabled, the drawer switches to the first enabled item
// or shows the placeholder when there is none.
func (w *NavDrawer) DisableItem(it *NavDrawerItem) {
	if !w.owns(it) {
		return
	}
	it.disabled = true
	w.updateItemState(it)
	w.updateSelection()
}

// ItemEnabled reports whether an item is enabled.
func (w *NavDrawer) ItemEnabled(it *NavDrawerItem) bool {
	return w.owns(it) && !it.dest.Disabled()
}

// Disable disables all items and shows the placeholder.
func (w *NavDrawer) Disable() {
	if w.Disabled() {
		return
	}
	w.DisableableWidget.Disable()
	for _, it := range w.items {
		w.updateItemState(it)
	}
	w.updateSelection()
}

// Enable enables all items, except those disabled with [NavDrawer.DisableItem],
// and switches to the first enabled item.
func (w *NavDrawer) Enable() {
	if !w.Disabled() {
		return
	}
	w.DisableableWidget.Enable()
	for _, it := range w.items {
		w.updateItemState(it)
	}
	w.updateSelection()
}

// ScrollToTop scrolls the navigation area to the top.
func (w *NavDrawer) ScrollToTop() {
	w.scroll.ScrollToOffset(fyne.Position{})
}

// updateSelection switches to the first enabled item when the selected item is disabled
// and shows the placeholder when no item is enabled.
func (w *NavDrawer) updateSelection() {
	if w.selected != nil && !w.selected.dest.Disabled() {
		return
	}
	first := slices.IndexFunc(w.items, func(it *NavDrawerItem) bool {
		return !it.dest.Disabled()
	})
	wasEmpty := w.selected == nil
	if first == -1 {
		if wasEmpty {
			return
		}
		w.selected.dest.setActive(false)
		w.selected.content.Hide()
		w.selected = nil
		if w.placeholder != nil {
			w.placeholder.Show()
		}
		w.Refresh() // showing a never-visible object may not repaint
		return
	}
	if wasEmpty && w.placeholder != nil {
		w.placeholder.Hide()
	}
	w.selectItem(w.items[first])
	if wasEmpty {
		w.ScrollToTop()
		w.Refresh()
	}
}

func (w *NavDrawer) owns(it *NavDrawerItem) bool {
	return it != nil && it.drawer == w
}

func (w *NavDrawer) updateItemState(it *NavDrawerItem) {
	if w.Disabled() || it.disabled {
		it.dest.Disable()
	} else {
		it.dest.Enable()
	}
}

func (w *NavDrawer) selectItem(it *NavDrawerItem) {
	if it == w.selected {
		return
	}
	if current := w.selected; current != nil {
		current.dest.setActive(false)
		current.content.Hide()
	}
	it.dest.setActive(true)
	it.content.Show()
	w.selected = it
	w.updateIndicator()
	if it.OnSelected != nil {
		it.OnSelected()
	}
}

// updateIndicator places the indicator on the separator next to the selected item.
func (w *NavDrawer) updateIndicator() {
	if w.selected == nil {
		return
	}
	dest := w.selected.dest
	w.indicator.Move(fyne.NewPos(w.separator.Position().X, w.column.Position().Y+dest.Position().Y))
	w.indicator.Resize(fyne.NewSize(w.separator.Size().Width, dest.Size().Height))
	w.indicator.Refresh()
}

func (w *NavDrawer) CreateRenderer() fyne.WidgetRenderer {
	p := w.Theme().Size(theme.SizeNamePadding)
	r := &navDrawerRenderer{
		// negative padding lets the separator span the full height
		content: container.New(layout.NewCustomPaddedLayout(-p, -p, 0, 0),
			container.NewBorder(nil, nil, w.scroll, nil, w.body),
		),
		w: w,
	}
	r.updateColors()
	return r
}

type navDrawerRenderer struct {
	content *fyne.Container
	w       *NavDrawer
}

func (r *navDrawerRenderer) Destroy() {}

func (r *navDrawerRenderer) Layout(size fyne.Size) {
	r.content.Resize(size)
}

func (r *navDrawerRenderer) MinSize() fyne.Size {
	return r.content.MinSize()
}

func (r *navDrawerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.content}
}

func (r *navDrawerRenderer) Refresh() {
	r.updateColors()
	r.content.Refresh()
}

func (r *navDrawerRenderer) updateColors() {
	th := r.w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	// transparent instead of hidden, because showing a never-visible object may not repaint
	if r.w.selected == nil {
		r.w.indicator.FillColor = color.Transparent
	} else {
		r.w.indicator.FillColor = th.Color(theme.ColorNamePrimary, v)
	}
	r.w.indicator.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	r.w.indicator.Refresh()
}

// navDrawerStripLayout lays out the item column with a trailing separator
// and keeps the indicator next to the selected item.
type navDrawerStripLayout struct {
	w *NavDrawer
}

func (l *navDrawerStripLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	w := l.w
	p := w.Theme().Size(theme.SizeNamePadding)
	sepWidth := w.separator.MinSize().Width
	w.column.Move(fyne.NewPos(0, p))
	w.column.Resize(fyne.NewSize(size.Width-sepWidth, w.column.MinSize().Height))
	w.separator.Move(fyne.NewPos(size.Width-sepWidth, 0))
	w.separator.Resize(fyne.NewSize(sepWidth, size.Height))
	w.updateIndicator()
}

func (l *navDrawerStripLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	w := l.w
	p := w.Theme().Size(theme.SizeNamePadding)
	s := w.column.MinSize()
	return fyne.NewSize(max(w.MinWidth, s.Width+w.separator.MinSize().Width), s.Height+2*p)
}

// drawerDestination is a tappable row with icon, text and badge in a [NavDrawer].
type drawerDestination struct {
	widget.DisableableWidget

	badge        *widget.Label
	hover        *canvas.Rectangle
	hovered      bool
	icon         *canvas.Image
	iconDisabled fyne.Resource
	iconEnabled  fyne.Resource
	iconSelected fyne.Resource
	isActive     bool
	onTapped     func()
	title        *widget.Label
}

var _ fyne.Tappable = (*drawerDestination)(nil)
var _ desktop.Hoverable = (*drawerDestination)(nil)
var _ desktop.Cursorable = (*drawerDestination)(nil)

func newDrawerDestination(icon fyne.Resource, text string, onTapped func()) *drawerDestination {
	iconSize := theme.Size(theme.SizeNameInlineIcon)
	w := &drawerDestination{
		badge:        widget.NewLabel(""),
		hover:        canvas.NewRectangle(color.Transparent),
		icon:         NewImageFromResource(theme.NewThemedResource(icon), fyne.NewSquareSize(iconSize)),
		iconEnabled:  theme.NewThemedResource(icon),
		iconSelected: theme.NewPrimaryThemedResource(icon),
		iconDisabled: theme.NewDisabledResource(icon),
		onTapped:     onTapped,
		title:        widget.NewLabel(text),
	}
	w.ExtendBaseWidget(w)
	w.title.Truncation = fyne.TextTruncateEllipsis
	w.badge.Hide()
	return w
}

func (w *drawerDestination) setActive(active bool) {
	w.isActive = active
	w.Refresh()
}

func (w *drawerDestination) setBadge(text string) {
	w.badge.Text = text
	w.Refresh()
}

func (w *drawerDestination) setText(text string) {
	w.title.Text = text
	w.Refresh()
}

func (w *drawerDestination) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	var importance widget.Importance
	switch {
	case w.Disabled():
		w.icon.Resource = w.iconDisabled
		importance = widget.LowImportance
	case w.isActive:
		w.icon.Resource = w.iconSelected
		importance = widget.HighImportance
	default:
		w.icon.Resource = w.iconEnabled
		importance = widget.MediumImportance
	}
	w.title.Importance = importance
	w.badge.Importance = importance
	bold := w.isActive && !w.Disabled()
	w.title.TextStyle.Bold = bold
	w.badge.TextStyle.Bold = bold
	if w.badge.Text != "" {
		w.badge.Show()
	} else {
		w.badge.Hide()
	}
	w.hover.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	if w.hovered && !w.Disabled() {
		w.hover.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		w.hover.FillColor = color.Transparent
	}
	w.BaseWidget.Refresh()
}

func (w *drawerDestination) Tapped(_ *fyne.PointEvent) {
	if w.Disabled() || w.onTapped == nil {
		return
	}
	w.onTapped()
}

func (w *drawerDestination) Cursor() desktop.Cursor {
	if w.hovered && !w.Disabled() {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *drawerDestination) MouseIn(_ *desktop.MouseEvent) {
	w.hovered = true
	w.Refresh()
}

func (w *drawerDestination) MouseMoved(_ *desktop.MouseEvent) {}

func (w *drawerDestination) MouseOut() {
	w.hovered = false
	w.Refresh()
}

func (w *drawerDestination) CreateRenderer() fyne.WidgetRenderer {
	p := w.Theme().Size(theme.SizeNamePadding)
	c := container.NewStack(
		w.hover,
		container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, p),
			container.NewBorder(nil, nil, container.NewCenter(w.icon), w.badge, w.title),
		),
	)
	return widget.NewSimpleRenderer(c)
}
