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

	OnChanged func(column string, descending bool)

	ascResource       fyne.Resource
	column            string
	columns           []string
	defaultColumn     string
	defaultDescending bool
	descResource      fyne.Resource
	descending        bool
	offResource       fyne.Resource
}

// NewSortChip returns a new [SortChip] object.
func NewSortChip(changed func(col string, descending bool)) *SortChip {
	w := &SortChip{
		OnChanged: changed,
	}
	w.ExtendBaseWidget(w)
	w.OnTapped = w.showMenu
	w.ascResource = theme.NewThemedResource(iconSortAscendingSvg)
	w.descResource = theme.NewThemedResource(iconSortDescendingSvg)
	w.offResource = theme.NewThemedResource(iconSortSvg)
	return w
}

// Set defines the columns for sorting and the default column and direction.
func (w *SortChip) Set(columns []string, defaultColumn string, defaultDescending bool) {
	if len(columns) == 0 {
		w.columns = nil
		w.column = ""
		w.defaultColumn = ""
		w.defaultDescending = false
		w.descending = false
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

	w.column = defaultColumn
	w.defaultColumn = defaultColumn
	w.defaultDescending = defaultDescending
	w.descending = defaultDescending
	w.Refresh()
}

// ResetSilent resets the sorting to default without calling OnChanged.
func (w *SortChip) ResetSilent() {
	w.column = w.defaultColumn
	w.descending = w.defaultDescending
	w.Refresh()
}

func (w *SortChip) Refresh() {
	if w.column == "" {
		w.Text = "(no sort)"
	} else {
		w.Text = w.column
	}
	var iconResource fyne.Resource
	if w.descending {
		iconResource = w.descResource
	} else {
		iconResource = w.ascResource
	}

	w.LeadingIcon = iconResource

	isDefault := w.descending == w.defaultDescending && w.column == w.defaultColumn
	w.On = !isDefault
	w.Chip.Refresh()
}

func (w *SortChip) showMenu() {
	if len(w.columns) == 0 {
		return
	}
	oldColumn := w.column
	oldDirection := w.descending

	onChanged := func(column string, descending bool) {
		w.column = column
		w.descending = descending
		if oldColumn == w.column && oldDirection == w.descending {
			return
		}
		w.Refresh()
		if w.OnChanged != nil {
			w.OnChanged(w.column, w.descending)
		}
	}

	var items []*fyne.MenuItem

	sortTitle := fyne.NewMenuItem("Sort by ", nil)
	sortTitle.Disabled = true
	items = append(items, sortTitle)

	for _, c := range w.columns {
		it := fyne.NewMenuItem(c, func() {
			onChanged(c, w.descending)
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

	for _, d := range []bool{false, true} {
		var name string
		if d {
			name = "Descending"
		} else {
			name = "Ascending"
		}
		it := fyne.NewMenuItem(name, func() {
			onChanged(w.column, d)
		})
		if d == w.descending {
			it.Icon = theme.ConfirmIcon()
		} else {
			it.Icon = iconBlankSvg
		}
		items = append(items, it)
	}

	items = append(items, fyne.NewMenuItemSeparator())
	reset := fyne.NewMenuItem("Reset", func() {
		onChanged(w.defaultColumn, w.defaultDescending)
	})
	reset.Icon = theme.NewThemedResource(iconRestoreSvg)
	reset.Disabled = w.column == w.defaultColumn && w.descending == w.defaultDescending
	items = append(items, reset)

	menu := fyne.NewMenu("", items...)
	ShowPopUpMenuBelowLeading(w, menu)
}
