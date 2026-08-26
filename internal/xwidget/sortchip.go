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

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// SortChip represents a widget for sorting.
//
// It shows the currently selected column and order for sorting.
// When clicked it will show the sorting options in a drop down menu.
// The chip is shown in normal state when default sorting is selected.
// Otherwise it is shown in active state.
type SortChip struct {
	widget.DisableableWidget

	OnChanged func(column string, dir SortDir)

	ascResource      fyne.Resource
	bg               *canvas.Rectangle
	column           string
	columns          []string
	defaultColumn    string
	defaultDirection SortDir
	descResource     fyne.Resource
	direction        SortDir
	focused          bool
	hovered          bool
	icon             *widget.Icon
	label            *widget.Label
	offResource      fyne.Resource
}

var _ desktop.Hoverable = (*SortChip)(nil)
var _ fyne.Disableable = (*SortChip)(nil)
var _ fyne.Focusable = (*SortChip)(nil)
var _ fyne.Tappable = (*SortChip)(nil)
var _ fyne.Widget = (*SortChip)(nil)

// NewSortChip returns a new [SortChip] object.
func NewSortChip(changed func(col string, dir SortDir)) *SortChip {
	w := &SortChip{
		ascResource:  theme.NewThemedResource(iconSortAscendingSvg),
		bg:           canvas.NewRectangle(color.Transparent),
		descResource: theme.NewThemedResource(iconSortDescendingSvg),
		icon:         widget.NewIcon(iconBlankSvg),
		label:        widget.NewLabel(""),
		offResource:  theme.NewThemedResource(iconSortSvg),
		OnChanged:    changed,
	}
	w.ExtendBaseWidget(w)
	w.bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	w.bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	return w
}

// Set defines the columns for sorting and the default column and direction.
func (w *SortChip) Set(columns []string, defaultColumn string, defaultDirection SortDir) {
	if !slices.Contains(columns, defaultColumn) {
		defaultColumn = ""
	}
	if len(columns) == 0 {
		defaultColumn = ""
		defaultDirection = SortOff
		clear(w.columns)
	} else {
		columns2 := xslices.Deduplicate(columns)
		slices.Sort(columns2)
		w.columns = columns2
		if defaultColumn == "" {
			defaultColumn = columns2[0]
		}
		if defaultDirection != SortAsc && defaultDirection != SortDesc {
			defaultDirection = SortAsc
		}
	}
	w.column = defaultColumn
	w.defaultColumn = defaultColumn
	w.defaultDirection = defaultDirection
	w.direction = defaultDirection
	w.Refresh()
}

// ResetSilent resets the sorting to default without calling OnChanged.
func (w *SortChip) ResetSilent() {
	w.column = w.defaultColumn
	w.direction = w.defaultDirection
	w.Refresh()
}

func (w *SortChip) Refresh() {
	w.updateState()
	w.bg.Refresh()
	w.label.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *SortChip) updateState() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if w.column == "" || w.direction == SortOff {
		w.label.Text = "(no sort)"
	} else {
		w.label.Text = w.column
	}
	var iconResource fyne.Resource
	switch w.direction {
	case SortAsc:
		iconResource = w.ascResource
	case SortDesc:
		iconResource = w.descResource
	case SortOff:
		iconResource = w.offResource
	default:
		iconResource = iconBlankSvg
	}

	if w.Disabled() {
		w.label.Importance = widget.LowImportance
		w.icon.Resource = theme.NewDisabledResource(iconResource)
		w.bg.StrokeColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		w.label.Importance = widget.MediumImportance
		w.icon.Resource = iconResource
		w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	}

	isDefault := w.direction == w.defaultDirection && w.column == w.defaultColumn
	if !isDefault {
		if w.Disabled() {
			w.bg.FillColor = th.Color(theme.ColorNameDisabledButton, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameDisabledButton, v)
		} else {
			w.bg.FillColor = th.Color(theme.ColorNameSelection, v)
			w.bg.StrokeColor = th.Color(theme.ColorNameSelection, v)
		}
	} else {
		w.bg.FillColor = color.Transparent
	}

	if w.focused {
		w.bg.StrokeColor = th.Color(theme.ColorNameFocus, v)
	}
}

func (w *SortChip) Tapped(pe *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	w.showMenu()
}

func (w *SortChip) showMenu() {
	oldColum := w.column
	oldDirection := w.direction

	onChanged := func(column string, dir SortDir) {
		w.column = column
		w.direction = dir
		if oldColum == w.column && oldDirection == w.direction {
			return
		}
		w.updateState()
		w.Refresh()
		if w.OnChanged != nil {
			w.OnChanged(w.column, w.direction)
		}
	}

	var items []*fyne.MenuItem

	sortTitle := fyne.NewMenuItem("Sort by ", nil)
	sortTitle.Disabled = true
	items = append(items, sortTitle)

	for _, c := range w.columns {
		it := fyne.NewMenuItem(c, func() {
			onChanged(c, w.direction)
		})
		if c == w.column {
			it.Icon = theme.ConfirmIcon()
		} else {
			it.Icon = iconBlankSvg
		}
		items = append(items, it)
	}

	orderTitle := fyne.NewMenuItem("Order", nil)
	orderTitle.Disabled = true
	items = append(items, orderTitle)

	for _, d := range []SortDir{SortAsc, SortDesc} {
		it := fyne.NewMenuItem(d.String(), func() {
			onChanged(w.column, d)
		})
		if d == w.direction {
			it.Icon = theme.ConfirmIcon()
		} else {
			it.Icon = iconBlankSvg
		}
		items = append(items, it)
	}

	items = append(items, fyne.NewMenuItemSeparator())
	reset := fyne.NewMenuItem("Reset", func() {
		onChanged(w.defaultColumn, w.defaultDirection)
	})
	reset.Icon = theme.DeleteIcon()
	reset.Disabled = w.column == w.defaultColumn && w.direction == w.defaultDirection
	items = append(items, reset)

	menu := fyne.NewMenu("", items...)
	ShowPopUpMenuBelowLeading(w, menu)
}

func (w *SortChip) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (w *SortChip) MouseIn(me *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	w.hovered = true
}

func (w *SortChip) MouseMoved(me *desktop.MouseEvent) {
	// needed to satisfy the interface only
}

func (w *SortChip) MouseOut() {
	w.hovered = false
}

// FocusGained is called when the Check has been given focus.
func (w *SortChip) FocusGained() {
	if w.Disabled() {
		return
	}
	w.focused = true
	w.Refresh()
}

// FocusLost is called when the Check has had focus removed.
func (w *SortChip) FocusLost() {
	w.focused = false
	w.Refresh()
}

// TypedRune receives text input events when the Check is focused.
func (w *SortChip) TypedRune(r rune) {
	if w.Disabled() {
		return
	}
	if r == ' ' {
		w.showMenu()
	}
}

// TypedKey receives key input events when the Check is focused.
func (w *SortChip) TypedKey(key *fyne.KeyEvent) {}

func (w *SortChip) CreateRenderer() fyne.WidgetRenderer {
	w.updateState()
	p := theme.Padding()
	c := container.NewStack(
		w.bg,
		container.NewCenter(container.New(
			layout.NewCustomPaddedHBoxLayout(0),
			container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, -p), w.icon),
			w.label,
		),
		))
	return widget.NewSimpleRenderer(c)
}
