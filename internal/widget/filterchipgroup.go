package widget

import (
	"image/color"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ilayout "github.com/ErikKalkoken/evebuddy/internal/layout"
)

// FilterChip represents a filter chip widget
// Filter chips use tags or descriptive words to filter content.
type FilterChip struct {
	widget.DisableableWidget

	Text     string
	Selected bool

	// OnChanged is called when the state changed
	OnChanged func(selected bool)

	hovered    bool
	minSize    fyne.Size // cached for hover/top pos calcs
	iconPadded *fyne.Container
	icon       *widget.Icon
	label      *widget.Label
	bg         *canvas.Rectangle
	focused    bool
	mu         sync.RWMutex
}

var _ desktop.Hoverable = (*FilterChip)(nil)
var _ fyne.Disableable = (*FilterChip)(nil)
var _ fyne.Focusable = (*FilterChip)(nil)
var _ fyne.Tappable = (*FilterChip)(nil)
var _ fyne.Widget = (*FilterChip)(nil)

// NewFilterChip returns a new [FilterChip] object.
func NewFilterChip(text string, changed func(selected bool)) *FilterChip {
	w := &FilterChip{
		label:     widget.NewLabel(text),
		OnChanged: changed,
		Text:      text,
	}
	p := theme.Padding()
	w.icon = widget.NewIcon(theme.ConfirmIcon())
	w.iconPadded = container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), w.icon)
	w.iconPadded.Hide()
	w.bg = canvas.NewRectangle(color.Transparent)
	w.bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder) * 2
	w.bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	w.ExtendBaseWidget(w)
	return w
}

// SetSelected sets the state.
func (w *FilterChip) SetSelected(v bool) {
	w.mu.Lock()
	if w.Selected == v {
		w.mu.Unlock()
		return
	}
	w.Selected = v
	w.mu.Unlock()
	if w.OnChanged != nil {
		w.OnChanged(v)
	}
	w.Refresh()
}

func (w *FilterChip) Refresh() {
	w.updateState()
	w.bg.Refresh()
	w.label.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *FilterChip) updateState() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.mu.Lock()
	defer w.mu.Unlock()

	w.label.Text = w.Text

	if w.Disabled() {
		w.label.Importance = widget.LowImportance
		w.icon.Resource = theme.NewDisabledResource(theme.ConfirmIcon())
		w.bg.StrokeColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		w.label.Importance = widget.MediumImportance
		w.icon.Resource = theme.ConfirmIcon()
		w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	}
	if w.Selected {
		w.iconPadded.Show()
		if w.Disabled() {
			w.bg.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.bg.FillColor = th.Color(theme.ColorNameButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameButton, v)
		}
	} else {
		w.iconPadded.Hide()
		w.bg.FillColor = color.Transparent
	}

	if w.focused {
		w.bg.StrokeColor = th.Color(theme.ColorNameFocus, v)
	}
}

func (w *FilterChip) MinSize() fyne.Size {
	w.ExtendBaseWidget(w)
	w.minSize = w.BaseWidget.MinSize()
	return w.minSize
}

func (w *FilterChip) Tapped(pe *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	if !w.minSize.IsZero() &&
		(pe.Position.X > w.minSize.Width || pe.Position.Y > w.minSize.Height) {
		// tapped outside
		return
	}
	// if !w.focused {
	// 	if !fyne.CurrentDevice().IsMobile() {
	// 		if c := fyne.CurrentApp().Driver().CanvasForObject(w); c != nil {
	// 			c.Focus(w)
	// 		}
	// 	}
	// }
	w.SetSelected(!w.Selected)
}

func (w *FilterChip) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *FilterChip) MouseIn(me *desktop.MouseEvent) {
	w.MouseMoved(me)
}

func (w *FilterChip) MouseMoved(me *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	oldHovered := w.hovered
	w.hovered = w.minSize.IsZero() ||
		(me.Position.X <= w.minSize.Width && me.Position.Y <= w.minSize.Height)

	if oldHovered != w.hovered {
		w.Refresh()
	}
}

