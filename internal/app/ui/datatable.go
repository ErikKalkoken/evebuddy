package ui

import (
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type sortDir uint

const (
	sortNone sortDir = iota
	sortOff
	sortAsc
	sortDesc
)

type headerDef struct {
	label       string
	notSortable bool
	refresh     bool
	width       float32
}

func (h headerDef) Width() float32 {
	if h.width > 0 {
		return h.width
	}
	x := widget.NewLabel(h.label)
	return x.MinSize().Width
}

func maxHeaderWidth(headers []headerDef) float32 {
	var m float32
	for _, h := range headers {
		l := widget.NewLabel(h.label)
		m = max(l.MinSize().Width, m)
	}
	return m
}

// makeDataTable returns a table for showing data and which can be sorted.
func makeDataTable[S ~[]E, E any](
	headers []headerDef,
	data *S,
	makeCell func(int, E) []widget.RichTextSegment,
	columnSorter *columnSorter,
	filterRows func(int),
	onSelected func(int, E),
) *widget.Table {
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(*data), len(headers)
		},
		func() fyne.CanvasObject {
			return iwidget.NewRichText()
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			cell := co.(*iwidget.RichText)
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
	iconNone := theme.NewThemedResource(icons.BlankSvg)
	iconSortOff := theme.NewThemedResource(icons.SortSvg)
	t.CreateHeader = func() fyne.CanvasObject {
		icon := widget.NewIcon(iconSortOff)
		actionLabel := kxwidget.NewTappableLabel("Template", nil)
		label := widget.NewLabel("Template")
		return container.NewBorder(nil, nil, nil, icon, container.NewStack(actionLabel, label))
	}
	iconMap := map[sortDir]fyne.Resource{
		sortOff:  iconSortOff,
		sortAsc:  theme.NewPrimaryThemedResource(icons.SortAscendingSvg),
		sortDesc: theme.NewPrimaryThemedResource(icons.SortDescendingSvg),
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		h := headers[tci.Col]
		row := co.(*fyne.Container).Objects

		actionLabel := row[0].(*fyne.Container).Objects[0].(*kxwidget.TappableLabel)
		label := row[0].(*fyne.Container).Objects[1].(*widget.Label)
		icon := row[1].(*widget.Icon)

		dir := columnSorter.column(tci.Col)
		if dir == sortNone {
			label.SetText(h.label)
			label.Show()
			actionLabel.Hide()
			icon.Hide()
			return
		}
		actionLabel.OnTapped = func() {
			filterRows(tci.Col)
		}
		actionLabel.SetText(h.label)
		actionLabel.Show()
		icon.Show()
		label.Hide()

		r, ok := iconMap[dir]
		if !ok {
			r = iconNone
		}
		icon.SetResource(r)
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
	w := theme.Padding() + theme.IconInlineSize()
	for i, h := range headers {
		t.SetColumnWidth(i, h.Width()+w)
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
				row := container.New(rowLayout, widget.NewLabel(h.label), iwidget.NewRichText())
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
				data := row[1].(*iwidget.RichText)
				data.Segments = iwidget.AlignRichTextSegments(fyne.TextAlignTrailing, makeCell(col, r))
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
	cols       []sortDir
	headers    []headerDef
	sortButton *sortButton
	defaultIdx int
	defaultDir sortDir
}

func newColumnSorter(headers []headerDef) *columnSorter {
	cs := &columnSorter{
		cols:       make([]sortDir, len(headers)),
		headers:    headers,
		defaultIdx: -1,
		defaultDir: sortOff,
	}
	cs.clear()
	return cs
}

func newColumnSorterWithInit(headers []headerDef, idx int, dir sortDir) *columnSorter {
	cs := newColumnSorter(headers)
	cs.defaultIdx = idx
	cs.defaultDir = dir
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

// reset sets the columns to their default state.
func (cs *columnSorter) reset() {
	cs.set(cs.defaultIdx, cs.defaultDir)
}

// clear removes sorting from all columns.
func (cs *columnSorter) clear() {
	for i := range cs.cols {
		var dir sortDir
		if cs.headers[i].notSortable {
			dir = sortNone
		} else {
			dir = sortOff
		}
		cs.cols[i] = dir
	}
}

// set sets the sort direction for a column.
func (cs *columnSorter) set(idx int, dir sortDir) {
	cs.clear()
	cs.cols[idx] = dir
	if cs.sortButton != nil {
		cs.sortButton.set(idx, dir)
	}
}

// current returns which column is currently sorted or -1 if none are sorted.
func (cs *columnSorter) size() int {
	return len(cs.cols)
}

func (cs *columnSorter) sort(idx int, f func(sortCol int, dir sortDir)) {
	var dir sortDir
	if idx >= 0 {
		dir = cs.cols[idx]
		if dir == sortNone {
			return
		} else {
			dir++
			if dir > sortDesc {
				dir = sortOff
			}
			cs.set(idx, dir)
		}
	} else {
		idx, dir = cs.current()
	}
	if idx >= 0 && dir != sortOff {
		f(idx, dir)
	}
}

// newSortButton returns a new sortButton.
func (cs *columnSorter) newSortButton(headers []headerDef, process func(), window fyne.Window, ignoredColumns ...int) *sortButton {
	sortColumns := xslices.Map(headers, func(h headerDef) string {
		return h.label
	})
	w := &sortButton{
		sortColumns: sortColumns,
	}
	w.ExtendBaseWidget(w)
	w.Text = "???"
	w.Icon = icons.BlankSvg
	if len(headers) == 0 || cs.size() == 0 || len(ignoredColumns) > len(headers) {
		slog.Warn("makeSortButton called with invalid parameters")
		return w // early exit when called without proper data
	}
	ignored := set.Of(ignoredColumns...)
	w.OnTapped = func() {
		col, dir := cs.current()
		var fields []string
		for i, h := range headers {
			if !h.notSortable && !ignored.Contains(i) {
				fields = append(fields, h.label)
			}
		}
		radioCols := widget.NewRadioGroup(fields, nil)
		if col != -1 {
			radioCols.Selected = sortColumns[col]
		} else {
			radioCols.Selected = sortColumns[0] // default to first column
		}
		radioDir := widget.NewRadioGroup([]string{"Ascending", "Descending"}, nil)
		switch dir {
		case sortDesc:
			radioDir.Selected = "Descending"
		default:
			radioDir.Selected = "Ascending"
		}
		var d dialog.Dialog
		okButton := widget.NewButtonWithIcon("Sort", theme.ConfirmIcon(), func() {
			col := slices.Index(sortColumns, radioCols.Selected)
			if col == -1 {
				return
			}
			switch radioDir.Selected {
			case "Ascending":
				dir = sortAsc
			case "Descending":
				dir = sortDesc
			}
			cs.set(col, dir)
			process()
			w.set(col, dir)
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
				widget.NewButtonWithIcon("Reset", theme.DeleteIcon(), func() {
					cs.reset()
					process()
					d.Hide()
				}),
				okButton,
				layout.NewSpacer(),
			)),
			nil,
			nil,
			container.NewVBox(
				widget.NewLabel("Field"),
				radioCols,
				widget.NewLabel("Direction"),
				radioDir,
			),
		)
		d = dialog.NewCustomWithoutButtons("Sort By", c, window)
		_, s := window.Canvas().InteractiveArea()
		d.Resize(fyne.NewSize(s.Width, s.Height*0.8))
		d.Show()
	}
	w.set(cs.current())
	cs.sortButton = w
	return w
}

type sortButton struct {
	widget.Button

	sortColumns []string
}

func (w *sortButton) set(col int, dir sortDir) {
	switch dir {
	case sortAsc:
		w.Icon = theme.NewThemedResource(icons.SortAscendingSvg)
	case sortDesc:
		w.Icon = theme.NewThemedResource(icons.SortDescendingSvg)
	default:
		w.Icon = theme.NewThemedResource(icons.SortSvg)
	}
	if col != -1 {
		w.Text = w.sortColumns[col]
	} else {
		w.Text = "Sort"
	}
	w.Refresh()
}
