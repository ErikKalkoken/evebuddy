package widget

import (
	"image/color"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ilayout "github.com/ErikKalkoken/evebuddy/internal/layout"
)

// FilterChip represents a simple filter chip widget which only has two states: on or off.
// Filter chips use tags or descriptive words to filter content.
type FilterChip struct {
	widget.DisableableWidget

	Text string
	On   bool

	// OnChanged is called when the state changed
	OnChanged func(on bool)

	bg         *canvas.Rectangle
	focused    bool
	hovered    bool
	icon       *widget.Icon
	iconPadded *fyne.Container
	label      *widget.Label
	minSize    fyne.Size // cached for hover/top pos calcs
}

var _ desktop.Hoverable = (*FilterChip)(nil)
var _ fyne.Disableable = (*FilterChip)(nil)
var _ fyne.Focusable = (*FilterChip)(nil)
var _ fyne.Tappable = (*FilterChip)(nil)
var _ fyne.Widget = (*FilterChip)(nil)

// NewFilterChip returns a new [FilterChip] object.
func NewFilterChip(text string, changed func(on bool)) *FilterChip {
	w := &FilterChip{
		label:     widget.NewLabel(text),
		OnChanged: changed,
		Text:      text,
	}
	w.ExtendBaseWidget(w)
	p := theme.Padding()
	w.icon = widget.NewIcon(theme.ConfirmIcon())
	w.iconPadded = container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), w.icon)
	w.iconPadded.Hide()
	w.bg = canvas.NewRectangle(color.Transparent)
	w.bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder) * 2
	w.bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	return w
}

// SetState sets the state.
func (w *FilterChip) SetState(v bool) {
	if w.On == v {
		return
	}
	w.On = v
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
	if w.On {
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
	w.SetState(!w.On)
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
		w.SetState(!w.On)
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

	OnChanged func(selected []string)

	Options  []string // readonly TODO: Enable setting options
	Selected []string // readonly after first render

	chips    []*FilterChip
	options  []string
	selected []string
}

// NewFilterChipGroup returns a new [FilterChipGroup].
func NewFilterChipGroup(options []string, changed func([]string)) *FilterChipGroup {
	optionsCleaned := slices.DeleteFunc(deduplicateSlice(options), func(v string) bool {
		return v == ""
	})
	w := &FilterChipGroup{
		chips:     make([]*FilterChip, 0),
		OnChanged: changed,
		options:   optionsCleaned,
		Options:   slices.Clone(optionsCleaned),
		Selected:  make([]string, 0),
	}
	w.ExtendBaseWidget(w)
	isSelected := make(map[string]bool)
	for _, v := range w.options {
		w.chips = append(w.chips, NewFilterChip(v, func(on bool) {
			if on {
				isSelected[v] = true
			} else {
				isSelected[v] = false
			}
			w.selected = make([]string, 0)
			for _, o := range w.options {
				if isSelected[o] {
					w.selected = append(w.selected, o)
				}
			}
			w.Selected = slices.Clone(w.selected)
			if w.OnChanged != nil {
				w.OnChanged(w.Selected)
			}
		}))
	}
	return w
}

func (w *FilterChipGroup) CreateRenderer() fyne.WidgetRenderer {
	w.SetSelected(w.Selected)
	p := w.Theme().Size(theme.SizeNamePadding)
	box := container.New(ilayout.NewRowWrapLayoutWithCustomPadding(2*p, 2*p))
	for _, c := range w.chips {
		box.Add(c)
	}
	return widget.NewSimpleRenderer(container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 0, 0), box))
}

// SetSelected updates the selected options.
// Invalid elements including empty strings will be ignored.
func (w *FilterChipGroup) SetSelected(s []string) {
	isValid := make(map[string]bool)
	for _, v := range w.options {
		isValid[v] = true
	}
	isSelected := make(map[string]bool)
	for _, v := range s {
		if !isValid[v] {
			continue
		}
		isSelected[v] = true
	}
	for i, v := range w.options {
		w.chips[i].SetState(isSelected[v])
	}
}

// FilterChipSelect represents a filter chip widget that shows a drop down with options.
//
// Differences to the standard Select widget:
// Shows more clearly whether a filter is enabled and does not truncate the option names.
type FilterChipSelect struct {
	widget.DisableableWidget

	// The label shown for clearing a selection.
	ClearLabel string

	OnChanged func(selected string)
	Options   []string

	// The Placeholder is shown when nothing is selected.
	// To create a filter which is always selected leave placeholder empty
	// and select an initial option.
	// When in always selected state, the options are deduplicated, but not sorted.
	Placeholder string

	Selected string

	bg         *canvas.Rectangle
	focused    bool
	hovered    bool
	icon       *widget.Icon
	iconPadded *fyne.Container
	label      *widget.Label
	minSize    fyne.Size // cached for hover/top pos calcs

	isMobile bool
	window   fyne.Window
}

var _ desktop.Hoverable = (*FilterChipSelect)(nil)
var _ fyne.Disableable = (*FilterChipSelect)(nil)
var _ fyne.Focusable = (*FilterChipSelect)(nil)
var _ fyne.Tappable = (*FilterChipSelect)(nil)
var _ fyne.Widget = (*FilterChipSelect)(nil)