func (w *FilterChip) MouseOut() {
	if w.hovered {
		w.hovered = false
		w.Refresh()
	}
}

// FocusGained is called when the Check has been given focus.
func (w *FilterChip) FocusGained() {
	if w.Disabled() {
		return
	}
	w.focused = true
	w.Refresh()
}

// FocusLost is called when the Check has had focus removed.
func (w *FilterChip) FocusLost() {
	w.focused = false
	w.Refresh()
}

// TypedRune receives text input events when the Check is focused.
func (w *FilterChip) TypedRune(r rune) {
	if w.Disabled() {
		return
	}
	if r == ' ' {
		w.SetSelected(!w.Selected)
	}
}

// TypedKey receives key input events when the Check is focused.
func (w *FilterChip) TypedKey(key *fyne.KeyEvent) {}

func (w *FilterChip) CreateRenderer() fyne.WidgetRenderer {
	w.updateState()
	p := theme.Padding()
	c := container.NewHBox(container.NewStack(
		w.bg,
		container.New(
			layout.NewCustomPaddedLayout(0, 0, p, p),
			container.New(
				layout.NewCustomPaddedHBoxLayout(0),
				layout.NewSpacer(),
				w.iconPadded,
				w.label,
				layout.NewSpacer(),
			),
		)))
	return widget.NewSimpleRenderer(c)
}

// FilterCheckGround represents a group of filter chips.
// Filter chips use tags or descriptive words to filter content.
type FilterChipGroup struct {
	widget.DisableableWidget

	OnChanged func([]string)

	mu       sync.RWMutex
	Selected []string

	options []string
	chips   []*FilterChip
}

// NewFilterChipGroup returns a new [FilterChipGroup].
func NewFilterChipGroup(options []string, changed func([]string)) *FilterChipGroup {
	w := &FilterChipGroup{
		chips:     make([]*FilterChip, 0),
		OnChanged: changed,
		options:   options,
		Selected:  make([]string, 0),
	}
	w.ExtendBaseWidget(w)
	for _, o := range options {
		if o == "" {
			panic("Empty strings are not allowed as options")
		}
		w.chips = append(w.chips, NewFilterChip(o, func(selected bool) {
			w.toggleOption(o, selected)
			if w.OnChanged != nil {
				w.OnChanged(slices.Clone(w.Selected))
			}
		}))
	}
	return w
}

func (w *FilterChipGroup) toggleOption(o string, selected bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if selected {
		if slices.IndexFunc(w.Selected, func(s string) bool {
			return s == o
		}) != 0 {
			w.Selected = append(w.Selected, o)
		}
	} else {
		w.Selected = slices.DeleteFunc(w.Selected, func(s string) bool {
			return s == o
		})
	}
}

// SetSelected updates the selected options.
func (w *FilterChipGroup) SetSelected(s []string) {
	w.mu.Lock()
	w.Selected = slices.Clone(s)
	w.mu.Unlock()
	w.Refresh()

}

// Options returns the options.
func (w *FilterChipGroup) Options() []string {
	return slices.Clone(w.options)
}

func (w *FilterChipGroup) update() {
	optionMap := make(map[string]bool)
	for _, v := range w.options {
		optionMap[v] = true
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, v := range w.Selected {
		if v == "" {
			panic("Empty string in Selected")
		}
		if !optionMap[v] {
			panic("Invalid value in Selected: " + v)
		}
	}
	selected := make(map[string]bool)
	for _, v := range w.Selected {
		selected[v] = true
	}
	for i, v := range w.options {
		w.chips[i].Selected = selected[v]
	}
}

func (w *FilterChipGroup) Refresh() {
	w.update()
	for _, cf := range w.chips {
		cf.Refresh()
	}
	w.BaseWidget.Refresh()
}

func (w *FilterChipGroup) CreateRenderer() fyne.WidgetRenderer {
	w.update()
	p := w.Theme().Size(theme.SizeNamePadding)
	box := container.New(ilayout.NewRowWrapLayoutWithCustomPadding(2*p, 2*p))
	for _, c := range w.chips {
		box.Add(c)
	}
	return widget.NewSimpleRenderer(container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 0, 0), box))
}
