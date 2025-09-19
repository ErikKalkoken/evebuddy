package widget_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/stretchr/testify/assert"
)

type myRow struct {
	id     int
	planet string
}

func TestDataTable_CreateBasic(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{{
		Col:   0,
		Label: "ID",
		Width: 100,
	}, {
		Col:   1,
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
		headers.NewColumnSorter(0, iwidget.SortAsc),
		func(i int) {

		},
		nil,
	)
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSquareSize(300))

	test.AssertImageMatches(t, "datatable/basic.png", w.Canvas().Capture())
}

func TestDataTableDef_New(t *testing.T) {
	t.Run("can define column", func(t *testing.T) {
		def := iwidget.NewDataTableDef([]iwidget.ColumnDef{{
			Col:   0,
			Label: "Alpha",
		}})
		assert.Equal(t, "Alpha", def.Column(0).Label)
	})
	t.Run("should panic when label not defined", func(t *testing.T) {
		assert.Panics(t, func() {
			iwidget.NewDataTableDef([]iwidget.ColumnDef{{
				Col: 0,
			}})
		})
	})
	t.Run("should panic when col index too small", func(t *testing.T) {
		assert.Panics(t, func() {
			iwidget.NewDataTableDef([]iwidget.ColumnDef{{
				Col:   -1,
				Label: "Alpha",
			}})
		})
	})
	t.Run("should panic when col index too large", func(t *testing.T) {
		assert.Panics(t, func() {
			iwidget.NewDataTableDef([]iwidget.ColumnDef{{
				Col:   1,
				Label: "Alpha",
			}})
		})
	})
	t.Run("should panic when exected col index is not defined", func(t *testing.T) {
		assert.Panics(t, func() {
			iwidget.NewDataTableDef([]iwidget.ColumnDef{{
				Col:   0,
				Label: "Alpha",
			}, {
				Col:   2,
				Label: "Bravo",
			}})
		})
	})
	t.Run("should panic when col index is defined more then once", func(t *testing.T) {
		assert.Panics(t, func() {
			iwidget.NewDataTableDef([]iwidget.ColumnDef{{
				Col:   0,
				Label: "Alpha",
			}, {
				Col:   0,
				Label: "Bravo",
			}})
		})
	})
}