// NewFilterChipSelect returns a new [FilterChipSelect] widget with a drop down menu.
func NewFilterChipSelect(placeholder string, options []string, changed func(selected string)) *FilterChipSelect {
	w := newFilterChipSelect(placeholder, options, changed, nil)
	return w
}

// NewFilterChipSelectWithSearch returns a new [FilterChipSelect] widget with a search dialog.
func NewFilterChipSelectWithSearch(placeholder string, options []string, changed func(selected string), window fyne.Window) *FilterChipSelect {
	if placeholder == "" {
		// This variant must not be created without a placeholder
		placeholder = "PLACEHOLDER"
	}
	w := newFilterChipSelect(placeholder, options, changed, window)
	return w
}

func newFilterChipSelect(placeholder string, options []string, changed func(selected string), window fyne.Window) *FilterChipSelect {
	w := &FilterChipSelect{
		ClearLabel:  "Clear",
		Placeholder: placeholder,
		OnChanged:   changed,
		isMobile:    fyne.CurrentDevice().IsMobile(),
		window:      window,
	}
	w.ExtendBaseWidget(w)
	w.label = widget.NewLabel(w.Placeholder)
	w.icon = widget.NewIcon(theme.ConfirmIcon())
	p := theme.Padding()
	w.iconPadded = container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), w.icon)
	w.iconPadded.Hide()
	w.bg = canvas.NewRectangle(color.Transparent)
	w.bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder) * 2
	w.bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	w.setOptions(options)
	return w
}

// ClearSelected clears any selection.
func (w *FilterChipSelect) ClearSelected() {
	w.SetSelected("")
}

// SetSelected selects an option.
// An empty string will clear the selection.
// Invalid options will be ignored.
func (w *FilterChipSelect) SetSelected(v string) {
	if w.Selected == v {
		return
	}
	if v != "" && !slices.Contains(w.Options, v) {
		return
	}
	if v == "" && w.Placeholder == "" {
		return
	}
	w.Selected = v
	if w.OnChanged != nil {
		w.OnChanged(v)
	}
	w.Refresh()
}

// SetOptions sets the options.
// If a current selection no longer matches an option it will be cleared.
// Options are always sorted alphabetically and deduplicated.
// Empty option strings will be ignored.
func (w *FilterChipSelect) SetOptions(options []string) {
	if w.Selected != "" && !slices.Contains(options, w.Selected) {
		w.SetSelected("")
	}
	w.setOptions(options)
	w.Refresh()
}

func (w *FilterChipSelect) setOptions(options []string) {
	options = slices.DeleteFunc(options, func(s string) bool {
		return s == ""
	})
	if w.Placeholder == "" {
		w.Options = deduplicateSlice(options)
	} else {
		slices.SortFunc(options, func(a, b string) int {
			return strings.Compare(strings.ToLower(a), strings.ToLower(b))
		})
		w.Options = slices.Compact(options)
	}
}

func (w *FilterChipSelect) showInteraction() {
	if w.window == nil {
		w.showDropDownMenu()
	} else {
		w.showSearchDialog()
	}
}

func (w *FilterChipSelect) showDropDownMenu() {
	items := make([]*fyne.MenuItem, 0)
	if w.Placeholder != "" && w.Selected != "" {
		it := fyne.NewMenuItem(w.ClearLabel, func() {
			w.SetSelected("")
		})
		it.Icon = theme.DeleteIcon()
		items = append(items, it)
		items = append(items, fyne.NewMenuItemSeparator())
	}
	if len(w.Options) == 0 {
		it := fyne.NewMenuItem("No entries", nil)
		it.Disabled = true
		items = append(items, it)
	} else {
		for _, o := range w.Options {
			it := fyne.NewMenuItem(o, func() {
				w.SetSelected(o)
			})
			if w.Selected != "" {
				if o == w.Selected {
					it.Icon = theme.ConfirmIcon()
				} else {
					it.Icon = icons.BlankSvg
				}
			}
			items = append(items, it)
		}
	}
	m := fyne.NewMenu("", items...)
	pos := fyne.NewPos(0, w.minSize.Height)
	widget.ShowPopUpMenuAtRelativePosition(m, fyne.CurrentApp().Driver().CanvasForObject(w), pos, w)
}

