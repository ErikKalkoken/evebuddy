package ui

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type myRow struct {
	id     int
	planet string
}

func TestDataTable_CreateBasic(t *testing.T) {
	test.NewTempApp(t)
	headers := []headerDef{
		{Label: "ID", Width: 100},
		{Label: "Planet", Width: 100},
	}
	data := []myRow{{3, "Mercury"}, {8, "Venus"}, {42, "Earth"}}
	x := makeDataTable(
		headers, &data, func(col int, r myRow) []widget.RichTextSegment {
			switch col {
			case 0:
				return iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.id))
			case 1:
				return iwidget.NewRichTextSegmentFromText(r.planet)
			}
			panic(fmt.Sprintf("invalid col: %d", col))
		},
		newColumnSorterWithInit(headers, 0, sortAsc),
		func(i int) {

		},
		nil,
	)
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSquareSize(300))

	test.AssertImageMatches(t, "datatable/basic.png", w.Canvas().Capture())
}
