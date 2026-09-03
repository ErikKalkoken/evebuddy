package xwidget_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type myRow struct {
	id     int
	planet string
}

func TestDataTable_CreateBasic(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
		Label: "ID",
		Width: 100,
		Sort:  func(a, b myRow) int { return 0 },
		Update: func(r myRow, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(fmt.Sprint(r.id))
		},
	}, {
		Label: "Planet",
		Width: 100,
		Sort:  func(a, b myRow) int { return 0 },
		Update: func(r myRow, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(r.planet)
		},
	}})
	data := []myRow{{3, "Mercury"}, {8, "Venus"}, {42, "Earth"}}
	x := xwidget.MakeDataTable(
		columns,
		&data,
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		xwidget.NewColumnSorter(columns, 0, xwidget.SortAsc),
		func(i int) {},
		nil,
	)
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSquareSize(300))

	test.AssertImageMatches(t, "datatable/basic.png", w.Canvas().Capture())
}

func TestNewDataColumns(t *testing.T) {
	t.Run("can define column", func(t *testing.T) {
		columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
			Label: "Alpha",
		}})
		col, ok := columns.ColumnByIndex(0)
		require.True(t, ok)
		assert.Equal(t, "Alpha", col.Label)
	})
	t.Run("should panic when no cols defined", func(t *testing.T) {
		assert.Panics(t, func() {
			xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{})
		})
	})
}

func TestColumsSorter_CalcSortIdx(t *testing.T) {
	const (
		id1 = 0
		id2 = 1
		id3 = 2
	)
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
		Label: "Alpha",
		Sort: func(a, b myRow) int {
			return 0
		},
	}, {
		Label: "Bravo",
		Sort: func(a, b myRow) int {
			return 0
		},
	}, {
		Label: "Charlie",
	}})
	cases := []struct {
		name       string
		initialID  int
		initialDir xwidget.SortDir
		sortID     int
		wantID     int
		wantDir    xwidget.SortDir
		wantSort   bool
	}{
		{
			name:       "initial sort, asc->desc",
			initialID:  id1,
			initialDir: xwidget.SortAsc,
			sortID:     id1,
			wantID:     id1,
			wantDir:    xwidget.SortDesc,
			wantSort:   true,
		},
		{
			name:       "initial sort, desc->asc",
			initialID:  id1,
			initialDir: xwidget.SortDesc,
			sortID:     id1,
			wantID:     id1,
			wantDir:    xwidget.SortAsc,
			wantSort:   true,
		},
		{
			name:       "initial sort, none->asc",
			initialID:  id1,
			initialDir: xwidget.SortOff,
			sortID:     id1,
			wantID:     id1,
			wantDir:    xwidget.SortAsc,
			wantSort:   true,
		},
		{
			name:       "initial sort, don't sort",
			initialID:  id1,
			initialDir: xwidget.SortOff,
			sortID:     -1,
			wantSort:   false,
		},
		{
			name:       "initial sort, sort diabled",
			initialID:  id1,
			initialDir: xwidget.SortOff,
			sortID:     id3,
			wantSort:   false,
		},
		{
			name:       "initial sort 2, asc->desc",
			initialID:  id2,
			initialDir: xwidget.SortAsc,
			sortID:     id2,
			wantID:     id2,
			wantDir:    xwidget.SortDesc,
			wantSort:   true,
		},
		{
			name:       "initial no sort, asc->desc",
			initialID:  -1,
			initialDir: xwidget.SortOff,
			sortID:     id2,
			wantID:     id2,
			wantDir:    xwidget.SortAsc,
			wantSort:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := xwidget.NewColumnSorter(columns, tc.initialID, tc.initialDir)
			gotID, gotDir, gotSort := sc.CalcSort(tc.sortID)
			assert.Equal(t, tc.wantSort, gotSort)
			if tc.wantSort {
				assert.Equal(t, tc.wantID, gotID)
				assert.Equal(t, tc.wantDir, gotDir)
			}
		})
	}
}
