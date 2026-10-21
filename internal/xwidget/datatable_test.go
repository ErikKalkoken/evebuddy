package xwidget_test

import (
	"cmp"
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
		func(label string) {},
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
	t.Run("should panic when a label is duplicated", func(t *testing.T) {
		assert.Panics(t, func() {
			xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
				Label: "Alpha",
			}, {
				Label: "Alpha",
			}})
		})
	})
}

func TestColumsSorter_CalcSortIdx(t *testing.T) {
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
		sortLabel    string
		wantLabel    string
		wantDir      xwidget.SortDir
		wantSort     bool
	}{
		{
			name:         "initial sort, asc->desc",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortAsc,
			sortLabel:    "Alpha",
			wantLabel:    "Alpha",
			wantDir:      xwidget.SortDesc,
			wantSort:     true,
		},
		{
			name:         "initial sort, desc->asc",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortDesc,
			sortLabel:    "Alpha",
			wantLabel:    "Alpha",
			wantDir:      xwidget.SortAsc,
			wantSort:     true,
		},
		{
			name:         "initial sort, none->asc",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortOff,
			sortLabel:    "Alpha",
			wantLabel:    "Alpha",
			wantDir:      xwidget.SortAsc,
			wantSort:     true,
		},
		{
			name:         "initial sort, don't sort",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortOff,
			sortLabel:    "",
			wantSort:     false,
		},
		{
			name:         "initial sort, sort diabled",
			initialLabel: "Alpha",
			initialDir:   xwidget.SortOff,
			sortLabel:    "Charlie",
			wantSort:     false,
		},
		{
			name:         "initial sort 2, asc->desc",
			initialLabel: "Bravo",
			initialDir:   xwidget.SortAsc,
			sortLabel:    "Bravo",
			wantLabel:    "Bravo",
			wantDir:      xwidget.SortDesc,
			wantSort:     true,
		},
		{
			name:         "initial no sort, asc->desc",
			initialLabel: "Nonexistent",
			initialDir:   xwidget.SortOff,
			sortLabel:    "Bravo",
			wantLabel:    "Bravo",
			wantDir:      xwidget.SortAsc,
			wantSort:     true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := xwidget.NewColumnSorter(columns, tc.initialLabel, tc.initialDir)
			gotLabel, gotDir, gotSort := sc.CalcSort(tc.sortLabel)
			assert.Equal(t, tc.wantSort, gotSort)
			if tc.wantSort {
				assert.Equal(t, tc.wantLabel, gotLabel)
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

	t.Run("does not panic when no column is currently sorted", func(t *testing.T) {
		sc := xwidget.NewColumnSorter(columns, "Nonexistent", xwidget.SortAsc)
		var chip *kxwidget.SortChip
		assert.NotPanics(t, func() {
			chip = sc.NewSortChip(nil)
		})
		assert.Equal(t, "", chip.DefaultColumn)
	})

	t.Run("selecting a column updates the sorter and calls changed", func(t *testing.T) {
		sc := xwidget.NewColumnSorter(columns, "Alpha", xwidget.SortAsc)
		var called bool
		chip := sc.NewSortChip(func() { called = true })
		chip.OnChanged("Charlie", kxwidget.SortOrderDescending)
		gotLabel, gotDir, gotSort := sc.CalcSort("")
		assert.True(t, gotSort)
		assert.Equal(t, "Charlie", gotLabel)
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

func TestColumnSorter_CalcSort_Sequences(t *testing.T) {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
		Label: "Alpha",
		Sort:  func(a, b myRow) int { return 0 },
	}, {
		Label: "Bravo",
		Sort:  func(a, b myRow) int { return 0 },
	}, {
		Label: "Charlie",
	}})
	t.Run("empty label keeps current sort without toggling", func(t *testing.T) {
		// given
		sc := xwidget.NewColumnSorter(columns, "Alpha", xwidget.SortDesc)
		sc.CalcSort("")
		// when
		gotLabel, gotDir, gotSort := sc.CalcSort("")
		// then
		assert.True(t, gotSort)
		assert.Equal(t, "Alpha", gotLabel)
		assert.Equal(t, xwidget.SortDesc, gotDir)
	})
	t.Run("repeated taps cycle between ascending and descending", func(t *testing.T) {
		// given
		sc := xwidget.NewColumnSorter(columns, "Alpha", xwidget.SortOff)
		// when
		_, dir1, _ := sc.CalcSort("Alpha")
		_, dir2, _ := sc.CalcSort("Alpha")
		_, dir3, _ := sc.CalcSort("Alpha")
		// then
		assert.Equal(t, xwidget.SortAsc, dir1)
		assert.Equal(t, xwidget.SortDesc, dir2)
		assert.Equal(t, xwidget.SortAsc, dir3)
	})
	t.Run("tapping another column resets the previous one", func(t *testing.T) {
		// given
		sc := xwidget.NewColumnSorter(columns, "Alpha", xwidget.SortAsc)
		sc.CalcSort("Bravo")
		// when
		gotLabel, gotDir, gotSort := sc.CalcSort("Alpha")
		// then
		assert.True(t, gotSort)
		assert.Equal(t, "Alpha", gotLabel)
		assert.Equal(t, xwidget.SortAsc, gotDir)
	})
	t.Run("tapping an unsortable column keeps the current sort", func(t *testing.T) {
		// given
		sc := xwidget.NewColumnSorter(columns, "Alpha", xwidget.SortDesc)
		sc.CalcSort("Charlie")
		// when
		gotLabel, gotDir, gotSort := sc.CalcSort("")
		// then
		assert.True(t, gotSort)
		assert.Equal(t, "Alpha", gotLabel)
		assert.Equal(t, xwidget.SortDesc, gotDir)
	})
}

func TestColumnSorter_SortRows(t *testing.T) {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
		Label: "ID",
		Sort:  func(a, b myRow) int { return cmp.Compare(a.id, b.id) },
	}, {
		Label: "Planet",
	}})
	sc := xwidget.NewColumnSorter(columns, "ID", xwidget.SortAsc)
	t.Run("sorts ascending", func(t *testing.T) {
		// given
		rows := []myRow{{8, "Venus"}, {3, "Mercury"}, {42, "Earth"}}
		// when
		sc.SortRows(rows, "ID", xwidget.SortAsc, true)
		// then
		assert.Equal(t, []myRow{{3, "Mercury"}, {8, "Venus"}, {42, "Earth"}}, rows)
	})
	t.Run("sorts descending", func(t *testing.T) {
		// given
		rows := []myRow{{8, "Venus"}, {3, "Mercury"}, {42, "Earth"}}
		// when
		sc.SortRows(rows, "ID", xwidget.SortDesc, true)
		// then
		assert.Equal(t, []myRow{{42, "Earth"}, {8, "Venus"}, {3, "Mercury"}}, rows)
	})
	t.Run("does not sort when doSort is false", func(t *testing.T) {
		// given
		rows := []myRow{{8, "Venus"}, {3, "Mercury"}, {42, "Earth"}}
		// when
		sc.SortRows(rows, "ID", xwidget.SortAsc, false)
		// then
		assert.Equal(t, []myRow{{8, "Venus"}, {3, "Mercury"}, {42, "Earth"}}, rows)
	})
	t.Run("does not sort for an unknown column", func(t *testing.T) {
		// given
		rows := []myRow{{8, "Venus"}, {3, "Mercury"}, {42, "Earth"}}
		// when
		sc.SortRows(rows, "Zulu", xwidget.SortAsc, true)
		// then
		assert.Equal(t, []myRow{{8, "Venus"}, {3, "Mercury"}, {42, "Earth"}}, rows)
	})
	t.Run("does not sort for a column without sort func", func(t *testing.T) {
		// given
		rows := []myRow{{8, "Venus"}, {3, "Mercury"}, {42, "Earth"}}
		// when
		sc.SortRows(rows, "Planet", xwidget.SortAsc, true)
		// then
		assert.Equal(t, []myRow{{8, "Venus"}, {3, "Mercury"}, {42, "Earth"}}, rows)
	})
}

