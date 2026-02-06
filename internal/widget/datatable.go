package widget

import (
	"fmt"
	"iter"
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
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

// SortDir represents the sort direction for a data table.
type SortDir uint

const (
	sortNone SortDir = iota
	SortOff
	SortAsc
	SortDesc
)

func (s SortDir) isSorting() bool {
	return s == SortAsc || s == SortDesc
}

// DataColumn represents a column in a data table.
type DataColumn[T any] struct {
	// Column ID. Must be unique and > 0.
	ID int

	// Label of a column displayed to the user.
	Label string

	// Width of a column in Fyne units. Will auto size to header width when omitted.
	Width float32

	// Create for this column. Will use defaultCreate if not defined.
	Create func() fyne.CanvasObject

	// Update for this column. Must be defined.
	Update func(r T, co fyne.CanvasObject)

	// Sort defines the compare function to apply for sorting this column.
	// Omit to disable sort for this column.
	Sort func(a, b T) int
}

func (h DataColumn[T]) minWidth() float32 {
	if h.Width > 0 {
		return h.Width
	}
	x := widget.NewLabel(h.Label)
	return x.MinSize().Width
}

// DataColumns represents the columns of a data table.
type DataColumns[T any] struct {
	cols      []DataColumn[T]
	idxLookup map[int]int // maps IDs to index on the cols slice
}

// NewDataColumns creates and returns a [DataColumns].
// It panics if semantic checks fail.
func NewDataColumns[T any](cols []DataColumn[T]) DataColumns[T] {
	if len(cols) == 0 {
		panic("must define at least 1 column")
	}
	lookup := make(map[int]int)
	for idx, c := range cols {
		if c.ID < 1 {
			panic("IDs must be > 0")
		}
		if _, found := lookup[c.ID]; found {
			panic(fmt.Sprintf("%s: col %d duplicate", c.Label, c.ID))
		}
		lookup[c.ID] = idx
	}
	d := DataColumns[T]{
		cols:      cols,
		idxLookup: lookup,
	}
	return d
}

func (d DataColumns[T]) IDLookup(idx int) (int, bool) {
	if idx < 0 || idx >= len(d.cols) {
		return 0, false
	}
	return d.cols[idx].ID, true
}

// ColumnByIndex return the definition of a column. n is the slice index of the column.
func (d DataColumns[T]) ColumnByIndex(idx int) (DataColumn[T], bool) {
	if idx < 0 || idx >= len(d.cols) {
		return DataColumn[T]{}, false
	}
	return d.cols[idx], true
}

func (d DataColumns[T]) all() iter.Seq2[int, DataColumn[T]] {
	return slices.All(d.cols)
}

// maxColumnWidth returns the maximum width of any column.
func (d DataColumns[T]) maxColumnWidth() float32 {
	var m float32
	for _, c := range d.cols {
		l := widget.NewLabel(c.Label)
		m = max(l.MinSize().Width, m)
	}
	return m
}

func (d DataColumns[T]) size() int {
	return len(d.cols)
}

func (d DataColumns[T]) values() iter.Seq[DataColumn[T]] {
	return slices.Values(d.cols)
}

// ColumnSorter represents an ordered list of columns which can be sorted.
type ColumnSorter[T any] struct {
	cols       []SortDir
	columns    DataColumns[T]
	initialDir SortDir
	initialID  int
	isMobile   bool
	sortButton *SortButton
}

// NewColumnSorter returns a new ColumSorter.
// idx and dir defines the initially sorted column.
// It panics if semantic checks fail.
func NewColumnSorter[T any](columns DataColumns[T], id int, dir SortDir) *ColumnSorter[T] {
	if _, found := columns.idxLookup[id]; !found {
		dir = SortOff
	}
	if dir == SortOff {
		id = columns.cols[0].ID
	}
	cs := &ColumnSorter[T]{
		cols:       make([]SortDir, columns.size()),
		columns:    columns,
		initialDir: dir,
		initialID:  id,
		isMobile:   fyne.CurrentDevice().IsMobile(),
	}
	cs.init()
	cs.Set(id, dir)
	return cs
}

// init resets sorting for all columns.
func (cs *ColumnSorter[T]) init() {
	for i := range cs.cols {
		var dir SortDir
		if cs.columns.cols[i].Sort == nil {
			dir = sortNone
		} else {
			dir = SortOff
		}
		cs.cols[i] = dir
	}
}

// column returns the sort direction of a column
func (cs *ColumnSorter[T]) column(idx int) SortDir {
	if idx < 0 || idx >= len(cs.cols) {
		return SortOff
	}
	return cs.cols[idx]
}

// current returns which column is currently sorted or -1 if none are sorted.
func (cs *ColumnSorter[T]) current() (idx int, dir SortDir) {
	for i, v := range cs.cols {
		if v.isSorting() {
			return i, v
		}
	}
	return -1, SortOff
}

// reset sets the columns to their initial state.
func (cs *ColumnSorter[T]) reset() {
	cs.Set(cs.initialID, cs.initialDir)
}

// Set sets the sort direction for a column.
func (cs *ColumnSorter[T]) Set(id int, dir SortDir) {
	idx, ok := cs.columns.idxLookup[id]
	if !ok {
		return
	}
	cs.setIdx(idx, dir)
}

func (cs *ColumnSorter[T]) setIdx(idx int, dir SortDir) {
	cs.init()
	cs.cols[idx] = dir
	if cs.sortButton != nil {
		cs.sortButton.set(idx, dir)
	}
}

func (cs *ColumnSorter[T]) size() int {
	return len(cs.cols)
}

// CalcSort calculates how and if to apply sorting to column idx.
func (cs *ColumnSorter[T]) CalcSort(id int) (int, SortDir, bool) {
	idx, ok := cs.columns.idxLookup[id]
	if !ok {
		idx = -1
	}
	var dir SortDir
	if idx >= 0 {
		dir = cs.cols[idx]
		if dir == sortNone {
			return 0, 0, false
		}
		dir++
		if dir > SortDesc {
			dir = SortAsc
		}
		cs.setIdx(idx, dir)
	} else {
		idx, dir = cs.current()
	}
	doSort := idx >= 0 && dir.isSorting()
	if doSort {
		col := cs.columns.cols[idx]
		id = col.ID
	}
	return id, dir, doSort
}

func (cs *ColumnSorter[T]) SortRows(rows []T, sortCol int, dir SortDir, doSort bool) {
	if !doSort {
		return
	}
	idx, ok := cs.columns.idxLookup[sortCol]
	if !ok {
		return
	}
	f := cs.columns.cols[idx].Sort
	if f == nil {
		return
	}
	slices.SortFunc(rows, func(a, b T) int {
		x := f(a, b)
		if dir == SortAsc {
			return x
		} else {
			return -1 * x
		}
	})
}

// A SortButton represents a button for sorting a data table.
// It is supposed to be used in mobile views.
type SortButton struct {
	widget.Button

	sortColumns []string
}

// NewSortButton returns a new sortButton.
func (cs *ColumnSorter[T]) NewSortButton(process func(), window fyne.Window, ignoredColumns ...int) *SortButton {
	sortColumns := slices.Collect(xiter.Map(cs.columns.values(), func(h DataColumn[T]) string {
		return h.Label
	}))
	w := &SortButton{
		sortColumns: sortColumns,
	}
	w.ExtendBaseWidget(w)
	w.Text = "???"
	w.Icon = icons.BlankSvg
	if cs.columns.size() == 0 || cs.size() == 0 || len(ignoredColumns) > cs.columns.size() {
		panic("NewSortButton called with invalid parameters")
	}
	ignored := set.Of(ignoredColumns...)
	w.OnTapped = func() {
		col, dir := cs.current()
		var fields []string
		for i, h := range cs.columns.all() {
			if h.Sort != nil && !ignored.Contains(i) {
				fields = append(fields, h.Label)
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
		case SortDesc:
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
				dir = SortAsc
			case "Descending":
				dir = SortDesc
			}
			cs.setIdx(col, dir)
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
		if cs.isMobile {
			_, s := window.Canvas().InteractiveArea()
			d.Resize(fyne.NewSize(s.Width, s.Height*0.8))
		}
		d.Show()
	}
	w.set(cs.current())
	cs.sortButton = w
	return w
}

func (w *SortButton) set(idx int, dir SortDir) {
	switch dir {
	case SortAsc:
		w.Icon = theme.NewThemedResource(icons.SortAscendingSvg)
	case SortDesc:
		w.Icon = theme.NewThemedResource(icons.SortDescendingSvg)
	default:
		w.Icon = theme.NewThemedResource(icons.SortSvg)
	}
	if idx != -1 {
		w.Text = w.sortColumns[idx]
	} else {
		w.Text = "Sort"
	}
	w.Refresh()
}

func MakeDataTable[S ~[]E, E any](
	columns DataColumns[E],
	data *S,
	defaultCreate func() fyne.CanvasObject,
	columnSorter *ColumnSorter[E],
	filterRows func(id int),
	onSelected func(int, E),
) *widget.Table {
	if defaultCreate == nil {
		panic("Must define default create")
	}
	for _, col := range columns.cols {
		if col.Update == nil {
			panic(fmt.Sprintf("Column missing update: %d", col.ID))
		}
	}
	var t *widget.Table
	var isCustom bool
	for _, col := range columns.cols {
		if col.Create != nil {
			isCustom = true
			break
		}
	}
	if isCustom {
		stackIdxLookup := make(map[int]int)
		t = widget.NewTable(
			func() (rows int, cols int) {
				return len(*data), columns.size()
			},
			func() fyne.CanvasObject {
				c := container.NewStack()
				c.Add(defaultCreate())
				var stackIdx int
				for idx, col := range columns.cols {
					if f := col.Create; f != nil {
						c.Add(f())
						stackIdx++
						stackIdxLookup[idx] = stackIdx
					} else {
						stackIdxLookup[idx] = 0
					}
				}
				return c
			},
			func(tci widget.TableCellID, co fyne.CanvasObject) {
				if tci.Row >= len(*data) || tci.Row < 0 {
					return
				}
				stack := co.(*fyne.Container)
				var co2 fyne.CanvasObject
				for i, c := range stack.Objects {
					if stackIdxLookup[tci.Col] == i {
						c.Show()
						co2 = c
					} else {
						c.Hide()
					}
				}
				r := (*data)[tci.Row]
				columns.cols[tci.Col].Update(r, co2)
			},
		)
	} else {
		t = widget.NewTable(
			func() (rows int, cols int) {
				return len(*data), columns.size()
			},
			func() fyne.CanvasObject {
				return defaultCreate()
			},
			func(tci widget.TableCellID, co fyne.CanvasObject) {
				if tci.Row >= len(*data) || tci.Row < 0 {
					return
				}
				r := (*data)[tci.Row]
				columns.cols[tci.Col].Update(r, co)
			},
		)
	}
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
	iconMap := map[SortDir]fyne.Resource{
		SortOff:  iconSortOff,
		SortAsc:  theme.NewPrimaryThemedResource(icons.SortAscendingSvg),
		SortDesc: theme.NewPrimaryThemedResource(icons.SortDescendingSvg),
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		h, ok := columns.ColumnByIndex(tci.Col)
		if !ok {
			return
		}
		row := co.(*fyne.Container).Objects

		actionLabel := row[0].(*fyne.Container).Objects[0].(*kxwidget.TappableLabel)
		label := row[0].(*fyne.Container).Objects[1].(*widget.Label)
		icon := row[1].(*widget.Icon)

		dir := columnSorter.column(tci.Col)
		if dir == sortNone {
			label.SetText(h.Label)
			label.Show()
			actionLabel.Hide()
			icon.Hide()
			return
		}
		actionLabel.OnTapped = func() {
			if id, ok := columns.IDLookup(tci.Col); ok {
				filterRows(id)
			}
		}
		actionLabel.SetText(h.Label)
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
	for i, h := range columns.cols {
		t.SetColumnWidth(i, h.minWidth()+w)
	}
	return t
}

// MakeDataList returns a list for showing a data table in a generic way.
// This is meant for showing table content on mobile.
func MakeDataList[S ~[]E, E any](
	def DataColumns[E],
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
			rowLayout := kxlayout.NewColumns(def.maxColumnWidth() + theme.Padding())
			c := container.New(layout.NewCustomPaddedVBoxLayout(0))
			for _, h := range def.cols {
				row := container.New(rowLayout, widget.NewLabel(h.Label), NewRichText())
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
			for col := range def.size() {
				row := f[col*2].(*fyne.Container).Objects[1].(*fyne.Container).Objects
				cell := row[1].(*RichText)
				id, ok := def.IDLookup(col)
				if !ok {
					continue
				}
				cell.Segments = AlignRichTextSegments(fyne.TextAlignTrailing, makeCell(id, r))
				cell.Wrapping = fyne.TextWrapWord
				bg := f[col*2].(*fyne.Container).Objects[0]
				if col == 0 {
					bg.Show()
					for _, s := range cell.Segments {
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
				cell.Refresh()
				divider := f[col*2+1]
				if col > 0 && col < def.size()-1 {
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
