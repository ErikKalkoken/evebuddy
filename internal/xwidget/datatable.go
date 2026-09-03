package xwidget

import (
	"fmt"
	"image/color"
	"iter"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

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

func (s SortDir) String() string {
	switch s {
	case sortNone:
		return "Undefined"
	case SortOff:
		return "No sort"
	case SortAsc:
		return "Ascending"
	case SortDesc:
		return "Descending"
	}
	return ""
}

func (s SortDir) isSorting() bool {
	return s == SortAsc || s == SortDesc
}

// DataColumn represents a column in a data table.
type DataColumn[T any] struct {
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
//
// A column's ID is the index of its position in the cols slice passed to [NewDataColumns].
type DataColumns[T any] struct {
	cols           []DataColumn[T]
	maxColumnWidth float32
}

// NewDataColumns creates and returns a [DataColumns].
//
// Objects are immutable.
// It panics if semantic checks fail.
func NewDataColumns[T any](cols []DataColumn[T]) DataColumns[T] {
	if len(cols) == 0 {
		panic("must define at least 1 column")
	}
	cols2 := slices.Clone(cols)
	dc := DataColumns[T]{
		cols:           cols2,
		maxColumnWidth: maxColumnWidth(cols2),
	}
	return dc
}

func maxColumnWidth[T any](cols []DataColumn[T]) float32 {
	var m float32
	for _, c := range cols {
		l := widget.NewLabel(c.Label)
		m = max(l.MinSize().Width, m)
	}
	return m
}

// ColumnByIndex return the definition of a column. n is the slice index of the column.
func (dc DataColumns[T]) ColumnByIndex(idx int) (DataColumn[T], bool) {
	if idx < 0 || idx >= len(dc.cols) {
		return DataColumn[T]{}, false
	}
	return dc.cols[idx], true
}

// All returns all columns with their index.
func (dc DataColumns[T]) All() iter.Seq2[int, DataColumn[T]] {
	return slices.All(dc.cols)
}

// Size returns the number of columns.
func (dc DataColumns[T]) Size() int {
	return len(dc.cols)
}

// Values return all columns.
func (dc DataColumns[T]) Values() iter.Seq[DataColumn[T]] {
	return slices.Values(dc.cols)
}

// indexByLabel returns the index of the column with the given label.
func (dc DataColumns[T]) indexByLabel(label string) (int, bool) {
	for i, c := range dc.cols {
		if c.Label == label {
			return i, true
		}
	}
	return 0, false
}

// ColumnSorter represents an ordered list of columns which can be sorted.
type ColumnSorter[T any] struct {
	cols       []SortDir
	columns    DataColumns[T]
	initialDir SortDir
	initialIdx int
	isMobile   bool
}

// NewColumnSorter returns a new ColumSorter.
// label and dir defines the initially sorted column.
// It panics if semantic checks fail.
func NewColumnSorter[T any](columns DataColumns[T], label string, dir SortDir) *ColumnSorter[T] {
	idx, ok := columns.indexByLabel(label)
	if !ok {
		dir = SortOff
	}
	if dir == SortOff {
		idx = 0
	}
	cs := &ColumnSorter[T]{
		cols:       make([]SortDir, columns.Size()),
		columns:    columns,
		initialDir: dir,
		initialIdx: idx,
		isMobile:   fyne.CurrentDevice().IsMobile(),
	}
	cs.init()
	cs.setIdx(idx, dir)
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

// // reset sets the columns to their initial state.
// func (cs *ColumnSorter[T]) reset() {
// 	cs.Set(cs.initialID, cs.initialDir)
// }

// Set sets the sort direction for a column. Unknown labels are ignored.
func (cs *ColumnSorter[T]) Set(label string, dir SortDir) {
	idx, ok := cs.columns.indexByLabel(label)
	if !ok {
		return
	}
	cs.setIdx(idx, dir)
}

func (cs *ColumnSorter[T]) setIdx(idx int, dir SortDir) {
	cs.init()
	cs.cols[idx] = dir
}

// func (cs *ColumnSorter[T]) size() int {
// 	return len(cs.cols)
// }

// CalcSort calculates how and if to apply sorting to column idx.
func (cs *ColumnSorter[T]) CalcSort(idx int) (int, SortDir, bool) {
	if idx < 0 || idx >= len(cs.cols) {
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
	return idx, dir, doSort
}

// SortRows sorts the rows.
func (cs *ColumnSorter[T]) SortRows(rows []T, sortCol int, dir SortDir, doSort bool) {
	if !doSort {
		return
	}
	if sortCol < 0 || sortCol >= len(cs.columns.cols) {
		return
	}
	f := cs.columns.cols[sortCol].Sort
	if f == nil {
		return
	}
	slices.SortFunc(rows, func(a, b T) int {
		x := f(a, b)
		if dir == SortAsc {
			return x
		}
		return -1 * x
	})
}

// NewSortChip creates a [SortChip] for a [ColumnSorter].
func (cs *ColumnSorter[T]) NewSortChip(changed func(), ignoredColumns ...string) *kxwidget.SortChip {
	columns := slices.Collect(xiter.Map(cs.columns.Values(), func(h DataColumn[T]) string {
		return h.Label
	}))

	field2Col := make(map[string]int)
	for col, field := range columns {
		field2Col[field] = col
	}
	ignored := set.Of(ignoredColumns...)
	sortColumns := slices.DeleteFunc(columns, func(c string) bool {
		return ignored.Contains(c)
	})
	col, dir := cs.current() // TODO: Hack, replace with defaults
	defaultColumn := cs.columns.cols[col].Label
	defaultOrder := sortOrderFromDir(dir)
	w := kxwidget.NewSortChip(sortColumns, defaultColumn, defaultOrder, func(col string, order kxwidget.SortOrder) {
		cs.setIdx(field2Col[col], sortDirFromOrder(order))
		if changed != nil {
			changed()
		}
	})
	return w
}

func sortDirFromOrder(o kxwidget.SortOrder) SortDir {
	switch o {
	case kxwidget.SortOrderAscending:
		return SortAsc
	case kxwidget.SortOrderDescending:
		return SortDesc
	default:
		return sortNone
	}
}

func sortOrderFromDir(d SortDir) kxwidget.SortOrder {
	if d == SortDesc {
		return kxwidget.SortOrderDescending
	}
	return kxwidget.SortOrderAscending
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
	for idx, col := range columns.cols {
		if col.Update == nil {
			panic(fmt.Sprintf("Column missing update: %d", idx))
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
				return len(*data), columns.Size()
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
				return len(*data), columns.Size()
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
	iconNone := theme.NewThemedResource(iconBlankSvg)
	iconSortOff := theme.NewThemedResource(iconSortSvg)

	t.CreateHeader = func() fyne.CanvasObject {
		return newDataTableHeaderWidget()
	}
	iconMap := map[SortDir]fyne.Resource{
		SortOff:  iconSortOff,
		SortAsc:  theme.NewPrimaryThemedResource(iconSortAscendingSvg),
		SortDesc: theme.NewPrimaryThemedResource(iconSortDescendingSvg),
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		h, ok := columns.ColumnByIndex(tci.Col)
		if !ok {
			return
		}
		headerWidget, ok := co.(*dataTableHeaderWidget)
		if !ok {
			return
		}

		dir := columnSorter.column(tci.Col)
		r, ok := iconMap[dir]
		if !ok {
			r = iconNone
		}

		onTapped := func() {
			filterRows(tci.Col)
		}

		headerWidget.Update(h.Label, dir, r, onTapped)
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

// dataTableHeaderWidget represents a header cell in a data table.
type dataTableHeaderWidget struct {
	widget.BaseWidget

	actionLabel *kxwidget.TappableLabel
	icon        *widget.Icon
	label       *widget.Label
}

func newDataTableHeaderWidget() *dataTableHeaderWidget {
	w := &dataTableHeaderWidget{
		actionLabel: kxwidget.NewTappableLabel("", nil),
		icon:        widget.NewIcon(nil),
		label:       widget.NewLabel(""),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *dataTableHeaderWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, nil, w.icon, container.NewStack(w.actionLabel, w.label))
	return widget.NewSimpleRenderer(c)
}

func (w *dataTableHeaderWidget) Update(labelText string, dir SortDir, iconRes fyne.Resource, onTapped func()) {
	if dir == sortNone {
		w.label.SetText(labelText)
		w.label.Show()
		w.actionLabel.Hide()
		w.icon.Hide()
		return
	}

	w.actionLabel.OnTapped = onTapped
	w.actionLabel.SetText(labelText)
	w.actionLabel.Show()
	w.icon.Show()
	w.label.Hide()

	w.icon.SetResource(iconRes)
}

// MakeDataList returns a list for showing a data table in a generic way.
// This is meant for showing table content on mobile.
func MakeDataList[S ~[]E, E any](
	def DataColumns[E],
	data *S,
	makeCell func(string, E) []widget.RichTextSegment,
	onSelected func(E),
) *widget.List {
	var l *widget.List
	l = widget.NewList(
		func() int {
			return len(*data)
		},
		func() fyne.CanvasObject {
			return newDataCardWidget(def, makeCell)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(*data) || id < 0 {
				return
			}
			item := co.(*dataCardWidget[E])
			item.Update((*data)[id])
			l.SetItemHeight(id, co.(*dataCardWidget[E]).MinSize().Height) // some rows need more height for wrapped text
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
	l.HideSeparators = true
	return l
}

const (
	dataCardRowHighlightColor = theme.ColorNameInputBackground
	dataCardBorderColor       = theme.ColorNameInputBorder
)

// dataCardWidget renders a card with multiple rows from a data row.
// The first row is highlighted.
type dataCardWidget[E any] struct {
	widget.BaseWidget

	border         *canvas.Rectangle
	columns        DataColumns[E]
	makeCell       func(string, E) []widget.RichTextSegment
	maxColumnWidth float32
	rows           []*dataCardRowWidget
}

func newDataCardWidget[E any](columns DataColumns[E], makeCell func(string, E) []widget.RichTextSegment) *dataCardWidget[E] {
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = theme.Color(dataCardBorderColor)
	border.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	border.CornerRadius = theme.Size(theme.SizeNameCardRadius)
	w := &dataCardWidget[E]{
		rows:           make([]*dataCardRowWidget, len(columns.cols)),
		columns:        columns,
		makeCell:       makeCell,
		maxColumnWidth: columns.maxColumnWidth,
		border:         border,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *dataCardWidget[E]) CreateRenderer() fyne.WidgetRenderer {
	width := w.maxColumnWidth + theme.Padding()
	p := theme.Padding()
	rows := container.New(layout.NewCustomPaddedVBoxLayout(0))
	for i := range w.columns.cols {
		rowWidget := newDataCardRowWidget(width)
		rows.Add(rowWidget)
		w.rows[i] = rowWidget
		showDivider := i < w.columns.Size()-1
		if showDivider {
			divider := container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, 2*p), widget.NewSeparator())
			rows.Add(divider)
		}
	}
	rows.Add(NewSpacer(fyne.NewSize(1, p)))
	c := container.NewStack(rows, w.border)
	return widget.NewSimpleRenderer(c)
}

func (w *dataCardWidget[E]) Refresh() {
	w.BaseWidget.Refresh()
	w.border.StrokeColor = theme.Color(dataCardBorderColor)
	w.border.Refresh()

}

func (w *dataCardWidget[E]) Update(r E) {
	for col, item := range w.rows {
		label := w.columns.cols[col].Label
		cell := w.makeCell(label, r)
		isFirst := col == 0

		item.Update(label, cell, isFirst)
	}
}

// dataCardRowWidget renders a row in a data card.
type dataCardRowWidget struct {
	widget.BaseWidget

	bg    *canvas.Rectangle
	cell  *RichText
	label *RichText
	width float32
}

func newDataCardRowWidget(width float32) *dataCardRowWidget {
	label := NewRichText()
	cell := NewRichText()
	cell.Wrapping = fyne.TextWrapWord

	bg := canvas.NewRectangle(theme.Color(dataCardRowHighlightColor))
	bg.Hide()

	w := &dataCardRowWidget{
		bg:    bg,
		cell:  cell,
		label: label,
		width: width,
	}

	w.ExtendBaseWidget(w)
	return w
}

func (w *dataCardRowWidget) CreateRenderer() fyne.WidgetRenderer {
	rowLayout := kxlayout.NewColumns(w.width + theme.Padding())
	row := container.New(rowLayout, w.label, w.cell)
	c := container.NewStack(w.bg, row)
	return widget.NewSimpleRenderer(c)
}

func (w *dataCardRowWidget) Refresh() {
	w.bg.FillColor = theme.Color(dataCardRowHighlightColor)
	w.bg.Refresh()
	w.BaseWidget.Refresh()
}

func (w *dataCardRowWidget) Update(labelText string, cell []widget.RichTextSegment, isFirst bool) {
	if isFirst {
		w.bg.Show()
		for _, s := range cell {
			if x, ok := s.(*widget.TextSegment); ok {
				x.Style.TextStyle.Bold = true
			}
		}
		w.label.Set(cell)
	} else {
		w.cell.Segments = AlignRichTextSegments(fyne.TextAlignTrailing, cell)
		w.label.SetWithText(labelText)
		w.bg.Hide()
	}
	w.cell.Refresh()

}