func TestDataTable_OnSelected(t *testing.T) {
	test.NewTempApp(t)
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
		Label:  "ID",
		Update: func(r myRow, co fyne.CanvasObject) {},
	}, {
		Label:  "Planet",
		Update: func(r myRow, co fyne.CanvasObject) {},
	}})
	t.Run("reports column and row of selected cell", func(t *testing.T) {
		// given
		data := []myRow{{3, "Mercury"}, {8, "Venus"}, {42, "Earth"}}
		var gotCol int
		var gotRow myRow
		x := xwidget.MakeDataTable(
			columns,
			&data,
			func() fyne.CanvasObject { return widget.NewLabel("") },
			xwidget.NewColumnSorter(columns, "ID", xwidget.SortOff),
			func(string) {},
			func(col int, r myRow) {
				gotCol = col
				gotRow = r
			},
		)
		// when
		x.Select(widget.TableCellID{Row: 1, Col: 1})
		// then
		assert.Equal(t, 1, gotCol)
		assert.Equal(t, myRow{8, "Venus"}, gotRow)
	})
	t.Run("same cell can be selected again", func(t *testing.T) {
		// given
		data := []myRow{{3, "Mercury"}, {8, "Venus"}, {42, "Earth"}}
		var calls int
		x := xwidget.MakeDataTable(
			columns,
			&data,
			func() fyne.CanvasObject { return widget.NewLabel("") },
			xwidget.NewColumnSorter(columns, "ID", xwidget.SortOff),
			func(string) {},
			func(int, myRow) { calls++ },
		)
		x.Select(widget.TableCellID{Row: 1, Col: 1})
		// when
		x.Select(widget.TableCellID{Row: 1, Col: 1})
		// then
		assert.Equal(t, 2, calls)
	})
	t.Run("ignores rows no longer in data", func(t *testing.T) {
		// given
		data := []myRow{{3, "Mercury"}}
		var calls int
		x := xwidget.MakeDataTable(
			columns,
			&data,
			func() fyne.CanvasObject { return widget.NewLabel("") },
			xwidget.NewColumnSorter(columns, "ID", xwidget.SortOff),
			func(string) {},
			func(int, myRow) { calls++ },
		)
		// when
		x.OnSelected(widget.TableCellID{Row: 5, Col: 0})
		// then
		assert.Equal(t, 0, calls)
	})
	t.Run("does not panic without callback", func(t *testing.T) {
		// given
		data := []myRow{{3, "Mercury"}}
		x := xwidget.MakeDataTable(
			columns,
			&data,
			func() fyne.CanvasObject { return widget.NewLabel("") },
			xwidget.NewColumnSorter(columns, "ID", xwidget.SortOff),
			func(string) {},
			nil,
		)
		// when/then
		assert.NotPanics(t, func() {
			x.Select(widget.TableCellID{Row: 0, Col: 0})
		})
	})
}

