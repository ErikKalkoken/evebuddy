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

// FilterOption is an option for [FilterChipCompact].
// Options can be created on any goroutine.
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
// Empty choice strings are ignored.
func NewFilterOptionMultiChoice(name string, choices []string) FilterOption {
	choices2 := xslices.Deduplicate(choices) // also copies
	choices2 = slices.DeleteFunc(choices2, func(x string) bool {
		return x == ""
	})
	slices.Sort(choices2)
	return FilterOption{
		kind:    optionKindMultiChoice,
		name:    name,
		choices: choices2,
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

	// OnChanged is a callback that is called when the selection changed,
	// either by the user or through SetSelected or Reset.
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
	w.options = normalizeOptions(options)
	w.updateSelectedFromOptions()
	w.icon = widget.NewIcon(w.iconResource)
	w.clearItem = fyne.NewMenuItem(w.resetText, func() {
		w.Reset()
	})
	w.clearItem.Icon = theme.DeleteIcon()
	w.ExtendBaseWidget(w)
	w.setMenu()
	return w
}

// IsOn reports whether the filter is active.
func (w *FilterChipCompact) IsOn() bool {
	return w.isOn
}

func (w *FilterChipCompact) updateOn() {
	var isOn bool
	for _, v := range w.selected {
		if v != "" {
			isOn = true
			break
		}
	}
	w.isOn = isOn
}

// Reset resets the selection.
func (w *FilterChipCompact) Reset() {
	if !w.isOn {
		return
	}
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

// SetOptions sets new filter options.
//
// The order of filter options is preserved.
// A selected choice is kept even when the option's new choices no longer contain it.
// Does not call OnChanged, even when a selected option is removed.
func (w *FilterChipCompact) SetOptions(options ...FilterOption) {
	w.options = normalizeOptions(options)
	w.updateSelectedFromOptions()
	w.updateOn()
	w.setMenu()
	w.Refresh()
}

// normalizeOptions removes undefined and duplicate options
// and separators which are leading, trailing or consecutive.
func normalizeOptions(options []FilterOption) []FilterOption {
	var options2 []FilterOption
	names := make(map[string]bool)
	for _, o := range options {
		if o.kind == optionKindUndefined {
			continue // e.g. zero-value FilterOption{}
		}
		if o.kind == optionKindSeparator {
			if len(options2) == 0 || options2[len(options2)-1].kind == optionKindSeparator {
				continue
			}
			options2 = append(options2, o)
			continue
		}
		if names[o.name] {
			continue // ignoring duplicate option
		}
		names[o.name] = true
		options2 = append(options2, o)
	}
	if n := len(options2); n > 0 && options2[n-1].kind == optionKindSeparator {
		options2 = options2[:n-1]
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

// SetSelected sets the selection.
//
// Invalid option names are ignored.
func (w *FilterChipCompact) SetSelected(selected map[string]string) {
	selected2 := sanitizeSelected(w.options, selected)
	if maps.Equal(w.selected, selected2) {
		return
	}
	w.selected = selected2
	w.setMenu()
	w.processChanged()
}

// sanitizeSelected returns a new selection with an entry for every option.
// Unknown options are dropped and invalid choices are reset.
func sanitizeSelected(options []FilterOption, selected map[string]string) map[string]string {
	selected2 := make(map[string]string)
	for _, o := range options {
		if o.kind == optionKindSeparator {
			continue
		}
		choice := selected[o.name]
		if !slices.Contains(o.choices, choice) {
			choice = ""
		}
		selected2[o.name] = choice
	}
	return selected2
}

func (w *FilterChipCompact) setMenu() {
	if len(w.options) == 0 {
		w.menu.Items = nil
		return
	}
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
			}

		case optionKindMultiChoice:
			var items2 []*fyne.MenuItem
			choices := o.choices

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
	w.clearItem.Disabled = !w.isOn
	items1 = append(items1, w.clearItem)

	w.menu.Items = items1
}

func (w *FilterChipCompact) processChanged() {
	w.updateOn()
	w.clearItem.Disabled = !w.isOn
	w.Refresh()
	if w.OnChanged != nil {
		w.OnChanged(w.Selected())
	}
}

func (w *FilterChipCompact) CreateRenderer() fyne.WidgetRenderer {
	w.updateStyling()
	p := w.Theme().Size(theme.SizeNamePadding)
	return widget.NewSimpleRenderer(
		container.NewStack(
			w.background,
			container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 2*p, 2*p), w.icon),
		),
	)
}

func (w *FilterChipCompact) Refresh() {
	w.updateStyling()
	w.background.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *FilterChipCompact) updateStyling() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if w.isOn {
		if w.disabled {
			w.icon.SetResource(theme.NewDisabledResource(w.iconResource))
			w.background.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.background.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.icon.SetResource(w.iconResource)
			w.background.FillColor = th.Color(theme.ColorNameSelection, v)
			w.background.StrokeColor = th.Color(theme.ColorNameSelection, v)
		}
	} else {
		if w.disabled {
			w.icon.SetResource(theme.NewDisabledResource(w.iconResource))
		} else {
			w.icon.SetResource(w.iconResource)
		}
		w.background.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
		w.background.FillColor = color.Transparent
	}

	if w.focused && !w.disabled {
		w.background.StrokeColor = th.Color(theme.ColorNameFocus, v)
		w.background.StrokeWidth = th.Size(theme.SizeNameInputBorder) * 2
	} else {
		w.background.StrokeWidth = th.Size(theme.SizeNameInputBorder)
	}
	w.background.CornerRadius = th.Size(theme.SizeNameButtonRadius)
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
	if w.hovered && !w.disabled {
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
	if len(w.options) == 0 {
		return
	}
	ShowPopUpMenuBelowLeading(w, w.menu)
}
