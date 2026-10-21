package xwidget

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColumnSorter_New(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		Label: "Alpha",
	}, {
		Label: "Bravo",
	}})
	t.Run("should create normally", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortAsc)
		got := sc.column(0)
		assert.Equal(t, SortAsc, got)
	})
}

func TestColumnSorter_Column(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		Label: "Alpha",
	}, {
		Label: "Bravo",
	}, {
		Label: "Charlie",
	}})
	t.Run("return value", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		sc.Set("Bravo", SortDesc)
		got := sc.column(1)
		assert.Equal(t, SortDesc, got)
	})
	t.Run("out of bounds returns zero value 1", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		got := sc.column(4)
		assert.Equal(t, SortOff, got)
	})
	t.Run("out of bounds returns zero value 2", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		got := sc.column(-1)
		assert.Equal(t, SortOff, got)
	})
}

func TestColumnSorter_Set(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		Label: "Alpha",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}, {
		Label: "Bravo",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}, {
		Label: "Charlie",
	}})
	t.Run("sets the sort direction for the named column", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		sc.Set("Bravo", SortDesc)
		assert.Equal(t, SortDesc, sc.column(1))
	})
	t.Run("resets other columns when a new one is set", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortAsc)
		sc.Set("Bravo", SortDesc)
		assert.Equal(t, SortOff, sc.column(0))
		assert.Equal(t, SortDesc, sc.column(1))
	})
	t.Run("does nothing for an unknown label", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortAsc)
		sc.Set("Zulu", SortDesc)
		assert.Equal(t, SortAsc, sc.column(0))
		assert.Equal(t, SortOff, sc.column(1))
	})
	t.Run("sets the direction even for a column without a Sort func", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortAsc)
		sc.Set("Charlie", SortDesc)
		assert.Equal(t, SortDesc, sc.column(2))
	})
}

func TestColumnSorter_Current(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		Label: "Alpha",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}, {
		Label: "Bravo",
	}, {
		Label: "Charlie",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}})
	t.Run("return currently sorted column", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		sc.Set("Alpha", SortDesc)
		x, y := sc.current()
		assert.Equal(t, 0, x)
		assert.Equal(t, SortDesc, y)
	})
	t.Run("return currently sorted column 2", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		sc.Set("Charlie", SortDesc)
		x, y := sc.current()
		assert.Equal(t, 2, x)
		assert.Equal(t, SortDesc, y)
	})
	t.Run("return -1 if nothing set", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		x, y := sc.current()
		assert.Equal(t, -1, x)
		assert.Equal(t, SortOff, y)
	})
}

func TestDataCardRowWidget_Update(t *testing.T) {
	test.NewTempApp(t)
	t.Run("first row shows cell as bold label with highlight", func(t *testing.T) {
		// given
		w := newDataCardRowWidget(100)
		// when
		w.Update("Planet", RichTextSegmentsFromText("Earth"), true)
		// then
		assert.True(t, w.bg.Visible())
		assert.Equal(t, "Earth", w.label.String())
		require.Len(t, w.label.Segments, 1)
		assert.True(t, w.label.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold)
		assert.Empty(t, w.cell.Segments)
	})
	t.Run("other rows show label and trailing aligned cell without highlight", func(t *testing.T) {
		// given
		w := newDataCardRowWidget(100)
		// when
		w.Update("Planet", RichTextSegmentsFromText("Earth"), false)
		// then
		assert.False(t, w.bg.Visible())
		assert.Equal(t, "Planet", w.label.String())
		assert.Equal(t, "Earth", w.cell.String())
		require.Len(t, w.cell.Segments, 1)
		assert.Equal(t, fyne.TextAlignTrailing, w.cell.Segments[0].(*widget.TextSegment).Style.Alignment)
	})
	t.Run("can switch from first row to other row", func(t *testing.T) {
		// given
		w := newDataCardRowWidget(100)
		w.Update("Planet", RichTextSegmentsFromText("Earth"), true)
		// when
		w.Update("Planet", RichTextSegmentsFromText("Venus"), false)
		// then
		assert.False(t, w.bg.Visible())
		assert.Equal(t, "Planet", w.label.String())
		assert.Equal(t, "Venus", w.cell.String())
	})
	t.Run("can switch from other row to first row", func(t *testing.T) {
		// given
		w := newDataCardRowWidget(100)
		w.Update("Planet", RichTextSegmentsFromText("Earth"), false)
		// when
		w.Update("Planet", RichTextSegmentsFromText("Venus"), true)
		// then
		assert.True(t, w.bg.Visible())
		assert.Equal(t, "Venus", w.label.String())
	})
}

