package xwidget

import (
	"fmt"
	"image/color"
	"maps"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// TODO: Add tests

type filterOptionKind uint

const (
	optionKindUndefined filterOptionKind = iota
	optionKindMultiChoice
	optionKindSeparator
	optionKindToggle
)

type FilterOption struct {
	kind    filterOptionKind
	name    string
	choices []string
}

// NewToogleFilterOption creates a toogle option for [FilterChipCompact].
func NewToogleFilterOption(name string) FilterOption {
	return FilterOption{
		kind: optionKindToggle,
		name: name,
	}
}

// NewMultiChoiceFilterOption creates a multi-choice option for [FilterChipCompact].
// Choices are sorted alphabetically and deduplicated.
// Empty choice strings will be ignored.
func NewMultiChoiceFilterOption(name string, choices []string) FilterOption {
	return FilterOption{
		kind:    optionKindMultiChoice,
		name:    name,
		choices: choices,
	}
}

// NewSeparatorFilterOption creates a separator for [FilterChipCompact].
func NewSeparatorFilterOption() FilterOption {
	return FilterOption{kind: optionKindSeparator}
}

// FilterChipCompact represents a filter chip widget that allows the user to select
// and de-select multiple options and has a compact design.
type FilterChipCompact struct {
	widget.DisableableWidget

	// OnChanged is a callback that is called when the selection state changed.
	// It passes the current selection.
	OnChanged func(selected map[string]string)

	background           *canvas.Rectangle
	blankResource        fyne.Resource
	clearItem            *fyne.MenuItem
	focused              bool
	hovered              bool
	icon                 *widget.Icon
	iconOffResource      fyne.Resource
	iconOnResource       fyne.Resource
	isOn                 bool
	itemSelectedResource fyne.Resource
	menu                 *fyne.Menu
	minSize              fyne.Size // cached for hover/top pos calcs
	resetText            string
	selected             map[string]string
}

var _ desktop.Hoverable = (*FilterChipCompact)(nil)
var _ fyne.Disableable = (*FilterChipCompact)(nil)
var _ fyne.Focusable = (*FilterChipCompact)(nil)
var _ fyne.Tappable = (*FilterChipCompact)(nil)
var _ fyne.Widget = (*FilterChipCompact)(nil)

// NewFilterChipCompact creates and returns a new [FilterChipCompact].
func NewFilterChipCompact(changed func(map[string]string)) *FilterChipCompact {
	w := &FilterChipCompact{
		background:           canvas.NewRectangle(color.Transparent),
		blankResource:        iconBlankSvg,
		iconOffResource:      theme.NewThemedResource(iconFilterMenuOutlineSvg),
		iconOnResource:       theme.NewThemedResource(iconFilterMenuSvg),
		itemSelectedResource: theme.ConfirmIcon(),
		menu:                 fyne.NewMenu(""),
		OnChanged:            changed,
		resetText:            "Reset",
		selected:             make(map[string]string),
	}
	w.icon = widget.NewIcon(w.iconOffResource)
	w.background.CornerRadius = theme.Size(theme.SizeNameButtonRadius)
	w.clearItem = fyne.NewMenuItem(w.resetText, func() {
		w.Reset()
	})
	w.clearItem.Icon = theme.DeleteIcon()
	w.ExtendBaseWidget(w)
	return w
}

// SetOptions sets new filter options.
//
// The order of filter options is preserved.
func (w *FilterChipCompact) SetOptions(opts ...FilterOption) {
	w.removeOutdatedSelections(opts)
	opts2 := w.removeDuplicateOptions(opts)

	var items1 []*fyne.MenuItem

	for _, fo := range opts2 {
		if fo.kind == optionKindSeparator {
			items1 = append(items1, fyne.NewMenuItemSeparator())
			continue
		}

		it1 := fyne.NewMenuItem(fo.name, nil)

		switch fo.kind {
		case optionKindToggle:
			if w.selected[fo.name] != "" {
				it1.Icon = w.itemSelectedResource
			} else {
				it1.Icon = w.blankResource
			}
			it1.Action = func() {
				if w.selected[fo.name] == "" {
					w.selected[fo.name] = fo.name
					it1.Icon = w.itemSelectedResource
				} else {
					w.selected[fo.name] = ""
					it1.Icon = w.blankResource
				}
				w.processChanged()
				w.menu.Refresh()
			}

		case optionKindMultiChoice:
			it1.Icon = w.blankResource
			if w.selected[fo.name] != "" {
				it1.Label = fmt.Sprintf("%s: %s", fo.name, w.selected[fo.name])
			} else {
				it1.Label = fo.name
			}

			var items2 []*fyne.MenuItem
			choices := xslices.Deduplicate(fo.choices)
			choices = slices.DeleteFunc(choices, func(x string) bool {
				return x == ""
			})
			slices.Sort(choices)

			for _, c := range choices {
				it2 := fyne.NewMenuItem(c, nil)
				if w.selected[fo.name] == c {
					it2.Icon = w.itemSelectedResource
				} else {
					it2.Icon = w.blankResource
				}
				it2.Action = func() {
					if w.selected[fo.name] == c {
						w.selected[fo.name] = ""
						it1.Label = fo.name
					} else {
						w.selected[fo.name] = c
						it1.Label = fmt.Sprintf("%s: %s", fo.name, c)
					}
					for _, it := range items2 {
						if it.Label == w.selected[fo.name] {
							it.Icon = w.itemSelectedResource
						} else {
							it.Icon = w.blankResource
						}
					}
					w.processChanged()
					w.menu.Refresh()
				}
				items2 = append(items2, it2)
			}
			it1.ChildMenu = fyne.NewMenu("", items2...)

		default:
			panic("unreachable")
		}

		items1 = append(items1, it1)
	}

	items1 = append(items1, fyne.NewMenuItemSeparator())
	w.clearItem.Disabled = true
	items1 = append(items1, w.clearItem)

	w.menu.Items = items1
	w.Refresh()
}

func (w *FilterChipCompact) removeOutdatedSelections(opts []FilterOption) {
	optionNames := make(map[string]bool)
	for _, fo := range opts {
		if fo.kind != optionKindSeparator {
			optionNames[fo.name] = true
		}
	}
	for name := range w.selected {
		if !optionNames[name] {
			delete(w.selected, name)
		}
	}
}

func (w *FilterChipCompact) removeDuplicateOptions(opts []FilterOption) []FilterOption {
	var opts2 []FilterOption
	optionNames := make(map[string]bool)
	for _, fo := range opts {
		if fo.kind == optionKindSeparator {
			continue
		}
		if optionNames[fo.name] {
			continue // ignoring duplicate option
		}
		optionNames[fo.name] = true
		opts2 = append(opts2, fo)
	}
	return opts2
}

func (w *FilterChipCompact) processChanged() {
	w.Refresh()
	if w.OnChanged != nil {
		w.OnChanged(w.Selected())
	}
}

// Reset resets all selected options.
func (w *FilterChipCompact) Reset() {
	for _, it1 := range w.menu.Items {
		if it1.Label == w.resetText {
			continue
		}
		it1.Icon = w.blankResource

		if idx := strings.Index(it1.Label, ": "); idx != -1 {
			it1.Label = it1.Label[:idx]
		}

		if m := it1.ChildMenu; m != nil {
			for _, it2 := range m.Items {
				it2.Icon = w.blankResource
			}
		}
	}
	for name := range w.selected {
		w.selected[name] = ""
	}
	w.menu.Refresh()
	w.processChanged()
}

// Selected returns the current selection.
//
// The selection is a map of option names and choices.
// A toggle option has the option name as choice when selected.
// A multi choice option has the selected choice when selected.
// A blank choice means the option is not selected.
func (w *FilterChipCompact) Selected() map[string]string {
	return maps.Clone(w.selected)
}

func (w *FilterChipCompact) CreateRenderer() fyne.WidgetRenderer {
	w.updateState()
	p := theme.Padding()
	return widget.NewSimpleRenderer(
		container.NewStack(
			w.background,
			container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 2*p, 2*p), w.icon),
		),
	)
}

