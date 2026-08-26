package xwidget

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// SortChip represents a widget for sorting.
//
// It shows the currently selected column and order for sorting.
// When clicked it will show the sorting options in a drop down menu.
// The chip is shown in normal state when default sorting is selected.
// Otherwise it is shown in active state.
type SortChip struct {
	Chip

	OnChanged func(column string, dir SortDir)

	ascResource      fyne.Resource
	column           string
	columns          []string
	defaultColumn    string
	defaultDirection SortDir
	descResource     fyne.Resource
	direction        SortDir
	offResource      fyne.Resource
}

// NewSortChip returns a new [SortChip] object.
func NewSortChip(changed func(col string, dir SortDir)) *SortChip {
	w := &SortChip{
		OnChanged: changed,
	}
	w.ExtendBaseWidget(w)
	w.Init()
	return w
}

// Init initializes the widget.
// Must be called when extending the widget.
func (w *SortChip) Init() {
	w.OnTapped = w.showMenu
	w.ascResource = theme.NewThemedResource(iconSortAscendingSvg)
	w.descResource = theme.NewThemedResource(iconSortDescendingSvg)
	w.offResource = theme.NewThemedResource(iconSortSvg)
}

// Set defines the columns for sorting and the default column and direction.
func (w *SortChip) Set(columns []string, defaultColumn string, defaultDirection SortDir) {
	if len(columns) == 0 {
		w.columns = nil
		w.column = ""
		w.defaultColumn = ""
		w.defaultDirection = SortOff
		w.direction = SortOff
		w.Refresh()
		return
	}

	if !slices.Contains(columns, defaultColumn) {
		defaultColumn = ""
	}

	columns2 := xslices.Deduplicate(columns)
	slices.Sort(columns2)
	w.columns = columns2

	if defaultColumn == "" {
		defaultColumn = columns2[0]
	}
	if defaultDirection != SortAsc && defaultDirection != SortDesc {
		defaultDirection = SortAsc
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
	if w.column == "" || w.direction == SortOff {
		w.Text = "(no sort)"
	} else {
		w.Text = w.column
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
	w.LeadingIcon = iconResource

	isDefault := w.direction == w.defaultDirection && w.column == w.defaultColumn
	w.On = !isDefault
	w.Chip.Refresh()
}

func (w *SortChip) showMenu() {
	if len(w.columns) == 0 {
		return
	}
	oldColumn := w.column
	oldDirection := w.direction

	onChanged := func(column string, dir SortDir) {
		w.column = column
		w.direction = dir
		if oldColumn == w.column && oldDirection == w.direction {
			return
		}
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
	reset.Icon = theme.NewThemedResource(iconRestoreSvg)
	reset.Disabled = w.column == w.defaultColumn && w.direction == w.defaultDirection
	items = append(items, reset)

	menu := fyne.NewMenu("", items...)
	ShowPopUpMenuBelowLeading(w, menu)
}
