package widget

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/stretchr/testify/assert"
)

type myRow struct {
	id     int
	planet string
}

func TestDataTable_CreateBasic(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	headers := NewDataTableDef([]ColumnDef{
		{Label: "ID", Width: 100},
		{Label: "Planet", Width: 100},
	})
	data := []myRow{{3, "Mercury"}, {8, "Venus"}, {42, "Earth"}}
	x := MakeDataTable(
		headers, &data, func(col int, r myRow) []widget.RichTextSegment {
			switch col {
			case 0:
				return RichTextSegmentsFromText(fmt.Sprint(r.id))
			case 1:
				return RichTextSegmentsFromText(r.planet)
			}
			panic(fmt.Sprintf("invalid col: %d", col))
		},
		NewColumnSorterWithInit(headers, 0, SortAsc),
		func(i int) {

		},
		nil,
	)
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSquareSize(300))

	test.AssertImageMatches(t, "datatable/basic.png", w.Canvas().Capture())
}

func TestSortedColumsColumn(t *testing.T) {
	headers := NewDataTableDef([]ColumnDef{
		{Label: "Alpha"},
		{Label: "Bravo"},
		{Label: "Charlie"},
	})
	t.Run("return value", func(t *testing.T) {
		sc := NewColumnSorter(headers)
		sc.Set(1, SortDesc)
		got := sc.column(1)
		assert.Equal(t, SortDesc, got)
	})
	t.Run("out of bounds returns zero value 1", func(t *testing.T) {
		sc := NewColumnSorter(headers)
		got := sc.column(4)
		assert.Equal(t, SortOff, got)
	})
	t.Run("out of bounds returns zero value 2", func(t *testing.T) {
		sc := NewColumnSorter(headers)
		got := sc.column(-1)
		assert.Equal(t, SortOff, got)
	})
}

func TestSortedColumsCurrent(t *testing.T) {
	headers := NewDataTableDef([]ColumnDef{
		{Label: "Alpha"},
		{Label: "Bravo"},
		{Label: "Charlie"},
	})
	t.Run("return currently sorted column", func(t *testing.T) {
		sc := NewColumnSorter(headers)
		sc.Set(1, SortDesc)
		x, y := sc.current()
		assert.Equal(t, 1, x)
		assert.Equal(t, SortDesc, y)
	})
	t.Run("return -1 if nothing set", func(t *testing.T) {
		sc := NewColumnSorter(headers)
		x, y := sc.current()
		assert.Equal(t, -1, x)
		assert.Equal(t, SortOff, y)
	})
}

// func TestSortedColumsCycleColumn(t *testing.T) {
// 	cases := []struct {
// 		name    string
// 		col     int
// 		dir     sortDir
// 		wantCol int
// 		wantDir sortDir
// 	}{
// 		{"", 2, sortOff, 2, sortAsc},
// 		{"", 2, sortAsc, 2, sortDesc},
// 		{"", 2, sortDesc, -1, sortOff},
// 	}
// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			sc := newColumnSorter(headers)
// 			sc.set(tc.col, tc.dir)
// 			got := sc.cycleColumn(tc.col)
// 			assert.Equal(t, tc.wantDir, got)
// 			x, y := sc.current()
// 			assert.Equal(t, tc.wantCol, x)
// 			assert.Equal(t, tc.wantDir, y)
// 		})
// 	}
// }
