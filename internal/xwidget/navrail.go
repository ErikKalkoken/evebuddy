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
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

const (
	navRailIconScale       = 1.5 // × inline icon size
	navRailItemGapPaddings = 1   // × theme padding
	navRailHoverPaddings   = 3   // × theme padding
)

// railDestination is a tappable icon in a [NavRail].
type railDestination struct {
	widget.DisableableWidget
	ttwidget.ToolTipWidgetExtend

	hover        *canvas.Rectangle
	hovered      bool
	icon         *canvas.Image
	iconDisabled fyne.Resource
	iconEnabled  fyne.Resource
	iconSelected fyne.Resource
	isActive     bool
	onTapped     func()
}

var _ fyne.Tappable = (*railDestination)(nil)
var _ desktop.Hoverable = (*railDestination)(nil)
var _ desktop.Cursorable = (*railDestination)(nil)

func newRailDestination(icon fyne.Resource, tooltip string, onTapped func()) *railDestination {
	iconSize := theme.Size(theme.SizeNameInlineIcon) * navRailIconScale
	w := &railDestination{
		hover:        canvas.NewRectangle(color.Transparent),
		icon:         NewImageFromResource(theme.NewThemedResource(icon), fyne.NewSquareSize(iconSize)),
		iconEnabled:  theme.NewThemedResource(icon),
		iconSelected: theme.NewPrimaryThemedResource(icon),
		iconDisabled: theme.NewDisabledResource(icon),
		onTapped:     onTapped,
	}
	w.ExtendBaseWidget(w)
	// stays visible but transparent, so the destination never changes size
	w.hover.SetMinSize(fyne.NewSquareSize(iconSize + navRailHoverPaddings*theme.Padding()))
	w.SetToolTip(tooltip)
	return w
}

func (w *railDestination) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.DisableableWidget.ExtendBaseWidget(wid)
}

func (w *railDestination) setActive(active bool) {
	w.isActive = active
	w.Refresh()
}

func (w *railDestination) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	switch {
	case w.Disabled():
		w.icon.Resource = w.iconDisabled
	case w.isActive:
		w.icon.Resource = w.iconSelected
	default:
		w.icon.Resource = w.iconEnabled
	}
	radius := th.Size(theme.SizeNameButtonRadius)
	w.hover.CornerRadius = radius
	if w.hovered && !w.Disabled() {
		w.hover.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		w.hover.FillColor = color.Transparent
	}
	w.hover.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *railDestination) Tapped(_ *fyne.PointEvent) {
	if w.Disabled() || w.onTapped == nil {
		return
	}
	w.onTapped()
}

func (w *railDestination) Cursor() desktop.Cursor {
	if w.hovered && !w.Disabled() {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *railDestination) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseIn(e)
	w.hovered = true
	w.Refresh()
}

func (w *railDestination) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseMoved(e)
}

func (w *railDestination) MouseOut() {
	w.ToolTipWidgetExtend.MouseOut()
	w.hovered = false
	w.Refresh()
}

func (w *railDestination) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(
		container.NewCenter(w.hover),
		container.NewCenter(w.icon),
	)
	return widget.NewSimpleRenderer(c)
}

// NavRailItem is a destination in a [NavRail].
type NavRailItem struct {
	// OnSelected is an optional callback that fires when this item is selected.
	OnSelected func()

	// OnSelectedAgain is an optional callback that fires when this item is selected while already selected.
	OnSelectedAgain func()

	content  fyne.CanvasObject
	dest     *railDestination
	icon     fyne.Resource
	onTapped func() // action items run this when tapped; they have no content and are never selected
	rail     *NavRail
	tooltip  string
}

// NewNavRailItem returns a new item for a [NavRail].
func NewNavRailItem(icon fyne.Resource, tooltip string, content fyne.CanvasObject) *NavRailItem {
	return &NavRailItem{icon: icon, tooltip: tooltip, content: content}
}

// NewNavRailActionItem returns a new item for a [NavRail], which runs onTapped when tapped.
//
// It panics if onTapped is nil.
func NewNavRailActionItem(icon fyne.Resource, tooltip string, onTapped func()) *NavRailItem {
	if onTapped == nil {
		panic("onTapped must not be nil")
	}
	return &NavRailItem{icon: icon, tooltip: tooltip, onTapped: onTapped}
}

// NewNavRailMenuItem returns a new item for a [NavRail], which shows a pop-up menu when tapped.
//
// It panics if menu is nil.
func NewNavRailMenuItem(icon fyne.Resource, tooltip string, menu *fyne.Menu) *NavRailItem {
	if menu == nil {
		panic("menu must not be nil")
	}
	it := &NavRailItem{icon: icon, tooltip: tooltip}
	it.onTapped = func() {
		ShowPopUpMenuTrailingAbove(it.dest, menu)
	}
	return it
}

func (it *NavRailItem) isAction() bool {
	return it.onTapped != nil
}

// NavRail lets people switch between the top-level views of an app on desktop.
// It shows a vertical strip of icons with leading items at the top and trailing items at the bottom.
type NavRail struct {
	widget.BaseWidget

	body      *fyne.Container
	column    *fyne.Container
	indicator *canvas.Rectangle
	items     []*NavRailItem
	leading   *fyne.Container
	selected  *NavRailItem
	separator *widget.Separator
	strip     *fyne.Container
	trailing  *fyne.Container
}