func TestDataCardWidget_Update(t *testing.T) {
	test.NewTempApp(t)
	type planetRow struct {
		id     int
		planet string
	}
	columns := NewDataColumns([]DataColumn[planetRow]{{
		Label: "ID",
	}, {
		Label: "Planet",
	}})
	t.Run("updates rows from cells without being rendered", func(t *testing.T) {
		// given
		var calls []string
		w := newDataCardWidget(columns, func(label string, r planetRow) []widget.RichTextSegment {
			calls = append(calls, label)
			switch label {
			case "ID":
				return RichTextSegmentsFromText(fmt.Sprint(r.id))
			case "Planet":
				return RichTextSegmentsFromText(r.planet)
			}
			return nil
		})
		// when
		w.Update(planetRow{id: 42, planet: "Earth"})
		// then
		assert.Equal(t, []string{"ID", "Planet"}, calls)
		require.Len(t, w.rows, 2)
		assert.True(t, w.rows[0].bg.Visible())
		assert.Equal(t, "42", w.rows[0].label.String())
		assert.False(t, w.rows[1].bg.Visible())
		assert.Equal(t, "Planet", w.rows[1].label.String())
		assert.Equal(t, "Earth", w.rows[1].cell.String())
	})
	t.Run("keeps row content when rendered after update", func(t *testing.T) {
		// given
		w := newDataCardWidget(columns, func(label string, r planetRow) []widget.RichTextSegment {
			return RichTextSegmentsFromText(r.planet)
		})
		w.Update(planetRow{id: 42, planet: "Earth"})
		// when
		test.WidgetRenderer(w)
		// then
		assert.Equal(t, "Earth", w.rows[1].cell.String())
	})
}

func TestMakeDataTable_Header(t *testing.T) {
	test.NewTempApp(t)
	type planetRow struct {
		id     int
		planet string
	}
	columns := NewDataColumns([]DataColumn[planetRow]{{
		Label:  "ID",
		Update: func(r planetRow, co fyne.CanvasObject) {},
	}, {
		Label:  "Planet",
		Sort:   func(a, b planetRow) int { return 0 },
		Update: func(r planetRow, co fyne.CanvasObject) {},
	}})
	t.Run("tapping a sortable header calls filterRows with its label", func(t *testing.T) {
		// given
		data := []planetRow{}
		var got []string
		x := MakeDataTable(
			columns,
			&data,
			func() fyne.CanvasObject { return widget.NewLabel("") },
			NewColumnSorter(columns, "Planet", SortOff),
			func(label string) { got = append(got, label) },
			nil,
		)
		h := x.CreateHeader().(*dataTableHeaderWidget)
		x.UpdateHeader(widget.TableCellID{Row: -1, Col: 1}, h)
		// when
		test.Tap(h.actionLabel)
		// then
		assert.Equal(t, []string{"Planet"}, got)
	})
	t.Run("unsortable header is a plain label", func(t *testing.T) {
		// given
		data := []planetRow{}
		x := MakeDataTable(
			columns,
			&data,
			func() fyne.CanvasObject { return widget.NewLabel("") },
			NewColumnSorter(columns, "Planet", SortAsc),
			func(string) {},
			nil,
		)
		h := x.CreateHeader().(*dataTableHeaderWidget)
		// when
		x.UpdateHeader(widget.TableCellID{Row: -1, Col: 0}, h)
		// then
		assert.True(t, h.label.Visible())
		assert.Equal(t, "ID", h.label.Text)
		assert.False(t, h.actionLabel.Visible())
		assert.False(t, h.icon.Visible())
	})
	cases := []struct {
		dir  SortDir
		want fyne.Resource
	}{
		{SortOff, theme.NewThemedResource(iconSortSvg)},
		{SortAsc, theme.NewPrimaryThemedResource(iconSortAscendingSvg)},
		{SortDesc, theme.NewPrimaryThemedResource(iconSortDescendingSvg)},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("icon shows sort state %s", tc.dir), func(t *testing.T) {
			// given
			data := []planetRow{}
			sc := NewColumnSorter(columns, "Planet", SortOff)
			x := MakeDataTable(
				columns,
				&data,
				func() fyne.CanvasObject { return widget.NewLabel("") },
				sc,
				func(string) {},
				nil,
			)
			h := x.CreateHeader().(*dataTableHeaderWidget)
			sc.Set("Planet", tc.dir)
			// when
			x.UpdateHeader(widget.TableCellID{Row: -1, Col: 1}, h)
			// then
			assert.True(t, h.icon.Visible())
			assert.Equal(t, tc.want.Name(), h.icon.Resource.Name())
		})
	}
}

