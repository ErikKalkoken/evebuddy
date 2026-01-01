package widget

import (
	"cmp"
	"fmt"
	"iter"
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

type (
	// ColumnDef represents the definition for a column in a data table.
	ColumnDef struct {
		// Column index starting at 0. MANDATORY.
		Col int
		// Label of a column displayed to the user. MANDATORY.
		Label string
		// Whether a column is sortable.
		NoSort bool
		// Width of a column in Fyne units. Will try to auto size when zero.
		Width float32
	}

	// DataTableDef represents the definition for a data table.
	DataTableDef struct {
		cols []ColumnDef
	}

	// ColumnSorter represents an ordered list of columns which can be sorted.
	ColumnSorter struct {
		cols       []SortDir
		def        DataTableDef
		initialDir SortDir
		initialIdx int
		isMobile   bool
		sortButton *SortButton
	}

	// A SortButton represents a button for sorting a data table.
	// It is supposed to be used in mobile views.
	SortButton struct {
		widget.Button

		sortColumns []string
	}
)

func (h ColumnDef) minWidth() float32 {
	if h.Width > 0 {
		return h.Width
	}
	x := widget.NewLabel(h.Label)
	return x.MinSize().Width
}

// NewDataTableDef creates and returns a [DataTableDef].
func NewDataTableDef(cols []ColumnDef) DataTableDef {
	var incoming, expected set.Set[int]
	for i := range len(cols) {
		expected.Add(i)
	}
	for _, c := range cols {
		if c.Label == "" {
			panic("label not defined")
		}
		if !expected.Contains(c.Col) {
			panic(fmt.Sprintf("%s: col must be in [%d, %d]", c.Label, 0, len(cols)-1))
		}
		if incoming.Contains(c.Col) {
			panic(fmt.Sprintf("%s: col %d duplicate", c.Label, c.Col))
		}
		incoming.Add(c.Col)
	}
	cols2 := slices.Clone(cols)
	slices.SortFunc(cols2, func(a, b ColumnDef) int {
		return cmp.Compare(a.Col, b.Col)
	})
	d := DataTableDef{
		cols: cols2,
	}
	return d
}

// Column return the definition of a column.
func (d DataTableDef) Column(n int) ColumnDef {
	return d.cols[n]
}

func (d DataTableDef) all() iter.Seq2[int, ColumnDef] {
	return slices.All(d.cols)
}

// maxColumnWidth returns the maximum width of any column.
func (d DataTableDef) maxColumnWidth() float32 {
	var m float32
	for _, c := range d.cols {
		l := widget.NewLabel(c.Label)
		m = max(l.MinSize().Width, m)
	}
	return m
}

func (d DataTableDef) size() int {
	return len(d.cols)
}

func (d DataTableDef) values() iter.Seq[ColumnDef] {
	return slices.Values(d.cols)
}

// NewColumnSorter creates and returns a new [ColumSorter].
// idx and dir defines the initially sorted column.
func (d DataTableDef) NewColumnSorter(idx int, dir SortDir) *ColumnSorter {
	if idx < 0 || idx >= d.size() {
		panic(fmt.Sprintf("invalid idx. Allowed range: [0, %d]", d.size()-1))
	}
	cs := &ColumnSorter{
		cols:       make([]SortDir, d.size()),
		def:        d,
		initialDir: dir,
		initialIdx: idx,
		isMobile:   fyne.CurrentDevice().IsMobile(),
	}
	cs.clear()
	cs.Set(idx, dir)
	return cs
}

// column returns the sort direction of a column
func (cs *ColumnSorter) column(idx int) SortDir {
	if idx < 0 || idx >= len(cs.cols) {
		return SortOff
	}
	return cs.cols[idx]
}

// current returns which column is currently sorted or -1 if none are sorted.
func (cs *ColumnSorter) current() (int, SortDir) {
	for i, v := range cs.cols {
		if v.isSorting() {
			return i, v
		}
	}
	return -1, SortOff
}

// reset sets the columns to their initial state.
func (cs *ColumnSorter) reset() {
	cs.Set(cs.initialIdx, cs.initialDir)
}

// clear removes sorting from all columns.
func (cs *ColumnSorter) clear() {
	for i := range cs.cols {
		var dir SortDir
		if cs.def.Column(i).NoSort {
			dir = sortNone
		} else {
			dir = SortOff
		}
		cs.cols[i] = dir
	}
}

// Set sets the sort direction for a column.
func (cs *ColumnSorter) Set(idx int, dir SortDir) {
	cs.clear()
	cs.cols[idx] = dir
	if cs.sortButton != nil {
		cs.sortButton.set(idx, dir)
	}
}

// current returns which column is currently sorted or -1 if none are sorted.
func (cs *ColumnSorter) size() int {
	return len(cs.cols)
}

// Sort sorts columns idx by applying function f.
// It will re-apply the previous sort when idx is -1.
func (cs *ColumnSorter) Sort(idx int, f func(sortCol int, dir SortDir)) {
	var dir SortDir
	if idx >= 0 {
		dir = cs.cols[idx]
		if dir == sortNone {
			return
		}
		dir++
		if dir > SortDesc {
			dir = SortAsc
		}
		cs.Set(idx, dir)
	} else {
		idx, dir = cs.current()
	}
	if idx >= 0 && dir.isSorting() {
		f(idx, dir)
	}
}

// NewSortButton returns a new sortButton.
func (cs *ColumnSorter) NewSortButton(process func(), window fyne.Window, ignoredColumns ...int) *SortButton {
	sortColumns := slices.Collect(xiter.Map(cs.def.values(), func(h ColumnDef) string {
		return h.Label
	}))
	w := &SortButton{
		sortColumns: sortColumns,
	}
	w.ExtendBaseWidget(w)
	w.Text = "???"
	w.Icon = icons.BlankSvg
	if cs.def.size() == 0 || cs.size() == 0 || len(ignoredColumns) > cs.def.size() {
		slog.Warn("makeSortButton called with invalid parameters")
		return w // early exit when called without proper data
	}
	ignored := set.Of(ignoredColumns...)
	w.OnTapped = func() {
		col, dir := cs.current()
		var fields []string
		for i, h := range cs.def.all() {
			if !h.NoSort && !ignored.Contains(i) {
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
			cs.Set(col, dir)
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

func (w *SortButton) set(col int, dir SortDir) {
	switch dir {
	case SortAsc:
		w.Icon = theme.NewThemedResource(icons.SortAscendingSvg)
	case SortDesc:
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

// MakeDataTable returns a data table generated from the definition.
func MakeDataTable[S ~[]E, E any](
	def DataTableDef,
	data *S,
	makeCell func(int, E) []widget.RichTextSegment,
	columnSorter *ColumnSorter,
	filterRows func(int),
	onSelected func(int, E),
) *widget.Table {
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(*data), def.size()
		},
		func() fyne.CanvasObject {
			return NewRichText()
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			cell := co.(*RichText)
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
	iconMap := map[SortDir]fyne.Resource{
		SortOff:  iconSortOff,
		SortAsc:  theme.NewPrimaryThemedResource(icons.SortAscendingSvg),
		SortDesc: theme.NewPrimaryThemedResource(icons.SortDescendingSvg),
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		h := def.Column(tci.Col)
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
			filterRows(tci.Col)
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
	for i, h := range def.cols {
		t.SetColumnWidth(i, h.minWidth()+w)
	}
	return t
}

// MakeDataList returns a list for showing a data table in a generic way.
// This is meant for showing table content on mobile.
func MakeDataList[S ~[]E, E any](
	def DataTableDef,
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
				data := row[1].(*RichText)
				data.Segments = AlignRichTextSegments(fyne.TextAlignTrailing, makeCell(col, r))
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
