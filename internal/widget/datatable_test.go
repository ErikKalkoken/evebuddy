package widget_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type myRow struct {
	id     int
	planet string
}

func TestDataTable_CreateBasic(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	headers := iwidget.NewDataColumns([]iwidget.DataColumn[myRow]{{
		ID:    0,
		Label: "ID",
		Width: 100,
	}, {
		ID:    1,
		Label: "Planet",
		Width: 100,
	}})
	data := []myRow{{3, "Mercury"}, {8, "Venus"}, {42, "Earth"}}
	x := iwidget.MakeDataTable(
		headers, &data, func(col int, r myRow) []widget.RichTextSegment {
			switch col {
			case 0:
				return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.id))
			case 1:
				return iwidget.RichTextSegmentsFromText(r.planet)
			}
			panic(fmt.Sprintf("invalid col: %d", col))
		},
		iwidget.NewColumnSorter(headers, 0, iwidget.SortAsc),
		func(i int) {

		},
		nil,
	)
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSquareSize(300))

	test.AssertImageMatches(t, "datatable/basic.png", w.Canvas().Capture())
}

func TestNewDataColumns(t *testing.T) {
	t.Run("can define column", func(t *testing.T) {
		def := iwidget.NewDataColumns([]iwidget.DataColumn[myRow]{{
			ID:    0,
			Label: "Alpha",
		}})
		col, ok := def.ColumnByIndex(0)
		require.True(t, ok)
		assert.Equal(t, "Alpha", col.Label)
	})
	t.Run("should panic when col ID is negativ", func(t *testing.T) {
		assert.Panics(t, func() {
			iwidget.NewDataColumns([]iwidget.DataColumn[myRow]{{
				ID:    -1,
				Label: "Alpha",
			}})
		})
	})
	t.Run("should panic when col index is defined more then once", func(t *testing.T) {
		assert.Panics(t, func() {
			iwidget.NewDataColumns([]iwidget.DataColumn[myRow]{{
				ID:    0,
				Label: "Alpha",
			}, {
				ID:    0,
				Label: "Bravo",
			}})
		})
	})
	t.Run("should panic when no cols defined", func(t *testing.T) {
		assert.Panics(t, func() {
			iwidget.NewDataColumns([]iwidget.DataColumn[myRow]{})
		})
	})
}

func TestColumsSorter_CalcSortIdx(t *testing.T) {
	const (
		id1 = 99
		id2 = 5
		id3 = 21
	)
	def := iwidget.NewDataColumns([]iwidget.DataColumn[myRow]{{
		ID:    id1,
		Label: "Alpha",
	}, {
		ID:    id2,
		Label: "Bravo",
	}, {
		ID:     id3,
		Label:  "Charlie",
		NoSort: true,
	}})
	cases := []struct {
		name       string
		initialID  int
		initialDir iwidget.SortDir
		sortID     int
		wantID     int
		wantDir    iwidget.SortDir
		wantSort   bool
	}{
		{
			name:       "initial sort, asc->desc",
			initialID:  id1,
			initialDir: iwidget.SortAsc,
			sortID:     id1,
			wantID:     id1,
			wantDir:    iwidget.SortDesc,
			wantSort:   true,
		},
		{
			name:       "initial sort, desc->asc",
			initialID:  id1,
			initialDir: iwidget.SortDesc,
			sortID:     id1,
			wantID:     id1,
			wantDir:    iwidget.SortAsc,
			wantSort:   true,
		},
		{
			name:       "initial sort, none->asc",
			initialID:  id1,
			initialDir: iwidget.SortOff,
			sortID:     id1,
			wantID:     id1,
			wantDir:    iwidget.SortAsc,
			wantSort:   true,
		},
		{
			name:       "initial sort, don't sort",
			initialID:  id1,
			initialDir: iwidget.SortOff,
			sortID:     -1,
			wantSort:   false,
		},
		{
			name:       "initial sort, sort diabled",
			initialID:  id1,
			initialDir: iwidget.SortOff,
			sortID:     id3,
			wantSort:   false,
		},
		{
			name:       "initial sort 2, asc->desc",
			initialID:  id2,
			initialDir: iwidget.SortAsc,
			sortID:     id2,
			wantID:     id2,
			wantDir:    iwidget.SortDesc,
			wantSort:   true,
		},
		{
			name:       "initial no sort, asc->desc",
			initialID:  0,
			initialDir: iwidget.SortOff,
			sortID:     id2,
			wantID:     id2,
			wantDir:    iwidget.SortAsc,
			wantSort:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := iwidget.NewColumnSorter(def, tc.initialID, tc.initialDir)
			gotID, gotDir, gotSort := sc.CalcSort(tc.sortID)
			assert.Equal(t, tc.wantSort, gotSort)
			if tc.wantSort {
				assert.Equal(t, tc.wantID, gotID)
				assert.Equal(t, tc.wantDir, gotDir)
			}
		})
	}
}
