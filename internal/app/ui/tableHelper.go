package ui

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

type headerDef struct {
	Text    string
	Width   float32
	Refresh bool
}

func maxHeaderWidth(headers []headerDef) float32 {
	var m float32
	for _, h := range headers {
		l := widget.NewLabel(h.Text)
		m = max(l.MinSize().Width, m)
	}
	return m
}

// makeDataTable returns a table for showing data.
func makeDataTable[S ~[]E, E any](
	headers []headerDef,
	data *S,
	makeCell func(int, E) []widget.RichTextSegment,
	onSelected func(int, E),
) *widget.Table {
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(*data), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewRichText()
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			cell := co.(*widget.RichText)
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			cell.Segments = makeCell(tci.Col, r)
			cell.Truncation = fyne.TextTruncateClip
			cell.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.StickyColumnCount = 1
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		h := headers[tci.Col]
		label := co.(*widget.Label)
		label.SetText(h.Text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		if onSelected != nil {
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			onSelected(tci.Col, r)
		}
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.Width)
	}
	return t
}

// makeDataTable returns a table for showing data and which can be sorted.
func makeDataTableWithSort[S ~[]E, E any](
	headers []headerDef,
	data *S,
	makeCell func(int, E) []widget.RichTextSegment,
	sortedColumns *columnSorter,
	filterRows func(int),
	onSelected func(int, E),
) *widget.Table {
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(*data), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewRichText()
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			cell := co.(*widget.RichText)
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			cell.Segments = makeCell(tci.Col, r)
			cell.Truncation = fyne.TextTruncateClip
			cell.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.StickyColumnCount = 1
	iconSortAsc := theme.NewPrimaryThemedResource(icons.SortAscendingSvg)
	iconSortDesc := theme.NewPrimaryThemedResource(icons.SortDescendingSvg)
	iconSortOff := theme.NewThemedResource(icons.SortSvg)
	t.CreateHeader = func() fyne.CanvasObject {
		icon := widget.NewIcon(iconSortOff)
		label := kxwidget.NewTappableLabel("Template", nil)
		return container.NewBorder(nil, nil, nil, icon, label)
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		h := headers[tci.Col]
		row := co.(*fyne.Container).Objects
		label := row[0].(*kxwidget.TappableLabel)
		label.OnTapped = func() {
			filterRows(tci.Col)
		}
		label.SetText(h.Text)
		icon := row[1].(*widget.Icon)
		switch sortedColumns.column(tci.Col) {
		case sortOff:
			icon.SetResource(iconSortOff)
		case sortAsc:
			icon.SetResource(iconSortAsc)
		case sortDesc:
			icon.SetResource(iconSortDesc)
		}
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		if onSelected != nil {
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			onSelected(tci.Col, r)
		}
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.Width)
	}
	return t
}

// makeDataList returns a list for showing a data table in a generic way.
// This is meant for showing table content on mobile.
func makeDataList[S ~[]E, E any](
	headers []headerDef,
	data *S,
	makeCell func(int, E) []widget.RichTextSegment,
	onSelected func(E),
) *widget.List {
	var l *widget.List
	l = widget.NewList(
		func() int {
			return len(*data)
		},
		func() fyne.CanvasObject {
			p := theme.Padding()
			rowLayout := kxlayout.NewColumns(maxHeaderWidth(headers) + theme.Padding())
			c := container.New(layout.NewCustomPaddedVBoxLayout(0))
			for _, h := range headers {
				row := container.New(rowLayout, widget.NewLabel(h.Text), widget.NewRichText())
				bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
				bg.Hide()
				c.Add(container.NewStack(bg, row))
				c.Add(container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, 2*p), widget.NewSeparator()))
			}
			return c
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			f := co.(*fyne.Container).Objects
			if id >= len(*data) || id < 0 {
				return
			}
			r := (*data)[id]
			for col := range len(headers) {
				row := f[col*2].(*fyne.Container).Objects[1].(*fyne.Container).Objects
				data := row[1].(*widget.RichText)
				data.Segments = makeCell(col, r)
				data.Wrapping = fyne.TextWrapWord
				bg := f[col*2].(*fyne.Container).Objects[0]
				if col == 0 {
					bg.Show()
					for _, s := range data.Segments {
						x, ok := s.(*widget.TextSegment)
						if !ok {
							continue
						}
						x.Style.TextStyle.Bold = true
					}
					label := row[0].(*widget.Label)
					label.TextStyle.Bold = true
					label.Refresh()
				} else {
					bg.Hide()
				}
				data.Refresh()
				divider := f[col*2+1]
				if col > 0 && col < len(headers)-1 {
					divider.Show()
				} else {
					divider.Hide()
				}
			}
			l.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if onSelected != nil {
			if id >= len(*data) || id < 0 {
				return
			}
			r := (*data)[id]
			onSelected(r)
		}
	}
	return l
}

// columnSorter represents an ordered list of columns which can be sorted.
type columnSorter struct {
	cols []sortDir
}

func newColumnSorter(n int) *columnSorter {
	cs := &columnSorter{cols: make([]sortDir, n)}
	return cs
}

func newColumnSorterWithInit(n int, idx int, dir sortDir) *columnSorter {
	cs := newColumnSorter(n)
	cs.set(idx, dir)
	return cs
}

// column returns the sort direction of a column
func (cs *columnSorter) column(idx int) sortDir {
	if idx < 0 || idx >= len(cs.cols) {
		return sortOff
	}
	return cs.cols[idx]
}

// current returns which column is currently sorted or -1 if none are sorted.
func (cs *columnSorter) current() (int, sortDir) {
	for i, v := range cs.cols {
		if v != sortOff {
			return i, v
		}
	}
	return -1, sortOff
}

// reset clears all columns.
func (cs *columnSorter) reset() {
	for i := range cs.cols {
		cs.cols[i] = sortOff
	}
}

// set sets the sort direction for a column.
func (cs *columnSorter) set(idx int, dir sortDir) {
	cs.reset()
	cs.cols[idx] = dir
}

// current returns which column is currently sorted or -1 if none are sorted.
func (cs *columnSorter) size() int {
	return len(cs.cols)
}

func (cs *columnSorter) sort(idx int, f func(sortCol int, dir sortDir)) {
	var dir sortDir
	if idx >= 0 {
		dir = cs.cols[idx]
		dir++
		if dir > sortDesc {
			dir = sortOff
		}
		cs.set(idx, dir)
	} else {
		idx, dir = cs.current()
	}
	if idx >= 0 && dir != sortOff {
		f(idx, dir)
	}
}

// makeSortButton returns a button widget that can be used to sort columns.
func makeSortButton(headers []headerDef, columns set.Set[int], sc *columnSorter, process func(), window fyne.Window) *widget.Button {
	sortColumns := xslices.Map(headers, func(h headerDef) string {
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
