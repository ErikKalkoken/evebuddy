package xwidget

import (
	"fmt"
	"image/color"
	"maps"
	"slices"
	"strconv"
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
	kind      filterOptionKind
	name      string
	choices   []string
	hasSearch bool
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

// NewFilterOptionMultiChoiceWithSearch creates a multi-choice option for [FilterChipCompact],
// which lets the user pick a choice in a search dialog instead of a sub menu.
// Its menu item has a search icon.
// Use it for options with many choices.
// Choices are processed the same as for [NewFilterOptionMultiChoice].
func NewFilterOptionMultiChoiceWithSearch(name string, choices []string) FilterOption {
	o := NewFilterOptionMultiChoice(name, choices)
	o.hasSearch = true
	return o
}

// NewFilterOptionSeparator creates a separator for [FilterChipCompact].
func NewFilterOptionSeparator() FilterOption {
	return FilterOption{kind: optionKindSeparator}
}

// FilterChipCompact represents a filter chip widget that allows the user to select
// and de-select multiple options and has a compact design.
type FilterChipCompact struct {
	widget.DisableableWidget

	// OnChanged is a callback that is called when the selection changed,
	// either by the user or through SetSelected or Reset.
	// It passes the current selection.
	OnChanged func(selected map[string]string)

	background           *canvas.Rectangle
	blankResource        fyne.Resource
	clearItem            *fyne.MenuItem
	focused              bool
	hovered              bool
	icon                 *widget.Icon
	iconResource         fyne.Resource
	isOn                 bool
	itemSelectedResource fyne.Resource
	menu                 *fyne.Menu
	options              []FilterOption
	resetText            string
	searchResource       fyne.Resource
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
		searchResource:       theme.SearchIcon(),
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

// ResetSilent resets the selection without calling OnChanged.
func (w *FilterChipCompact) ResetSilent() {
	if !w.isOn {
		return
	}
	for name := range w.selected {
		w.selected[name] = ""
	}
	w.updateOn()
	w.setMenu()
	w.Refresh()
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
			if len(choices) > 0 && o.hasSearch {
				it1.Action = func() {
					w.showSearch(o)
				}
			} else if len(choices) > 0 {
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
			it1.Label = makeLabel(o.name)
			if o.hasSearch {
				it1.Icon = w.searchResource
			} else {
				it1.Icon = w.blankResource
				it1.ChildMenu = fyne.NewMenu("", items2...)
			}

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

// showSearch shows a search dialog for picking a choice of option o.
func (w *FilterChipCompact) showSearch(o FilterOption) *filterSearchPopUp {
	c := fyne.CurrentApp().Driver().CanvasForObject(w)
	if c == nil {
		return nil
	}
	p := newFilterSearchPopUp(c, o.name, o.choices, w.selected[o.name], func(choice string) {
		if v, found := w.selected[o.name]; !found || v == choice {
			return // option removed or unchanged while the dialog was open
		}
		w.selected[o.name] = choice
		w.setMenu()
		w.processChanged()
	})
	p.show()
	return p
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
		if w.Disabled() {
			w.icon.SetResource(theme.NewDisabledResource(w.iconResource))
			w.background.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.background.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.icon.SetResource(w.iconResource)
			w.background.FillColor = th.Color(theme.ColorNameSelection, v)
			w.background.StrokeColor = th.Color(theme.ColorNameSelection, v)
		}
	} else {
		if w.Disabled() {
			w.icon.SetResource(theme.NewDisabledResource(w.iconResource))
		} else {
			w.icon.SetResource(w.iconResource)
		}
		w.background.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
		w.background.FillColor = color.Transparent
	}

	if w.focused && !w.Disabled() {
		w.background.StrokeColor = th.Color(theme.ColorNameFocus, v)
		w.background.StrokeWidth = th.Size(theme.SizeNameInputBorder) * 2
	} else {
		w.background.StrokeWidth = th.Size(theme.SizeNameInputBorder)
	}
	w.background.CornerRadius = th.Size(theme.SizeNameButtonRadius)
}

func (w *FilterChipCompact) Tapped(pe *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	w.showMenu()
}

func (w *FilterChipCompact) Cursor() desktop.Cursor {
	if w.hovered && !w.Disabled() {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *FilterChipCompact) MouseIn(me *desktop.MouseEvent) {
	w.hovered = true
}

func (w *FilterChipCompact) MouseMoved(me *desktop.MouseEvent) {}

func (w *FilterChipCompact) MouseOut() {
	w.hovered = false
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
	if len(w.options) == 0 {
		return
	}
	ShowPopUpMenuBelowLeading(w, w.menu)
}

// filterSearchPopUp is a modal dialog for picking a choice from many by searching.
type filterSearchPopUp struct {
	cancel   *widget.Button
	canvas   fyne.Canvas
	choices  []string
	clear    *widget.Button
	entry    *SearchEntry
	filtered []string
	list     *widget.List
	noMatch  *widget.Label
	popUp    *widget.PopUp
	selected string
}

func newFilterSearchPopUp(c fyne.Canvas, name string, choices []string, selected string, onSelected func(string)) *filterSearchPopUp {
	p := &filterSearchPopUp{
		canvas:   c,
		choices:  choices,
		filtered: choices,
		selected: selected,
	}
	pick := func(choice string) {
		p.popUp.Hide()
		onSelected(choice)
	}
	p.list = widget.NewList(
		func() int {
			return len(p.filtered)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, widget.NewIcon(iconBlankSvg), nil, widget.NewLabel(""))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(p.filtered) {
				return
			}
			s := p.filtered[id]
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(s)
			if s == p.selected {
				border[1].(*widget.Icon).SetResource(theme.ConfirmIcon())
			} else {
				border[1].(*widget.Icon).SetResource(iconBlankSvg)
			}
		},
	)
	p.list.HideSeparators = true
	p.list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(p.filtered) {
			return
		}
		choice := p.filtered[id]
		if choice == p.selected {
			choice = "" // same toggle behavior as the sub menu
		}
		pick(choice)
	}
	p.noMatch = widget.NewLabel("No matches")
	p.noMatch.Importance = widget.LowImportance
	p.noMatch.Hide()
	p.entry = NewSearchEntry("Type to start searching...", func(search string) {
		p.applySearch(search)
	})
	p.clear = widget.NewButton("Clear", func() {
		pick("")
	})
	if selected == "" {
		p.clear.Hide()
	}
	p.cancel = widget.NewButton("Cancel", func() {
		p.popUp.Hide()
	})
	title := widget.NewLabel("Filter by " + name)
	title.TextStyle.Bold = true
	title.Truncation = fyne.TextTruncateEllipsis
	content := container.NewBorder(
		container.NewVBox(title, p.entry),
		container.NewHBox(layout.NewSpacer(), p.clear, p.cancel),
		nil,
		nil,
		container.NewStack(p.list, container.NewCenter(p.noMatch)),
	)
	p.popUp = widget.NewModalPopUp(content, c)
	return p
}

// applySearch filters the choices by a case-insensitive search.
// Short searches show all choices.
func (p *filterSearchPopUp) applySearch(search string) {
	if len(search) < 2 {
		p.filtered = p.choices
	} else {
		search = strings.ToLower(search)
		p.filtered = slices.DeleteFunc(slices.Clone(p.choices), func(s string) bool {
			return !strings.Contains(strings.ToLower(s), search)
		})
	}
	if len(p.filtered) == 0 {
		p.noMatch.Show()
	} else {
		p.noMatch.Hide()
	}
	p.list.UnselectAll()
	p.list.ScrollToOffset(0)
	p.list.Refresh()
}

func (p *filterSearchPopUp) show() {
	_, s := p.canvas.InteractiveArea()
	isMobile := fyne.CurrentDevice().IsMobile()
	if !isMobile {
		s = fyne.NewSize(min(600, s.Width), min(max(400, s.Height*0.8), s.Height))
	}
	p.popUp.Resize(s)
	p.popUp.Show()
	if !isMobile {
		p.canvas.Focus(p.entry) // on mobile this would cover the list with the keyboard
	}
}
