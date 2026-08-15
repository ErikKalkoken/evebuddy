package xwidget

import (
	"fmt"
	"image/color"
	"maps"
	"slices"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// TODO: Add API feature to enable/disable options

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

// NewFilterOptionToogle creates a toogle option for [FilterChipCompact].
func NewFilterOptionToogle(name string) FilterOption {
	return FilterOption{
		kind:    optionKindToggle,
		name:    name,
		choices: []string{name},
	}
}

// NewFilterOptionMultiChoice creates a multi-choice option for [FilterChipCompact].
// Choices are sorted alphabetically and deduplicated.
// Empty choice strings will be ignored.
func NewFilterOptionMultiChoice(name string, choices []string) FilterOption {
	return FilterOption{
		kind:    optionKindMultiChoice,
		name:    name,
		choices: choices,
	}
}

// NewFilterOptionSeparator creates a separator for [FilterChipCompact].
func NewFilterOptionSeparator() FilterOption {
	return FilterOption{kind: optionKindSeparator}
}

// FilterChipCompact represents a filter chip widget that allows the user to select
// and de-select multiple options and has a compact design.
type FilterChipCompact struct {
	widget.BaseWidget

	// OnChanged is a callback that is called when the selection state changed.
	// It passes the current selection.
	OnChanged func(selected map[string]string)

	background           *canvas.Rectangle
	blankResource        fyne.Resource
	clearItem            *fyne.MenuItem
	disabled             bool
	focused              bool
	hovered              bool
	icon                 *widget.Icon
	iconResource         fyne.Resource
	isOn                 bool
	itemSelectedResource fyne.Resource
	menu                 *fyne.Menu
	options              []FilterOption
	resetText            string
	selected             map[string]string
}

var _ desktop.Hoverable = (*FilterChipCompact)(nil)
var _ fyne.Disableable = (*FilterChipCompact)(nil)
var _ fyne.Focusable = (*FilterChipCompact)(nil)
var _ fyne.Tappable = (*FilterChipCompact)(nil)
var _ fyne.Widget = (*FilterChipCompact)(nil)

// NewFilterChipCompact creates and returns a new [FilterChipCompact].
func NewFilterChipCompact(options []FilterOption, changed func(map[string]string)) *FilterChipCompact {
	w := &FilterChipCompact{
		background:           canvas.NewRectangle(color.Transparent),
		blankResource:        iconBlankSvg,
		iconResource:         theme.NewThemedResource(iconFilterVariantSvg),
		itemSelectedResource: theme.ConfirmIcon(),
		menu:                 fyne.NewMenu(""),
		OnChanged:            changed,
		resetText:            "Clear",
		selected:             make(map[string]string),
	}
	w.options = removeDuplicateOptions(options)
	w.icon = widget.NewIcon(w.iconResource)
	w.background.CornerRadius = theme.Size(theme.SizeNameButtonRadius)
	w.clearItem = fyne.NewMenuItem(w.resetText, func() {
		w.Reset()
	})
	w.clearItem.Icon = theme.DeleteIcon()
	w.ExtendBaseWidget(w)
	if len(w.options) > 0 {
		w.setMenu()
	}
	return w
}

// SetOptions sets new filter options.
//
// The order of filter options is preserved.
func (w *FilterChipCompact) SetOptions(options ...FilterOption) {
	w.options = removeDuplicateOptions(options)
	w.updateSelectedFromOptions()
	w.setMenu()
}

func removeDuplicateOptions(options []FilterOption) []FilterOption {
	var options2 []FilterOption
	names := make(map[string]bool)
	for _, o := range options {
		if o.kind == optionKindSeparator {
			continue
		}
		if names[o.name] {
			continue // ignoring duplicate option
		}
		names[o.name] = true
		options2 = append(options2, o)
	}
	return options2
}

func (w *FilterChipCompact) updateSelectedFromOptions() {
	names := make(map[string]bool)
	for _, o := range w.options {
		if o.kind != optionKindSeparator {
			names[o.name] = true
		}
	}
	// remove outdated options
	for name := range w.selected {
		if !names[name] {
			delete(w.selected, name)
		}
	}
	// initialize new options
	for name := range names {
		if _, found := w.selected[name]; !found {
			w.selected[name] = ""
		}
	}
}

func (w *FilterChipCompact) SetSelected(selected map[string]string) {
	w.selected = sanitizeSelected(w.options, selected)
	w.setMenu()
}

func sanitizeSelected(options []FilterOption, selected map[string]string) map[string]string {
	optionsMap := make(map[string]FilterOption)
	for _, o := range options {
		if o.kind != optionKindSeparator {
			optionsMap[o.name] = o
		}
	}
	selected2 := maps.Clone(selected)
	for name, choice := range selected2 {
		o, ok := optionsMap[name]
		if !ok {
			delete(selected2, name)
			continue
		}
		if !slices.Contains(o.choices, choice) {
			selected2[name] = ""
		}
	}
	return selected2
}

func (w *FilterChipCompact) setMenu() {
	var items1 []*fyne.MenuItem

	for _, o := range w.options {
		if o.kind == optionKindSeparator {
			items1 = append(items1, fyne.NewMenuItemSeparator())
			continue
		}

		it1 := fyne.NewMenuItem(o.name, nil)

		switch o.kind {
		case optionKindToggle:
			if w.selected[o.name] != "" {
				it1.Icon = w.itemSelectedResource
			} else {
				it1.Icon = w.blankResource
			}
			it1.Action = func() {
				if w.selected[o.name] == "" {
					w.selected[o.name] = o.name
					it1.Icon = w.itemSelectedResource
				} else {
					w.selected[o.name] = ""
					it1.Icon = w.blankResource
				}
				w.processChanged()
				w.menu.Refresh()
			}

		case optionKindMultiChoice:
			var items2 []*fyne.MenuItem
			choices := xslices.Deduplicate(o.choices)
			choices = slices.DeleteFunc(choices, func(x string) bool {
				return x == ""
			})
			slices.Sort(choices)

			makeLabel := func(name string) string {
				selected := w.selected[name]
				var count string
				if x := len(choices); x > 99 {
					count = "99+"
				} else {
					count = strconv.Itoa(x)
				}
				title := fmt.Sprintf("%s (%s)", name, count)
				if selected == "" {
					return title
				}
				return fmt.Sprintf("%s: %s", title, selected)
			}
			if len(choices) > 0 {
				it1.Disabled = false
				for _, c := range choices {
					it2 := fyne.NewMenuItem(c, nil)
					if w.selected[o.name] == c {
						it2.Icon = w.itemSelectedResource
					} else {
						it2.Icon = w.blankResource
					}
					it2.Action = func() {
						if w.selected[o.name] == c {
							w.selected[o.name] = ""
						} else {
							w.selected[o.name] = c
						}
						it1.Label = makeLabel(o.name)
						for _, it := range items2 {
							if it.Label == w.selected[o.name] {
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
			} else {
				it1.Disabled = true
			}
			it1.Icon = w.blankResource
			it1.Label = makeLabel(o.name)
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
	w.menu.Refresh()
}

func (w *FilterChipCompact) processChanged() {
	w.Refresh()
	if w.OnChanged != nil {
		w.OnChanged(w.Selected())
	}
}

// Reset resets all options.
func (w *FilterChipCompact) Reset() {
	for name := range w.selected {
		w.selected[name] = ""
	}
	w.setMenu()
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
		if w.disabled {
			w.icon.SetResource(theme.NewDisabledResource(w.iconResource))
			w.background.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.background.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.icon.SetResource(w.iconResource)
			w.background.FillColor = th.Color(theme.ColorNameSelection, v)
			w.background.StrokeColor = th.Color(theme.ColorNameSelection, v)
		}
		w.clearItem.Disabled = false
	} else {
		if w.disabled {
			w.icon.SetResource(theme.NewDisabledResource(w.iconResource))
		} else {
			w.icon.SetResource(w.iconResource)
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

func (w *FilterChipCompact) Disabled() bool {
	return w.disabled
}

func (w *FilterChipCompact) Disable() {
	if w.disabled {
		return
	}
	w.disabled = true
	w.Refresh()
}

func (w *FilterChipCompact) Enable() {
	if !w.disabled {
		return
	}
	w.disabled = false
	w.Refresh()
}

func (w *FilterChipCompact) Tapped(pe *fyne.PointEvent) {
	if w.disabled {
		return
	}
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
	if w.disabled {
		return
	}
	oldHovered := w.hovered
	size := w.Size()
	w.hovered = size.IsZero() ||
		(me.Position.X <= size.Width && me.Position.Y <= size.Height)

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
	if w.disabled {
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
	if w.disabled {
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
