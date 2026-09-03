package xwidget_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
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
		xwidget.NewColumnSorter(columns, "ID", xwidget.SortAsc),
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
		name         string
		initialLabel string
		initialDir   xwidget.SortDir
		sortID       int
		wantID       int
		wantDir      xwidget.SortDir
		wantSort     bool
	}{
		{
			name:         "initial sort, asc->desc",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortAsc,
			sortID:       id1,
			wantID:       id1,
			wantDir:      xwidget.SortDesc,
			wantSort:     true,
		},
		{
			name:         "initial sort, desc->asc",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortDesc,
			sortID:       id1,
			wantID:       id1,
			wantDir:      xwidget.SortAsc,
			wantSort:     true,
		},
		{
			name:         "initial sort, none->asc",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortOff,
			sortID:       id1,
			wantID:       id1,
			wantDir:      xwidget.SortAsc,
			wantSort:     true,
		},
		{
			name:         "initial sort, don't sort",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortOff,
			sortID:       -1,
			wantSort:     false,
		},
		{
			name:         "initial sort, sort diabled",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortOff,
			sortID:       id3,
			wantSort:     false,
		},
		{
			name:         "initial sort 2, asc->desc",
			initialLabel: "Bravo",
			initialDir:   xwidget.SortAsc,
			sortID:       id2,
			wantID:       id2,
			wantDir:      xwidget.SortDesc,
			wantSort:     true,
		},
		{
			name:         "initial no sort, asc->desc",
			initialLabel: "Nonexistent",
			initialDir:   xwidget.SortOff,
			sortID:       id2,
			wantID:       id2,
			wantDir:      xwidget.SortAsc,
			wantSort:     true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := xwidget.NewColumnSorter(columns, tc.initialLabel, tc.initialDir)
			gotID, gotDir, gotSort := sc.CalcSort(tc.sortID)
			assert.Equal(t, tc.wantSort, gotSort)
			if tc.wantSort {
				assert.Equal(t, tc.wantID, gotID)
				assert.Equal(t, tc.wantDir, gotDir)
			}
		})
	}
}

func TestColumnSorter_NewSortChip(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
		Label: "Alpha",
		Sort:  func(a, b myRow) int { return 0 },
	}, {
		Label: "Bravo",
		Sort:  func(a, b myRow) int { return 0 },
	}, {
		Label: "Charlie",
		Sort:  func(a, b myRow) int { return 0 },
	}})

	t.Run("sets default column and order from current sort", func(t *testing.T) {
		sc := xwidget.NewColumnSorter(columns, "Bravo", xwidget.SortDesc)
		chip := sc.NewSortChip(nil)
		assert.Equal(t, "Bravo", chip.DefaultColumn)
		assert.Equal(t, "Bravo", chip.Column)
		assert.Equal(t, kxwidget.SortOrderDescending, chip.Order)
	})

	t.Run("selecting a column updates the sorter and calls changed", func(t *testing.T) {
		sc := xwidget.NewColumnSorter(columns, "Alpha", xwidget.SortAsc)
		var called bool
		chip := sc.NewSortChip(func() { called = true })
		chip.OnChanged("Charlie", kxwidget.SortOrderDescending)
		gotID, gotDir, gotSort := sc.CalcSort(-1)
		assert.True(t, gotSort)
		assert.Equal(t, 2, gotID)
		assert.Equal(t, xwidget.SortDesc, gotDir)
		assert.True(t, called)
	})

	t.Run("offers all columns when none are ignored", func(t *testing.T) {
		sc := xwidget.NewColumnSorter(columns, "Alpha", xwidget.SortAsc)
		chip := sc.NewSortChip(nil)
		w := test.NewWindow(chip)
		defer w.Close()
		w.Resize(fyne.NewSize(300, 400))

		test.Tap(chip)

		assert.NotNil(t, findMenuItem(w, "Alpha"))
		assert.NotNil(t, findMenuItem(w, "Bravo"))
		assert.NotNil(t, findMenuItem(w, "Charlie"))
	})

	t.Run("excludes ignored columns from the menu", func(t *testing.T) {
		sc := xwidget.NewColumnSorter(columns, "Alpha", xwidget.SortAsc)
		chip := sc.NewSortChip(nil, "Bravo", "Charlie")
		w := test.NewWindow(chip)
		defer w.Close()
		w.Resize(fyne.NewSize(300, 400))

		test.Tap(chip)

		assert.NotNil(t, findMenuItem(w, "Alpha"))
		assert.Nil(t, findMenuItem(w, "Bravo"))
		assert.Nil(t, findMenuItem(w, "Charlie"))
	})
}

// findMenuItem searches the topmost canvas overlay of w for a text object matching label.
func findMenuItem(w fyne.Window, label string) fyne.CanvasObject {
	overlay := w.Canvas().Overlays().Top()
	if overlay == nil {
		return nil
	}
	return findObjectByText(overlay, label)
}

func findObjectByText(obj fyne.CanvasObject, want string) fyne.CanvasObject {
	switch v := obj.(type) {
	case *widget.Label:
		if v.Text == want {
			return v
		}
	case *canvas.Text:
		if v.Text == want {
			return v
		}
	}
	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := findObjectByText(child, want); found != nil {
				return found
			}
		}
	}
	if wd, ok := obj.(fyne.Widget); ok {
		for _, child := range test.WidgetRenderer(wd).Objects() {
			if found := findObjectByText(child, want); found != nil {
				return found
			}
		}
	}
	return nil
}