func (w *FilterChipSelect) showSearchDialog() {
	itemsFiltered := slices.Clone(w.Options)
	var d dialog.Dialog
	list := widget.NewList(
		func() int {
			return len(itemsFiltered)
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(icons.BlankSvg)
			if w.Selected == "" {
				icon.Hide()
			} else {
				icon.Show()
			}
			return container.NewBorder(
				nil,
				nil,
				icon,
				nil,
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(itemsFiltered) {
				return
			}
			s := itemsFiltered[id]
			box := co.(*fyne.Container).Objects
			box[0].(*widget.Label).SetText(s)
			if w.Selected == "" {
				return
			}
			icon := box[1].(*widget.Icon)
			if s == w.Selected {
				icon.SetResource(theme.ConfirmIcon())
			} else {
				icon.SetResource(iconBlankSvg)
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(itemsFiltered) {
			return
		}
		w.SetSelected(itemsFiltered[id])
		d.Hide()
	}
	list.HideSeparators = true
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = NewIconButton(theme.CancelIcon(), func() {
		entry.SetText("")
	})
	entry.OnChanged = func(search string) {
		if len(search) < 2 {
			itemsFiltered = slices.Clone(w.Options)
			list.Refresh()
			return
		}
		itemsFiltered = make([]string, 0)
		search2 := strings.ToLower(search)
		for _, s := range w.Options {
			if strings.Contains(strings.ToLower(s), search2) {
				itemsFiltered = append(itemsFiltered, s)
			}
		}
		list.Refresh()
	}
	clear := widget.NewButton("Clear", func() {
		w.SetSelected("")
		d.Hide()
	})
	if w.Selected != "" {
		entry.Disable()
		clear.Show()
	} else {
		clear.Hide()
	}
	empty := widget.NewLabel("No entries")
	empty.Importance = widget.LowImportance
	if len(w.Options) == 0 {
		empty.Show()
		entry.Disable()
		clear.Hide()
	} else {
		empty.Hide()
	}
	c := container.NewBorder(
		container.NewBorder(
			nil,
			clear,
			nil,
			widget.NewButton("Cancel", func() {
				d.Hide()
			}),
			entry,
		),
		empty,
		nil,
		nil,
		list,
	)
	d = dialog.NewCustomWithoutButtons("Filter by "+w.Placeholder, c, w.window)
	_, s := w.window.Canvas().InteractiveArea()
	if w.isMobile {
		d.Resize(fyne.NewSize(s.Width, s.Height))
	} else {
		d.Resize(fyne.NewSize(600, max(400, s.Height*0.8)))
	}
	d.Show()
	w.window.Canvas().Focus(entry)
}

func (w *FilterChipSelect) updateState() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if w.Disabled() {
		w.label.Importance = widget.LowImportance
		w.icon.Resource = theme.NewDisabledResource(theme.ConfirmIcon())
		w.bg.StrokeColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		w.label.Importance = widget.MediumImportance
		w.icon.Resource = theme.ConfirmIcon()
		w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	}
	if w.Selected != "" {
		w.label.Text = w.Selected
		w.iconPadded.Show()
		if w.Disabled() {
			w.bg.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.bg.FillColor = th.Color(theme.ColorNameButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameButton, v)
		}
	} else {
		w.label.Text = w.Placeholder
		w.iconPadded.Hide()
		w.bg.FillColor = color.Transparent
	}

	if w.focused {
		w.bg.StrokeColor = th.Color(theme.ColorNameFocus, v)
	}
}

func (w *FilterChipSelect) Refresh() {
	w.updateState()
	w.bg.Refresh()
	w.label.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *FilterChipSelect) MinSize() fyne.Size {
	w.ExtendBaseWidget(w)
	w.minSize = w.BaseWidget.MinSize()
	return w.minSize
}

func (w *FilterChipSelect) Tapped(pe *fyne.PointEvent) {
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
	w.showInteraction()
}

func (w *FilterChipSelect) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *FilterChipSelect) MouseIn(me *desktop.MouseEvent) {
	w.MouseMoved(me)
}

func (w *FilterChipSelect) MouseMoved(me *desktop.MouseEvent) {
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

func (w *FilterChipSelect) MouseOut() {
	if w.hovered {
		w.hovered = false
		w.Refresh()
	}
}

// FocusGained is called when the Check has been given focus.
func (w *FilterChipSelect) FocusGained() {
	if w.Disabled() {
		return
	}
	w.focused = true
	w.Refresh()
}

// FocusLost is called when the Check has had focus removed.
func (w *FilterChipSelect) FocusLost() {
	w.focused = false
	w.Refresh()
}

// TypedRune receives text input events when the Check is focused.
func (w *FilterChipSelect) TypedRune(r rune) {
	if w.Disabled() {
		return
	}
	if r == ' ' {
		w.showInteraction()
	}
}

// TypedKey receives key input events when the Check is focused.
func (w *FilterChipSelect) TypedKey(key *fyne.KeyEvent) {}

func (w *FilterChipSelect) CreateRenderer() fyne.WidgetRenderer {
	w.updateState()
	p := theme.Padding()
	c := container.NewHBox(container.NewStack(
		w.bg,
		container.New(
			layout.NewCustomPaddedLayout(0, 0, p, p),
			container.New(layout.NewCustomPaddedHBoxLayout(0),
				layout.NewSpacer(),
				w.iconPadded,
				w.label,
				widget.NewIcon(theme.MenuDropDownIcon()),
				layout.NewSpacer(),
			),
		)))
	return widget.NewSimpleRenderer(c)
}

func deduplicateSlice[S ~[]E, E comparable](s S) []E {
	seen := make(map[E]bool)
	s2 := make([]E, 0)
	for _, v := range s {
		if seen[v] {
			continue
		}
		s2 = append(s2, v)
		seen[v] = true
	}
	return s2
}