func TestDataTable_CustomCreate(t *testing.T) {
	test.NewTempApp(t)
	// given
	var got []fyne.CanvasObject
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
		Label:  "Done",
		Create: func() fyne.CanvasObject { return widget.NewCheck("", nil) },
		Update: func(r myRow, co fyne.CanvasObject) { got = append(got, co) },
	}, {
		Label:  "Planet",
		Update: func(r myRow, co fyne.CanvasObject) { got = append(got, co) },
	}, {
		Label:  "Progress",
		Create: func() fyne.CanvasObject { return widget.NewProgressBar() },
		Update: func(r myRow, co fyne.CanvasObject) { got = append(got, co) },
	}})
	data := []myRow{{3, "Mercury"}}
	x := xwidget.MakeDataTable(
		columns,
		&data,
		func() fyne.CanvasObject { return widget.NewLabel("") },
		xwidget.NewColumnSorter(columns, "Planet", xwidget.SortOff),
		func(string) {},
		nil,
	)
	cases := []struct {
		col  int
		want func(fyne.CanvasObject) bool
	}{
		{0, func(co fyne.CanvasObject) bool { _, ok := co.(*widget.Check); return ok }},
		{1, func(co fyne.CanvasObject) bool { _, ok := co.(*widget.Label); return ok }},
		{2, func(co fyne.CanvasObject) bool { _, ok := co.(*widget.ProgressBar); return ok }},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("column %d gets its own object", tc.col), func(t *testing.T) {
			// given
			got = nil
			co := x.CreateCell()
			// when
			x.UpdateCell(widget.TableCellID{Row: 0, Col: tc.col}, co)
			// then
			require.Len(t, got, 1)
			assert.True(t, tc.want(got[0]))
			var visible []fyne.CanvasObject
			for _, o := range co.(*fyne.Container).Objects {
				if o.Visible() {
					visible = append(visible, o)
				}
			}
			assert.Equal(t, []fyne.CanvasObject{got[0]}, visible)
		})
	}
}

func TestDataTable_Panics(t *testing.T) {
	test.NewTempApp(t)
	t.Run("when default create is missing", func(t *testing.T) {
		columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
			Label:  "ID",
			Update: func(r myRow, co fyne.CanvasObject) {},
		}})
		data := []myRow{}
		assert.Panics(t, func() {
			xwidget.MakeDataTable(columns, &data, nil, xwidget.NewColumnSorter(columns, "ID", xwidget.SortOff), func(string) {}, nil)
		})
	})
	t.Run("when a column has no update", func(t *testing.T) {
		columns := xwidget.NewDataColumns([]xwidget.DataColumn[myRow]{{
			Label: "ID",
		}})
		data := []myRow{}
		assert.Panics(t, func() {
			xwidget.MakeDataTable(
				columns,
				&data,
				func() fyne.CanvasObject { return widget.NewLabel("") },
				xwidget.NewColumnSorter(columns, "ID", xwidget.SortOff),
				func(string) {},
				nil,
			)
		})
	})
}