// NewNavRail returns a new navigation rail. The first leading non-action item is selected initially.
//
// It panics if there is no leading non-action item or an item already belongs to another rail.
func NewNavRail(leading []*NavRailItem, trailing ...*NavRailItem) *NavRail {
	first := slices.IndexFunc(leading, func(it *NavRailItem) bool {
		return !it.isAction()
	})
	if first == -1 {
		panic("must define at least one leading non-action item")
	}
	gap := navRailItemGapPaddings * theme.Padding()
	w := &NavRail{
		body:      container.NewStack(),
		leading:   container.New(layout.NewCustomPaddedVBoxLayout(gap)),
		trailing:  container.New(layout.NewCustomPaddedVBoxLayout(gap)),
		indicator: canvas.NewRectangle(color.Transparent),
		separator: widget.NewSeparator(),
	}
	w.column = container.NewBorder(w.leading, w.trailing, nil, nil)
	// no padding, so the indicator touches the hover background
	w.strip = container.New(layout.NewCustomPaddedHBoxLayout(0), w.column, w.separator)
	w.ExtendBaseWidget(w)
	add := func(c *fyne.Container, it *NavRailItem) {
		if it.rail != nil {
			panic("item already belongs to a rail")
		}
		it.rail = w
		it.dest = newRailDestination(it.icon, it.tooltip, func() {
			if it.isAction() {
				it.onTapped()
				return
			}
			w.Select(it)
		})
		c.Add(it.dest)
		w.items = append(w.items, it)
		if !it.isAction() {
			it.content.Hide()
			w.body.Add(it.content)
		}
	}
	for _, it := range leading {
		add(w.leading, it)
	}
	for _, it := range trailing {
		add(w.trailing, it)
	}
	w.selectItem(leading[first])
	return w
}

// Select switches to an item. Does nothing when the item is disabled or an action item.
func (w *NavRail) Select(it *NavRailItem) {
	if !w.owns(it) || it.isAction() || it.dest.Disabled() {
		return
	}
	if it == w.selected {
		if it.OnSelectedAgain != nil {
			it.OnSelectedAgain()
		}
		return
	}
	w.selectItem(it)
}

// Selected returns the currently selected item.
func (w *NavRail) Selected() *NavRailItem {
	return w.selected
}

// EnableItem enables an item.
func (w *NavRail) EnableItem(it *NavRailItem) {
	if !w.owns(it) {
		return
	}
	it.dest.Enable()
}

// DisableItem disables an item. Disabled items can not be selected.
// When the selected item is disabled, the rail switches to the first enabled non-action item.
func (w *NavRail) DisableItem(it *NavRailItem) {
	if !w.owns(it) {
		return
	}
	it.dest.Disable()
	if it != w.selected {
		return
	}
	for _, x := range w.items {
		if !x.isAction() && !x.dest.Disabled() {
			w.selectItem(x)
			return
		}
	}
}

// ItemEnabled reports whether an item is enabled.
func (w *NavRail) ItemEnabled(it *NavRailItem) bool {
	return w.owns(it) && !it.dest.Disabled()
}

func (w *NavRail) owns(it *NavRailItem) bool {
	return it != nil && it.rail == w
}

func (w *NavRail) selectItem(it *NavRailItem) {
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

func (w *NavRail) Refresh() {
	w.leading.Refresh()
	w.trailing.Refresh()
	w.BaseWidget.Refresh()
}

// updateIndicator places the indicator on the separator next to the selected item.
func (w *NavRail) updateIndicator() {
	dest := w.selected.dest
	parent := w.leading
	if slices.Contains(w.trailing.Objects, fyne.CanvasObject(dest)) {
		parent = w.trailing
	}
	x := w.strip.Position().X + w.separator.Position().X
	y := w.strip.Position().Y + w.column.Position().Y + parent.Position().Y + dest.Position().Y
	w.indicator.Move(fyne.NewPos(x, y))
	w.indicator.Resize(fyne.NewSize(w.Theme().Size(theme.SizeNameSeparatorThickness), dest.Size().Height))
	w.indicator.Refresh()
}

func (w *NavRail) CreateRenderer() fyne.WidgetRenderer {
	r := &navRailRenderer{
		content: container.NewBorder(nil, nil, w.strip, nil, w.body),
		w:       w,
	}
	r.updateColors()
	return r
}

type navRailRenderer struct {
	content *fyne.Container
	w       *NavRail
}

func (r *navRailRenderer) Destroy() {}

func (r *navRailRenderer) Layout(size fyne.Size) {
	r.content.Resize(size)
	r.w.updateIndicator()
}

func (r *navRailRenderer) MinSize() fyne.Size {
	return r.content.MinSize()
}

func (r *navRailRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.content, r.w.indicator}
}

func (r *navRailRenderer) Refresh() {
	r.updateColors()
	r.w.updateIndicator()
	r.content.Refresh()
}

func (r *navRailRenderer) updateColors() {
	th := r.w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	r.w.indicator.FillColor = th.Color(theme.ColorNamePrimary, v)
	r.w.indicator.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
}