func (w *FilterChipCompact) Refresh() {
	w.updateState()
	w.background.Refresh()
	w.icon.Refresh()
	w.menu.Refresh()
	w.BaseWidget.Refresh()
}

func (w *FilterChipCompact) updateState() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	var isOn bool
	for _, v := range w.selected {
		if v != "" {
			isOn = true
		}
	}
	w.isOn = isOn

	if isOn {
		if w.Disabled() {
			w.icon.SetResource(theme.NewDisabledResource(w.iconOnResource))
			w.background.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.background.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.icon.SetResource(w.iconOnResource)
			w.background.FillColor = th.Color(theme.ColorNameSelection, v)
			w.background.StrokeColor = th.Color(theme.ColorNameSelection, v)
		}
		w.clearItem.Disabled = false
	} else {
		if w.Disabled() {
			w.icon.SetResource(theme.NewDisabledResource(w.iconOffResource))
		} else {
			w.icon.SetResource(w.iconOffResource)
		}
		w.background.StrokeColor = theme.Color(theme.ColorNameInputBorder)
		w.background.FillColor = color.Transparent
		w.clearItem.Disabled = true
	}

	if w.focused {
		w.background.StrokeColor = th.Color(theme.ColorNameFocus, v)
		w.background.StrokeWidth = theme.Size(theme.SizeNameInputBorder) * 2
	} else {
		w.background.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	}
}

func (w *FilterChipCompact) MinSize() fyne.Size {
	w.ExtendBaseWidget(w)
	w.minSize = w.BaseWidget.MinSize()
	return w.minSize
}

func (w *FilterChipCompact) Tapped(pe *fyne.PointEvent) {
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
	w.showMenu()
}

func (w *FilterChipCompact) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *FilterChipCompact) MouseIn(me *desktop.MouseEvent) {
	w.MouseMoved(me)
}

func (w *FilterChipCompact) MouseMoved(me *desktop.MouseEvent) {
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

func (w *FilterChipCompact) MouseOut() {
	if w.hovered {
		w.hovered = false
		w.Refresh()
	}
}

// FocusGained is called when the Check has been given focus.
func (w *FilterChipCompact) FocusGained() {
	if w.Disabled() {
		return
	}
	w.focused = true
	w.Refresh()
}

// FocusLost is called when the Check has had focus removed.
func (w *FilterChipCompact) FocusLost() {
	w.focused = false
	w.Refresh()
}

// TypedRune receives text input events when the Check is focused.
func (w *FilterChipCompact) TypedRune(r rune) {
	if w.Disabled() {
		return
	}
	if r == ' ' {
		w.showMenu()
	}
}

// TypedKey receives key input events when the Check is focused.
func (w *FilterChipCompact) TypedKey(key *fyne.KeyEvent) {}

func (w *FilterChipCompact) showMenu() {
	ShowPopUpMenuBelowLeading(w, w.menu)
}