func TestDataTableHeaderWidget_Update(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can switch from sortable to unsortable", func(t *testing.T) {
		// given
		w := newDataTableHeaderWidget()
		w.Update("Planet", SortAsc, theme.NewThemedResource(iconSortSvg), func() {})
		// when
		w.Update("ID", sortNone, nil, nil)
		// then
		assert.True(t, w.label.Visible())
		assert.Equal(t, "ID", w.label.Text)
		assert.False(t, w.actionLabel.Visible())
		assert.False(t, w.icon.Visible())
	})
	t.Run("can switch from unsortable to sortable", func(t *testing.T) {
		// given
		w := newDataTableHeaderWidget()
		w.Update("ID", sortNone, nil, nil)
		var tapped bool
		// when
		w.Update("Planet", SortAsc, theme.NewThemedResource(iconSortSvg), func() { tapped = true })
		// then
		assert.False(t, w.label.Visible())
		assert.True(t, w.actionLabel.Visible())
		assert.Equal(t, "Planet", w.actionLabel.Text)
		assert.True(t, w.icon.Visible())
		test.Tap(w.actionLabel)
		assert.True(t, tapped)
	})
}

func TestMakeDataList(t *testing.T) {
	test.NewTempApp(t)
	type planetRow struct {
		id     int
		planet string
	}
	columns := NewDataColumns([]DataColumn[planetRow]{{
		Label: "Planet",
	}})
	makeCell := func(label string, r planetRow) []widget.RichTextSegment {
		return RichTextSegmentsFromText(r.planet)
	}
	t.Run("reports selected row", func(t *testing.T) {
		// given
		data := []planetRow{{3, "Mercury"}, {8, "Venus"}}
		var got planetRow
		l := MakeDataList(columns, &data, makeCell, func(r planetRow) { got = r })
		// when
		l.Select(1)
		// then
		assert.Equal(t, planetRow{8, "Venus"}, got)
	})
	t.Run("same row can be selected again", func(t *testing.T) {
		// given
		data := []planetRow{{3, "Mercury"}, {8, "Venus"}}
		var calls int
		l := MakeDataList(columns, &data, makeCell, func(planetRow) { calls++ })
		l.Select(1)
		// when
		l.Select(1)
		// then
		assert.Equal(t, 2, calls)
	})
	t.Run("ignores rows no longer in data", func(t *testing.T) {
		// given
		data := []planetRow{{3, "Mercury"}}
		var calls int
		l := MakeDataList(columns, &data, makeCell, func(planetRow) { calls++ })
		// when
		l.OnSelected(5)
		// then
		assert.Equal(t, 0, calls)
	})
	t.Run("does not panic without callback", func(t *testing.T) {
		// given
		data := []planetRow{{3, "Mercury"}}
		l := MakeDataList(columns, &data, makeCell, nil)
		// when/then
		assert.NotPanics(t, func() {
			l.Select(0)
		})
	})
	t.Run("updates card with row at position", func(t *testing.T) {
		// given
		data := []planetRow{{3, "Mercury"}, {8, "Venus"}}
		l := MakeDataList(columns, &data, makeCell, nil)
		co := l.CreateItem()
		// when
		l.UpdateItem(1, co)
		// then
		assert.Equal(t, "Venus", co.(*dataCardWidget[planetRow]).rows[0].label.String())
	})
}
