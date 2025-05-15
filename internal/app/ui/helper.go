package ui

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// makeGridOrList makes and returns a GridWrap on desktop and a List on mobile.
//
// This allows the grid items to render nicely as list on mobile and also enable truncation.
func makeGridOrList(isMobile bool, length func() int, makeCreateItem func(trunc fyne.TextTruncation) func() fyne.CanvasObject, updateItem func(id int, co fyne.CanvasObject), makeOnSelected func(unselectAll func()) func(int)) fyne.CanvasObject {
	var w fyne.CanvasObject
	if isMobile {
		w = widget.NewList(length, makeCreateItem(fyne.TextTruncateEllipsis), updateItem)
		l := w.(*widget.List)
		l.OnSelected = makeOnSelected(func() {
			l.UnselectAll()
		})
	} else {
		w = widget.NewGridWrap(length, makeCreateItem(fyne.TextTruncateOff), updateItem)
		g := w.(*widget.GridWrap)
		g.OnSelected = makeOnSelected(func() {
			g.UnselectAll()
		})
	}
	return w
}

// sortedColumns represents an ordered list of columns which can be sorted.
type sortedColumns struct {
	cols []sortDir
}

func newSortedColumns(n int) *sortedColumns {
	sc := &sortedColumns{cols: make([]sortDir, n)}
	return sc
}

// column returns the sort direction of a column
func (sc *sortedColumns) column(idx int) sortDir {
	return sc.cols[idx]
}

// current returns which column is currently sorted or -1 if none are sorted.
func (sc *sortedColumns) current() (int, sortDir) {
	for i, v := range sc.cols {
		if v != sortOff {
			return i, v
		}
	}
	return -1, sortOff
}

// cycleColumn cycles the sort direction of a given column and returns the new direction.
func (sc *sortedColumns) cycleColumn(idx int) sortDir {
	dir := sc.cols[idx]
	dir++
	if dir > sortDesc {
		dir = sortOff
	}
	sc.set(idx, dir)
	return dir
}

// set sets the sort direction for a column.
func (sc *sortedColumns) set(idx int, dir sortDir) {
	for i := range sc.cols {
		sc.cols[i] = sortOff
	}
	sc.cols[idx] = dir
}

func makeSortButton(headers []iwidget.HeaderDef, columns set.Set[int], sc *sortedColumns, process func(), window fyne.Window) *widget.Button {
	sortColumns := xslices.Map(headers, func(h iwidget.HeaderDef) string {
		return h.Text
	})
	b := widget.NewButtonWithIcon("", icons.BlankSvg, nil)
	setSortButton := func() {
		var icon fyne.Resource
		col, order := sc.current()
		switch order {
		case sortAsc:
			icon = theme.NewThemedResource(icons.SortAscendingSvg)
		case sortDesc:
			icon = theme.NewThemedResource(icons.SortDescendingSvg)
		}
		b.Text = sortColumns[col]
		b.Icon = icon
		b.Refresh()
	}
	b.OnTapped = func() {
		col, order := sc.current()
		var fields []string
		for i, h := range headers {
			if columns.Contains(i) {
				fields = append(fields, h.Text)
			}
		}
		cols := widget.NewRadioGroup(fields, nil)
		cols.Selected = sortColumns[col]
		dir := widget.NewRadioGroup([]string{"Ascending", "Descending"}, nil)
		switch order {
		case sortAsc:
			dir.Selected = "Ascending"
		case sortDesc:
			dir.Selected = "Descending"
		}
		c := container.NewVBox(
			widget.NewLabel("Field"),
			cols,
			widget.NewLabel("Direction"),
			dir,
		)
		d := dialog.NewCustomConfirm("Sort By", "OK", "Cancel", c, func(ok bool) {
			if !ok {
				return
			}
			col := slices.Index(sortColumns, cols.Selected)
			if col == -1 {
				return
			}
			switch dir.Selected {
			case "Ascending":
				order = sortAsc
			case "Descending":
				order = sortDesc
			}
			sc.set(col, order)
			process()
			setSortButton()
		}, window)
		d.Show()
	}
	setSortButton()
	return b
}
