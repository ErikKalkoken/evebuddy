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
)

// FilterChipSelect represents a filter chip widget that allows users to select from a list of options.
//
// There are two variants:
//   - Option can be selected from a drop down list (better for few options)
//   - Option can be selected with a search dialog (better for many options)
type FilterChipSelect struct {
	widget.DisableableWidget

	// The label shown for clearing a selection.
	ClearLabel string

	OnChanged func(selected string)
	Options   []string

	// The Text is shown when nothing is selected.
	// To create a filter which is always selected leave Text empty
	// and select an initial option.
	// When in always selected state, the options are deduplicated, but not sorted.
	Text string

	// The currently selected option or empty when nothing is selected.
	// This can also be used to set an initial option.
	Selected string

	// Whether to disable sorting of options.
	SortDisabled bool

	bg                   *canvas.Rectangle
	focused              bool
	hovered              bool
	iconOn               *widget.Icon
	iconOnPadded         *fyne.Container
	iconTrailing         *widget.Icon
	isMobile             bool
	label                *widget.Label
	minSize              fyne.Size // cached for hover/top pos calcs
	resourceIconOn       fyne.Resource
	resourceIconTrailing fyne.Resource
	window               fyne.Window
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
		// This variant requires a placeholder
		placeholder = "PLACEHOLDER"
	}
	w := newFilterChipSelect(placeholder, options, changed, window)
	return w
}

func newFilterChipSelect(placeholder string, options []string, changed func(selected string), window fyne.Window) *FilterChipSelect {
	w := &FilterChipSelect{
		ClearLabel:           "Clear",
		iconTrailing:         widget.NewIcon(theme.MenuDropDownIcon()),
		isMobile:             fyne.CurrentDevice().IsMobile(),
		OnChanged:            changed,
		Text:                 placeholder,
		resourceIconOn:       theme.ConfirmIcon(),
		resourceIconTrailing: theme.MenuDropDownIcon(),
		window:               window,
	}
	w.ExtendBaseWidget(w)
	w.label = widget.NewLabel(w.Text)
	w.iconOn = widget.NewIcon(theme.ConfirmIcon())
	p := theme.Padding()
	w.iconOnPadded = container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), w.iconOn)
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
	if v == "" && w.Text == "" {
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
	w.Options = deduplicateSlice(options)
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
	if w.Text != "" && w.Selected != "" {
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
		options := slices.Clone(w.Options)
		if !w.SortDisabled {
			slices.SortFunc(options, func(a, b string) int {
				return strings.Compare(strings.ToLower(a), strings.ToLower(b))
			})
		}
		for _, o := range options {
			it := fyne.NewMenuItem(o, func() {
				w.SetSelected(o)
			})
			if w.Selected != "" {
				if o == w.Selected {
					it.Icon = theme.ConfirmIcon()
				} else {
					it.Icon = iconBlankSvg
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
	if !w.SortDisabled {
		slices.SortFunc(itemsFiltered, func(a, b string) int {
			return strings.Compare(strings.ToLower(a), strings.ToLower(b))
		})
	}
	var d dialog.Dialog
	list := widget.NewList(
		func() int {
			return len(itemsFiltered)
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(iconBlankSvg)
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
	d = dialog.NewCustomWithoutButtons("Filter by "+w.Text, c, w.window)
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
		w.iconOn.Resource = theme.NewDisabledResource(w.resourceIconOn)
		w.bg.StrokeColor = th.Color(theme.ColorNameDisabled, v)
		w.iconTrailing.Resource = theme.NewDisabledResource(w.resourceIconTrailing)
	} else {
		w.label.Importance = widget.MediumImportance
		w.iconOn.Resource = w.resourceIconOn
		w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
		w.iconTrailing.Resource = theme.NewThemedResource(w.resourceIconTrailing)
	}
	if w.Selected != "" {
		w.label.Text = w.Selected
		w.iconOnPadded.Show()
		if w.Disabled() {
			w.bg.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.bg.FillColor = th.Color(theme.ColorNameButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameButton, v)
		}
	} else {
		w.label.Text = w.Text
		w.iconOnPadded.Hide()
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
	w.iconOn.Refresh()
	w.iconTrailing.Refresh()
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
				container.NewVBox(layout.NewSpacer(), w.iconOnPadded, layout.NewSpacer()),
				container.NewVBox(layout.NewSpacer(), w.label, layout.NewSpacer()),
				container.NewVBox(layout.NewSpacer(), w.iconTrailing, layout.NewSpacer()),
				layout.NewSpacer(),
			),
		)))
	return widget.NewSimpleRenderer(c)
}
