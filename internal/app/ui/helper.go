package ui

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
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

func newSortedColumnsWithDefault(n int, idx int, dir sortDir) *sortedColumns {
	sc := newSortedColumns(n)
	sc.set(idx, dir)
	return sc
}

// column returns the sort direction of a column
func (sc *sortedColumns) column(idx int) sortDir {
	if idx < 0 || idx >= len(sc.cols) {
		return sortOff
	}
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

// reset clears all columns.
func (sc *sortedColumns) reset() {
	for i := range sc.cols {
		sc.cols[i] = sortOff
	}
}

// set sets the sort direction for a column.
func (sc *sortedColumns) set(idx int, dir sortDir) {
	sc.reset()
	sc.cols[idx] = dir
}

// current returns which column is currently sorted or -1 if none are sorted.
func (sc *sortedColumns) size() int {
	return len(sc.cols)
}

// makeSortButton returns a button widget that can be used to sort columns.
func makeSortButton(headers []iwidget.HeaderDef, columns set.Set[int], sc *sortedColumns, process func(), window fyne.Window) *widget.Button {
	sortColumns := xslices.Map(headers, func(h iwidget.HeaderDef) string {
		return h.Text
	})
	b := widget.NewButtonWithIcon("???", icons.BlankSvg, nil)
	if len(headers) == 0 || sc.size() == 0 {
		return b // early exit when called without proper data
	}
	setSortButton := func() {
		var icon fyne.Resource
		col, order := sc.current()
		switch order {
		case sortAsc:
			icon = theme.NewThemedResource(icons.SortAscendingSvg)
		case sortDesc:
			icon = theme.NewThemedResource(icons.SortDescendingSvg)
		default:
			icon = theme.NewThemedResource(icons.SortSvg)
		}
		if col != -1 {
			b.Text = sortColumns[col]
		} else {
			b.Text = "Sort"
		}
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
		if col != -1 {
			cols.Selected = sortColumns[col]
		} else {
			cols.Selected = sortColumns[0] // default to first column
		}
		dir := widget.NewRadioGroup([]string{"Ascending", "Descending"}, nil)
		switch order {
		case sortDesc:
			dir.Selected = "Descending"
		default:
			dir.Selected = "Ascending"
		}
		var d dialog.Dialog
		okButton := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
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
			d.Hide()
		})
		okButton.Importance = widget.HighImportance
		p := theme.Padding()
		c := container.NewBorder(
			nil,
			container.New(layout.NewCustomPaddedLayout(3*p, 0, p, p), container.NewHBox(
				layout.NewSpacer(),
				widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
					d.Hide()
				}),
				widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), func() {
					sc.reset()
					process()
					setSortButton()
					d.Hide()
				}),
				okButton,
				layout.NewSpacer(),
			)),
			nil,
			nil,
			container.NewVBox(
				widget.NewLabel("Field"),
				cols,
				widget.NewLabel("Direction"),
				dir,
			),
		)
		d = dialog.NewCustomWithoutButtons("Sort By", c, window)
		_, s := window.Canvas().InteractiveArea()
		d.Resize(fyne.NewSize(s.Width, s.Height*0.8))
		d.Show()
	}
	setSortButton()
	return b
}
